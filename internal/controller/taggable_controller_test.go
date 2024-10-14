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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tagsv1alpha1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/tags/v1alpha1"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Taggable Resource Controller", func() {
	Context("When reconciling a resource", func() {

		It("should successfully reconcile the resource", func() {
			// TODO: Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})

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
			tt := tt // pin variable to avoid parallel test issues
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
			{
				name:     "long name",
				owner:    &MockObject{ObjectMeta: metav1.ObjectMeta{Name: "very-long-owner-name-that-will-be-truncated"}},
				valueRef: "tagValues/12345",
				want:     "testkind-very-long-owner-name-that-will-be-truncated-12345",
			},
		}

		for _, tt := range tests {
			tt := tt // pin variable to avoid parallel test issues
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
			tt := tt // pin variable to avoid parallel test issues
			It("should return the correct change status for "+tt.name, func() {
				got := tagBindingChanged(tt.expected, tt.actual)
				Expect(got).To(Equal(tt.want))
			})
		}
	})
})

// MockObject implementation remains the same
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
