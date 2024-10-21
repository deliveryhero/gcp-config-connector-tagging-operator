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
	kmsv1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/kms/v1beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestKMSKeyRingMetadataProvider_GetResourceID(t *testing.T) {
	t.Parallel()

	typeMeta := metav1.TypeMeta{
		Kind:       "KMSKeyRing",
		APIVersion: "kms.cnrm.cloud.google.com/v1beta1",
	}

	testCases := []struct {
		name        string
		r           *kmsv1beta1.KMSKeyRing
		projectInfo *resourcemanagerpb.Project
		want        string
	}{
		{
			name: "with generated name",
			r: &kmsv1beta1.KMSKeyRing{
				TypeMeta: typeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-key-ring",
				},
				Spec: kmsv1beta1.KMSKeyRingSpec{
					Location: "us-central1",
				},
			},
			projectInfo: &resourcemanagerpb.Project{
				ProjectId: "test-project",
			},
			want: "//cloudkms.googleapis.com/projects/test-project/locations/us-central1/keyRings/test-key-ring",
		},
		{
			name: "with overridden resource id",
			r: &kmsv1beta1.KMSKeyRing{
				TypeMeta: typeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-key-ring",
				},
				Spec: kmsv1beta1.KMSKeyRingSpec{
					Location:   "us-central1",
					ResourceID: ptr.To("overridden-key-ring-id"),
				},
			},
			projectInfo: &resourcemanagerpb.Project{
				ProjectId: "test-project",
			},
			want: "//cloudkms.googleapis.com/projects/test-project/locations/us-central1/keyRings/overridden-key-ring-id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := &KMSKeyRingMetadataProvider{}

			got := p.GetResourceID(tc.projectInfo, tc.r)

			require.Equal(t, tc.want, got)
		})
	}
}

func TestKMSKeyRingMetadataProvider_GetResourceLocation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		r    *kmsv1beta1.KMSKeyRing
		want string
	}{
		{
			name: "default",
			r: &kmsv1beta1.KMSKeyRing{
				Spec: kmsv1beta1.KMSKeyRingSpec{
					Location: "us-central1",
				},
			},
			want: "us-central1",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := &KMSKeyRingMetadataProvider{}
			got := p.GetResourceLocation(tc.r)
			require.Equal(t, tc.want, got)
		})
	}
}
