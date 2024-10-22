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
	"strings"

	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	redisv1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/redis/v1beta1"
	"github.com/deliveryhero/gcp-config-connector-tagging-operator/internal/controller"
)

// +kubebuilder:rbac:groups=redis.cnrm.cloud.google.com,resources=redisinstances,verbs=get;list;watch;update

var _ controller.ResourceMetadataProvider[redisv1beta1.RedisInstance] = &RedisInstanceMetadataProvider{}

type RedisInstanceMetadataProvider struct{}

func (in *RedisInstanceMetadataProvider) GetResourceLocation(r *redisv1beta1.RedisInstance) string {
	return r.Spec.Region
}

func (in *RedisInstanceMetadataProvider) GetResourceID(projectInfo *resourcemanagerpb.Project, r *redisv1beta1.RedisInstance) string {
	name := r.Name
	if r.Spec.ResourceID != nil {
		name = *r.Spec.ResourceID
	}

	region := r.Spec.Region

	projectNumber := strings.TrimPrefix(projectInfo.Name, "projects/")

	return fmt.Sprintf("//redis.googleapis.com/projects/%s/locations/%s/instances/%s", projectNumber, region, name)
}
