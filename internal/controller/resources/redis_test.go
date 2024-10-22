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

package resources

import (
	"testing"

	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	redisv1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/redis/v1beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestRedisInstanceMetadataProvider_GetResourceID(t *testing.T) {
	t.Parallel()

	typeMeta := metav1.TypeMeta{
		Kind:       "RedisInstance",
		APIVersion: "redis.cnrm.cloud.google.com/v1beta1",
	}

	testCases := []struct {
		name        string
		r           *redisv1beta1.RedisInstance
		projectInfo *resourcemanagerpb.Project
		want        string
	}{
		{
			name: "with generated name",
			r: &redisv1beta1.RedisInstance{
				TypeMeta: typeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-instance",
				},
				Spec: redisv1beta1.RedisInstanceSpec{
					Region: "us-central1",
				},
			},
			projectInfo: &resourcemanagerpb.Project{
				Name:      "projects/123456789", // Example project name with number
				ProjectId: "test-project",
			},
			want: "//redis.googleapis.com/projects/123456789/locations/us-central1/instances/test-instance",
		},
		{
			name: "with overridden resource id",
			r: &redisv1beta1.RedisInstance{
				TypeMeta: typeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-instance",
				},
				Spec: redisv1beta1.RedisInstanceSpec{
					Region:     "us-central1",
					ResourceID: ptr.To("overridden-instance-id"),
				},
			},
			projectInfo: &resourcemanagerpb.Project{
				Name:      "projects/987654321", // Example project name with number
				ProjectId: "test-project",
			},
			want: "//redis.googleapis.com/projects/987654321/locations/us-central1/instances/overridden-instance-id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := &RedisInstanceMetadataProvider{}

			got := p.GetResourceID(tc.projectInfo, tc.r)

			require.Equal(t, tc.want, got)
		})
	}
}

func TestRedisInstanceMetadataProvider_GetResourceLocation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		r    *redisv1beta1.RedisInstance
		want string
	}{
		{
			name: "default",
			r: &redisv1beta1.RedisInstance{
				Spec: redisv1beta1.RedisInstanceSpec{
					Region: "us-central1",
				},
			},
			want: "us-central1",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := &RedisInstanceMetadataProvider{}
			got := p.GetResourceLocation(tc.r)
			require.Equal(t, tc.want, got)
		})
	}
}
