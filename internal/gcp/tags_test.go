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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
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

type notFoundTagKeysServer struct {
	resourcemanagerpb.UnimplementedTagKeysServer
	createCalled bool
}

func (s *notFoundTagKeysServer) GetNamespacedTagKey(ctx context.Context, req *resourcemanagerpb.GetNamespacedTagKeyRequest) (*resourcemanagerpb.TagKey, error) {
	return nil, status.Error(codes.NotFound, "tag key not found")
}

func (s *notFoundTagKeysServer) CreateTagKey(ctx context.Context, req *resourcemanagerpb.CreateTagKeyRequest) (*resourcemanagerpb.Operation, error) {
	s.createCalled = true
	return nil, status.Error(codes.Internal, "create not implemented in test")
}

type permissionDeniedTagKeysServer struct {
	resourcemanagerpb.UnimplementedTagKeysServer
	createCalled bool
}

func (s *permissionDeniedTagKeysServer) GetNamespacedTagKey(ctx context.Context, req *resourcemanagerpb.GetNamespacedTagKeyRequest) (*resourcemanagerpb.TagKey, error) {
	return nil, status.Error(codes.PermissionDenied, "permission denied")
}

func (s *permissionDeniedTagKeysServer) CreateTagKey(ctx context.Context, req *resourcemanagerpb.CreateTagKeyRequest) (*resourcemanagerpb.Operation, error) {
	s.createCalled = true
	return nil, status.Error(codes.Internal, "create not implemented in test")
}

type notFoundTagValuesServer struct {
	resourcemanagerpb.UnimplementedTagValuesServer
	createCalled bool
}

func (s *notFoundTagValuesServer) GetNamespacedTagValue(ctx context.Context, req *resourcemanagerpb.GetNamespacedTagValueRequest) (*resourcemanagerpb.TagValue, error) {
	return nil, status.Error(codes.NotFound, "tag value not found")
}

func (s *notFoundTagValuesServer) CreateTagValue(ctx context.Context, req *resourcemanagerpb.CreateTagValueRequest) (*resourcemanagerpb.Operation, error) {
	s.createCalled = true
	return nil, status.Error(codes.Internal, "create not implemented in test")
}

type permissionDeniedTagValuesServer struct {
	resourcemanagerpb.UnimplementedTagValuesServer
	createCalled bool
}

func (s *permissionDeniedTagValuesServer) GetNamespacedTagValue(ctx context.Context, req *resourcemanagerpb.GetNamespacedTagValueRequest) (*resourcemanagerpb.TagValue, error) {
	return nil, status.Error(codes.PermissionDenied, "permission denied")
}

func (s *permissionDeniedTagValuesServer) CreateTagValue(ctx context.Context, req *resourcemanagerpb.CreateTagValueRequest) (*resourcemanagerpb.Operation, error) {
	s.createCalled = true
	return nil, status.Error(codes.Internal, "create not implemented in test")
}

// TestLookupKey_NotFoundTriggersCreate verifies that a NotFound error causes auto-creation.
func TestLookupKey_NotFoundTriggersCreate(t *testing.T) {
	lis := bufconn.Listen(bufSize)
	srv := &notFoundTagKeysServer{}
	s := grpc.NewServer()
	resourcemanagerpb.RegisterTagKeysServer(s, srv)
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
	assert.NoError(t, err)
	defer conn.Close()

	keysClient, err := resourcemanager.NewTagKeysClient(ctx, option.WithGRPCConn(conn))
	assert.NoError(t, err)

	mgr := NewTagsManager(keysClient, nil, nil)
	// CreateTagKey will fail with Internal — but the important thing is that it was attempted (not silently failed)
	_, err = mgr.LookupKey(ctx, "proj", "new-key")
	assert.Error(t, err, "expected an error because CreateTagKey is not fully implemented in the test server")
	assert.True(t, srv.createCalled, "CreateTagKey should have been called when NotFound is returned")
}

// TestLookupKey_PermissionDeniedDoesNotCreate verifies that PermissionDenied does NOT trigger auto-creation.
func TestLookupKey_PermissionDeniedDoesNotCreate(t *testing.T) {
	lis := bufconn.Listen(bufSize)
	srv := &permissionDeniedTagKeysServer{}
	s := grpc.NewServer()
	resourcemanagerpb.RegisterTagKeysServer(s, srv)
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
	assert.NoError(t, err)
	defer conn.Close()

	keysClient, err := resourcemanager.NewTagKeysClient(ctx, option.WithGRPCConn(conn))
	assert.NoError(t, err)

	mgr := NewTagsManager(keysClient, nil, nil)
	_, err = mgr.LookupKey(ctx, "proj", "some-key")
	assert.Error(t, err, "expected an error on PermissionDenied")
	assert.False(t, srv.createCalled, "CreateTagKey must NOT be called when PermissionDenied is returned")
}

// TestLookupValue_NotFoundTriggersCreate verifies that a NotFound error causes auto-creation.
func TestLookupValue_NotFoundTriggersCreate(t *testing.T) {
	lis := bufconn.Listen(bufSize)
	keysSrv := &fakeTagKeysServer{}
	valuesSrv := &notFoundTagValuesServer{}
	s := grpc.NewServer()
	resourcemanagerpb.RegisterTagKeysServer(s, keysSrv)
	resourcemanagerpb.RegisterTagValuesServer(s, valuesSrv)
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
	assert.NoError(t, err)
	defer conn.Close()

	keysClient, err := resourcemanager.NewTagKeysClient(ctx, option.WithGRPCConn(conn))
	assert.NoError(t, err)
	valuesClient, err := resourcemanager.NewTagValuesClient(ctx, option.WithGRPCConn(conn))
	assert.NoError(t, err)

	mgr := NewTagsManager(keysClient, valuesClient, nil)
	_, err = mgr.LookupValue(ctx, "test-project", "existing-key", "new-value")
	assert.Error(t, err, "expected an error because CreateTagValue is not fully implemented in the test server")
	assert.True(t, valuesSrv.createCalled, "CreateTagValue should have been called when NotFound is returned")
}

// TestLookupValue_PermissionDeniedDoesNotCreate verifies that PermissionDenied does NOT trigger auto-creation.
func TestLookupValue_PermissionDeniedDoesNotCreate(t *testing.T) {
	lis := bufconn.Listen(bufSize)
	srv := &permissionDeniedTagValuesServer{}
	s := grpc.NewServer()
	resourcemanagerpb.RegisterTagValuesServer(s, srv)
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
	assert.NoError(t, err)
	defer conn.Close()

	valuesClient, err := resourcemanager.NewTagValuesClient(ctx, option.WithGRPCConn(conn))
	assert.NoError(t, err)

	mgr := NewTagsManager(nil, valuesClient, nil)
	_, err = mgr.LookupValue(ctx, "proj", "some-key", "some-value")
	assert.Error(t, err, "expected an error on PermissionDenied")
	assert.False(t, srv.createCalled, "CreateTagValue must NOT be called when PermissionDenied is returned")
}

// TestLookupKeyNoCreate_NotFoundReturnsNil verifies that NotFound returns nil without creating.
func TestLookupKeyNoCreate_NotFoundReturnsNil(t *testing.T) {
	lis := bufconn.Listen(bufSize)
	srv := &notFoundTagKeysServer{}
	s := grpc.NewServer()
	resourcemanagerpb.RegisterTagKeysServer(s, srv)
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
	assert.NoError(t, err)
	defer conn.Close()

	keysClient, err := resourcemanager.NewTagKeysClient(ctx, option.WithGRPCConn(conn))
	assert.NoError(t, err)

	mgr := NewTagsManager(keysClient, nil, nil)
	key, err := mgr.LookupKeyNoCreate(ctx, "proj", "missing-key")
	assert.NoError(t, err, "NotFound should not be an error for LookupKeyNoCreate")
	assert.Nil(t, key, "expected nil key when not found")
	assert.False(t, srv.createCalled, "CreateTagKey must NOT be called by LookupKeyNoCreate")
}

// TestLookupValueNoCreate_NotFoundReturnsNil verifies that NotFound returns nil without creating.
func TestLookupValueNoCreate_NotFoundReturnsNil(t *testing.T) {
	lis := bufconn.Listen(bufSize)
	valuesSrv := &notFoundTagValuesServer{}
	s := grpc.NewServer()
	resourcemanagerpb.RegisterTagValuesServer(s, valuesSrv)
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
	assert.NoError(t, err)
	defer conn.Close()

	valuesClient, err := resourcemanager.NewTagValuesClient(ctx, option.WithGRPCConn(conn))
	assert.NoError(t, err)

	mgr := NewTagsManager(nil, valuesClient, nil)
	val, err := mgr.LookupValueNoCreate(ctx, "proj", "some-key", "missing-value")
	assert.NoError(t, err, "NotFound should not be an error for LookupValueNoCreate")
	assert.Nil(t, val, "expected nil value when not found")
	assert.False(t, valuesSrv.createCalled, "CreateTagValue must NOT be called by LookupValueNoCreate")
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
		name      string
		projectID string
		key       string
		want      string
	}{
		{
			name:      "simple key",
			projectID: "my-project",
			key:       "my-key",
			want:      "key:my-project:my-key",
		},
		{
			name:      "empty key",
			projectID: "my-project",
			key:       "",
			want:      "key:my-project:",
		},
		{
			name:      "different projects produce different cache keys",
			projectID: "other-project",
			key:       "my-key",
			want:      "key:other-project:my-key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := cacheKeyTagKey(tc.projectID, tc.key)
			assert.Equal(t, tc.want, got, fmt.Sprintf("cacheKeyTagKey(%q, %q) should return %q", tc.projectID, tc.key, tc.want))
		})
	}
}

func TestCacheKeyTagValue(t *testing.T) {
	testCases := []struct {
		name      string
		projectID string
		key       string
		value     string
		want      string
	}{
		{
			name:      "simple key and value",
			projectID: "my-project",
			key:       "my-key",
			value:     "my-value",
			want:      "value:my-project:my-key:my-value",
		},
		{
			name:      "empty key",
			projectID: "my-project",
			key:       "",
			value:     "my-value",
			want:      "value:my-project::my-value",
		},
		{
			name:      "empty value",
			projectID: "my-project",
			key:       "my-key",
			value:     "",
			want:      "value:my-project:my-key:",
		},
		{
			name:      "different projects produce different cache keys",
			projectID: "other-project",
			key:       "my-key",
			value:     "my-value",
			want:      "value:other-project:my-key:my-value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := cacheKeyTagValue(tc.projectID, tc.key, tc.value)
			assert.Equal(t, tc.want, got, fmt.Sprintf("cacheKeyTagValue(%q, %q, %q) should return %q", tc.projectID, tc.key, tc.value, tc.want))
		})
	}
}
