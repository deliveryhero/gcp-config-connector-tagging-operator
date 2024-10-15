package resources

import (
	"fmt"

	kmsv1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/kms/v1beta1"

	"github.com/deliveryhero/gcp-config-connector-tagging-operator/internal/controller"
)

// +kubebuilder:rbac:groups=kms.cnrm.cloud.google.com,resources=kmskeyrings,verbs=get;list;watch
// +kubebuilder:rbac:groups=kms.cnrm.cloud.google.com,resources=kmskeyrings/finalizers,verbs=update

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
