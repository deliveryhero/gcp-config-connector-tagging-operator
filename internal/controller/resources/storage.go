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
	"fmt"

	storagev1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/storage/v1beta1"
	"k8s.io/utils/ptr"

	"github.com/deliveryhero/gcp-config-connector-tagging-operator/internal/controller"
)

// +kubebuilder:rbac:groups=storage.cnrm.cloud.google.com,resources=storagebuckets,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=storage.cnrm.cloud.google.com,resources=storagebuckets/finalizers,verbs=update

var _ controller.ResourceMetadataProvider[storagev1beta1.StorageBucket] = &StorageBucketMetadataProvider{}

type StorageBucketMetadataProvider struct{}

func (in *StorageBucketMetadataProvider) GetResourceLocation(r *storagev1beta1.StorageBucket) string {
	return ptr.Deref(r.Spec.Location, "")
}

func (in *StorageBucketMetadataProvider) GetResourceID(_ string, r *storagev1beta1.StorageBucket) string {
	name := r.Name
	if r.Spec.ResourceID != nil {
		name = *r.Spec.ResourceID
	}

	return fmt.Sprintf("//storage.googleapis.com/projects/_/buckets/%s", name)
}
