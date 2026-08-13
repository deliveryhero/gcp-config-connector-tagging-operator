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

package controller

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	tagsv1alpha1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/tags/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Taggable Resource Controller", func() {
	Describe("OwnerIndexValue function", func() {
		tests := []struct {
			name       string
			apiVersion string
			kind       string
			want       string
			wantErr    bool
		}{
			{
				apiVersion: "storage.cnrm.cloud.google.com/v1beta1",
				kind:       "StorageBucket",
				name:       "test-bucket",
				want:       "storage.cnrm.cloud.google.com/v1beta1/StorageBucket/test-bucket",
				wantErr:    false,
			},
			{
				apiVersion: "",
				kind:       "StorageBucket",
				name:       "test-bucket",
				want:       "/StorageBucket/test-bucket",
				wantErr:    false,
			},
			{
				apiVersion: "storage.cnrm.cloud.google.com/v1beta1",
				kind:       "",
				name:       "test-bucket",
				want:       "storage.cnrm.cloud.google.com/v1beta1//test-bucket",
				wantErr:    false,
			},
			{
				apiVersion: "storage.cnrm.cloud.google.com/v1beta1",
				kind:       "StorageBucket",
				name:       "",
				want:       "storage.cnrm.cloud.google.com/v1beta1/StorageBucket/",
				wantErr:    false,
			},
		}

		for _, tt := range tests {
			tt := tt
			It("should return the correct owner index value for "+tt.name, func() {
				got := ownerIndexValue(tt.apiVersion, tt.kind, tt.name)
				Expect(got).To(Equal(tt.want))
			})
		}
	})

	Describe("TagBindingResourceName function", func() {
		tests := []struct {
			name     string
			owner    client.Object
			valueRef string
			want     string
		}{
			{
				name:     "valid input",
				owner:    &MockObject{ObjectMeta: metav1.ObjectMeta{Name: "test-bucket"}},
				valueRef: "tagValues/12345",
				want:     "testkind-test-bucket-12345",
			},
		}

		for _, tt := range tests {
			tt := tt
			It("should return the correct resource name for "+tt.name, func() {
				got := tagBindingResourceName(tt.owner, tt.valueRef)
				Expect(got).To(Equal(tt.want))
			})
		}
	})

	Describe("TagBindingChanged function", func() {
		tests := []struct {
			name     string
			expected *tagsv1alpha1.TagsLocationTagBinding
			actual   *tagsv1alpha1.TagsLocationTagBinding
			want     bool
		}{
			{
				name: "different tag value ref",
				expected: &tagsv1alpha1.TagsLocationTagBinding{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"cnrm.cloud.google.com/project-id":         "test-project-1",
							"tags.cnrm.cloud.google.com/tag-value-ref": "tagValues/12345",
						},
					},
				},
				actual: &tagsv1alpha1.TagsLocationTagBinding{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"cnrm.cloud.google.com/project-id":         "test-project",
							"tags.cnrm.cloud.google.com/tag-value-ref": "tagValues/67890",
						},
					},
				},
				want: true,
			},
		}

		for _, tt := range tests {
			tt := tt
			It("should return the correct change status for "+tt.name, func() {
				got := tagBindingChanged(tt.expected, tt.actual)
				Expect(got).To(Equal(tt.want))
			})
		}
	})
})

func TestDetermineProjectID(t *testing.T) {
	tests := []struct {
		name                string
		resourceAnnotations map[string]string
		nsAnnotations       map[string]string
		resourceNamespace   string
		wantProjectID       string
		wantErr             bool
	}{
		{
			name:                "project ID from resource annotation",
			resourceAnnotations: map[string]string{projectIDAnnotation: "resource-project"},
			resourceNamespace:   "my-ns",
			wantProjectID:       "resource-project",
		},
		{
			name:                "project ID from namespace annotation",
			resourceAnnotations: map[string]string{},
			nsAnnotations:       map[string]string{projectIDAnnotation: "ns-project"},
			resourceNamespace:   "my-ns",
			wantProjectID:       "ns-project",
		},
		{
			name:                "no project ID annotation returns error",
			resourceAnnotations: map[string]string{},
			nsAnnotations:       map[string]string{},
			resourceNamespace:   "my-ns",
			wantErr:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:        tt.resourceNamespace,
					Annotations: tt.nsAnnotations,
				},
			}
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(ns).
				Build()

			resource := &MockObject{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-resource",
					Namespace:   tt.resourceNamespace,
					Annotations: tt.resourceAnnotations,
				},
			}

			reconciler := &TaggableResourceReconciler[MockObject, *mockMetadataProvider, *MockObject]{
				Client: fakeClient,
			}

			projectID, err := reconciler.determineProjectID(context.Background(), resource)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantProjectID, projectID)
			}
		})
	}
}

type mockMetadataProvider struct{}

func (m *mockMetadataProvider) GetResourceLocation(r *MockObject) string { return "" }
func (m *mockMetadataProvider) GetResourceID(projectInfo *resourcemanagerpb.Project, r *MockObject) string {
	return ""
}

type MockObject struct {
	mock.Mock
	metav1.ObjectMeta
}

func (m *MockObject) DeepCopyObject() runtime.Object {
	if c := m.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (m *MockObject) DeepCopyInto(out *MockObject) {
	out.Mock = mock.Mock{}
	out.ObjectMeta = m.ObjectMeta
}

func (m *MockObject) DeepCopy() *MockObject {
	if m == nil {
		return nil
	}
	out := new(MockObject)
	m.DeepCopyInto(out)
	return out
}

func (m *MockObject) GetObjectKind() schema.ObjectKind {
	return &metav1.TypeMeta{
		APIVersion: "testgroup/v1",
		Kind:       "TestKind",
	}
}
