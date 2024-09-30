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

	"k8s.io/utils/ptr"

	sqlv1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/sql/v1beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSQLInstanceMetadataProvider_GetResourceID(t *testing.T) {
	t.Parallel()

	typeMeta := metav1.TypeMeta{
		Kind:       "SQLInstance",
		APIVersion: "sql.cnrm.cloud.google.com/v1beta1",
	}

	testCases := []struct {
		name      string
		r         *sqlv1beta1.SQLInstance
		projectID string
		want      string
	}{
		{
			name: "with generated name",
			r: &sqlv1beta1.SQLInstance{
				TypeMeta: typeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-instance",
				},
				Spec: sqlv1beta1.SQLInstanceSpec{
					Region: ptr.To("us-central1"),
				},
			},
			projectID: "test-project",
			want:      "//sqladmin.googleapis.com/projects/test-project/instances/test-instance",
		},
		{
			name: "with overridden resource id",
			r: &sqlv1beta1.SQLInstance{
				TypeMeta: typeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-instance",
				},
				Spec: sqlv1beta1.SQLInstanceSpec{
					Region:     ptr.To("us-central1"),
					ResourceID: ptr.To("overridden-instance-id"),
				},
			},
			projectID: "test-project",
			want:      "//sqladmin.googleapis.com/projects/test-project/instances/overridden-instance-id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := &SQLInstanceMetadataProvider{}

			got := p.GetResourceID(tc.projectID, tc.r)

			require.Equal(t, tc.want, got)
		})
	}
}

func TestSQLInstanceMetadataProvider_GetResourceLocation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		r    *sqlv1beta1.SQLInstance
		want string
	}{
		{
			name: "default",
			r: &sqlv1beta1.SQLInstance{
				Spec: sqlv1beta1.SQLInstanceSpec{
					Region: ptr.To("us-central1"),
				},
			},
			want: "us-central1",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := &SQLInstanceMetadataProvider{}
			got := p.GetResourceLocation(tc.r)
			require.Equal(t, tc.want, got)
		})
	}
}
