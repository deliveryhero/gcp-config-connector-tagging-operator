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

	kmsv1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/kms/v1beta1"

	"github.com/deliveryhero/gcp-config-connector-tagging-operator/internal/controller"
)

// +kubebuilder:rbac:groups=kms.cnrm.cloud.google.com,resources=kmskeyrings,verbs=get;list;watch

var _ controller.ResourceMetadataProvider[kmsv1beta1.KMSKeyRing] = &KMSKeyRingMetadataProvider{}

type KMSKeyRingMetadataProvider struct{}

func (in *KMSKeyRingMetadataProvider) GetResourceLocation(r *kmsv1beta1.KMSKeyRing) string {
	return r.Spec.Location
}

func (in *KMSKeyRingMetadataProvider) GetResourceID(projectID string, r *kmsv1beta1.KMSKeyRing) string {
	name := r.Name
	if r.Spec.ResourceID != nil {
		name = *r.Spec.ResourceID
	}

	return fmt.Sprintf("//cloudkms.googleapis.com/projects/%s/locations/%s/keyRings/%s", projectID, r.Spec.Location, name)
}
