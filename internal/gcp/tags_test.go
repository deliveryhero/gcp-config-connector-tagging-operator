/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gcp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func bufDialer(lis *bufconn.Listener) (net.Conn, error) {
	return lis.Dial()
}

type fakeTagKeysServer struct {
	resourcemanagerpb.UnimplementedTagKeysServer
}

func (s *fakeTagKeysServer) GetNamespacedTagKey(ctx context.Context, req *resourcemanagerpb.GetNamespacedTagKeyRequest) (*resourcemanagerpb.TagKey, error) {
	if req.Name == "test-project/existing-key" {
		return &resourcemanagerpb.TagKey{
			Name:      "projects/test-project/existing-key",
			ShortName: "existing-key",
		}, nil
	}
	return nil, fmt.Errorf("tag key not found")
}

type fakeTagValuesServer struct {
	resourcemanagerpb.UnimplementedTagValuesServer
}

func (s *fakeTagValuesServer) GetNamespacedTagValue(ctx context.Context, req *resourcemanagerpb.GetNamespacedTagValueRequest) (*resourcemanagerpb.TagValue, error) {
	if req.Name == "test-project/existing-key/existing-value" {
		return &resourcemanagerpb.TagValue{
			Name:      "projects/test-project/existing-key/existing-value",
			ShortName: "existing-value",
		}, nil
	}
	return nil, fmt.Errorf("tag value not found")
}

func TestLookupKeyWithFakeGRPCServer(t *testing.T) {
	lis := bufconn.Listen(bufSize)

	s := grpc.NewServer()
	resourcemanagerpb.RegisterTagKeysServer(s, &fakeTagKeysServer{})

	go func() {
		if err := s.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			t.Errorf("Server exited with error: %v", err)
		}
	}()
	defer s.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return bufDialer(lis)
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err, "Failed to dial bufnet")
	defer conn.Close()

	keysClient, err := resourcemanager.NewTagKeysClient(ctx, option.WithGRPCConn(conn))
	assert.NoError(t, err, "Failed to create TagKeysClient")

	mgr := NewTagsManager(keysClient, nil, nil)

	key, err := mgr.LookupKey(ctx, "test-project", "existing-key")
	assert.NoError(t, err, "LookupKey failed")
	assert.Equal(t, "projects/test-project/existing-key", key.Name, "Expected key name 'projects/test-project/existing-key'")
}

func TestLookupValueWithFakeGRPCServer(t *testing.T) {
	lis := bufconn.Listen(bufSize)

	s := grpc.NewServer()
	resourcemanagerpb.RegisterTagKeysServer(s, &fakeTagKeysServer{})
	resourcemanagerpb.RegisterTagValuesServer(s, &fakeTagValuesServer{})

	var wg sync.WaitGroup
	wg.Add(1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		defer wg.Done()
		if err := s.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			t.Errorf("Server exited with error: %v", err)
		}
	}()

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return bufDialer(lis)
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err, "Failed to dial bufnet")
	defer conn.Close()

	valuesClient, err := resourcemanager.NewTagValuesClient(ctx, option.WithGRPCConn(conn))
	assert.NoError(t, err, "Failed to create TagValuesClient")

	mgr := NewTagsManager(nil, valuesClient, nil)

	value, err := mgr.LookupValue(ctx, "test-project", "existing-key", "existing-value")
	assert.NoError(t, err, "LookupValue failed")
	assert.Equal(t, "projects/test-project/existing-key/existing-value", value.Name, "Expected value name 'projects/test-project/existing-key/existing-value'")

	cancel()

	s.Stop()

	lis.Close()
}

func TestCacheKeyTagKey(t *testing.T) {
	testCases := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "simple key",
			key:  "my-key",
			want: "key:my-key",
		},
		{
			name: "empty key",
			key:  "",
			want: "key:",
		},
		{
			name: "key with special characters",
			key:  "key-with-special_chars",
			want: "key:key-with-special_chars",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := cacheKeyTagKey(tc.key)
			assert.Equal(t, tc.want, got, fmt.Sprintf("cacheKeyTagKey(%q) should return %q", tc.key, tc.want))
		})
	}
}

func TestCacheKeyTagValue(t *testing.T) {
	testCases := []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{
			name:  "simple key and value",
			key:   "my-key",
			value: "my-value",
			want:  "value:my-key:my-value",
		},
		{
			name:  "empty key",
			key:   "",
			value: "my-value",
			want:  "value::my-value",
		},
		{
			name:  "empty value",
			key:   "my-key",
			value: "",
			want:  "value:my-key:",
		},
		{
			name:  "key and value with special characters",
			key:   "key-with-special_chars",
			value: "value-with-special_chars",
			want:  "value:key-with-special_chars:value-with-special_chars",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := cacheKeyTagValue(tc.key, tc.value)
			assert.Equal(t, tc.want, got, fmt.Sprintf("cacheKeyTagValue(%q, %q) should return %q", tc.key, tc.value, tc.want))
		})
	}
}
