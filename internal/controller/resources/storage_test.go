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
	storagev1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/storage/v1beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestStorageBucketMetadataProvider_GetResourceID(t *testing.T) {
	t.Parallel()

	typeMeta := metav1.TypeMeta{
		Kind:       "StorageBucket",
		APIVersion: "storage.cnrm.cloud.google.com/v1beta1",
	}

	testCases := []struct {
		name        string
		r           *storagev1beta1.StorageBucket
		projectInfo *resourcemanagerpb.Project
		want        string
	}{
		{
			name: "with generated name",
			r: &storagev1beta1.StorageBucket{
				TypeMeta: typeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-bucket",
				},
				Spec: storagev1beta1.StorageBucketSpec{
					Location: ptr.To("us-central1"),
				},
			},
			projectInfo: &resourcemanagerpb.Project{
				ProjectId: "test-project",
			},
			want: "//storage.googleapis.com/projects/_/buckets/test-bucket",
		},
		{
			name: "with overridden resource id",
			r: &storagev1beta1.StorageBucket{
				TypeMeta: typeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-bucket",
				},
				Spec: storagev1beta1.StorageBucketSpec{
					Location:   ptr.To("us-central1"),
					ResourceID: ptr.To("overridden-bucket-id"),
				},
			},
			projectInfo: &resourcemanagerpb.Project{
				ProjectId: "test-project",
			},
			want: "//storage.googleapis.com/projects/_/buckets/overridden-bucket-id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := &StorageBucketMetadataProvider{}

			got := p.GetResourceID(tc.projectInfo, tc.r)

			require.Equal(t, tc.want, got)
		})
	}
}

func TestStorageBucketMetadataProvider_GetResourceLocation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		r    *storagev1beta1.StorageBucket
		want string
	}{
		{
			name: "default",
			r: &storagev1beta1.StorageBucket{
				Spec: storagev1beta1.StorageBucketSpec{
					Location: ptr.To("us-central1"),
				},
			},
			want: "us-central1",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := &StorageBucketMetadataProvider{}
			got := p.GetResourceLocation(tc.r)
			require.Equal(t, tc.want, got)
		})
	}
}
