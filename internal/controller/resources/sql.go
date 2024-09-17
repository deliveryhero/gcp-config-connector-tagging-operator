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

	"k8s.io/utils/ptr"

	sqlv1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/sql/v1beta1"
	"github.com/deliveryhero/gcp-config-connector-tagging-operator/internal/controller"
)

// +kubebuilder:rbac:groups=sql.cnrm.cloud.google.com,resources=sqlinstances,verbs=get;list;watch

var _ controller.ResourceMetadataProvider[sqlv1beta1.SQLInstance] = &SQLInstanceMetadataProvider{}

type SQLInstanceMetadataProvider struct{}

func (in *SQLInstanceMetadataProvider) GetResourceLocation(r *sqlv1beta1.SQLInstance) string {
	return ptr.Deref(r.Spec.Region, "")
}

func (in *SQLInstanceMetadataProvider) GetResourceID(projectID string, r *sqlv1beta1.SQLInstance) string {
	name := r.Name
	if r.Spec.ResourceID != nil {
		name = *r.Spec.ResourceID
	}

	return fmt.Sprintf("//sqladmin.googleapis.com/projects/%s/instances/%s", projectID, name)
}
