package controller

import (
	"context"
	"fmt"
	"strings"

	ccv1alpha1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/k8s/v1alpha1"
	tagsv1alpha1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/tags/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/deliveryhero/gcp-config-connector-tagging-operator/internal/gcp"
)

const (
	projectIDAnnotation = "cnrm.cloud.google.com/project-id"
	tagBindingOwnerKey  = ".metadata.controller"
)

type ResourceMetadataProvider[R any] interface {
	GetResourceLocation(r *R) string
	GetResourceID(projectID string, r *R) string
}

type ResourcePointer[T any] interface {
	*T
	client.Object
}

// +kubebuilder:rbac:groups=tags.cnrm.cloud.google.com,resources=tagslocationtagbindings,verbs=get;list;watch;create;update;patch;delete

// TaggableResourceReconciler reconciles any Google Cloud Config Connector object that can be tagged
type TaggableResourceReconciler[T any, P ResourceMetadataProvider[T], PT ResourcePointer[T]] struct {
	client.Client
	Scheme           *runtime.Scheme
	TagsManager      *gcp.TagsManager
	MetadataProvider P
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *TaggableResourceReconciler[T, P, PT]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	resource := r.newPT()
	if err := r.Get(ctx, req.NamespacedName, resource); err != nil {
		log.Error(err, "unable to fetch resource")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	projectID := r.determineProjectID(ctx, resource)
	gvk := resource.GetObjectKind().GroupVersionKind()
	ownerIndex := ownerIndexValue(gvk.GroupVersion().String(), gvk.Kind, resource.GetName())

	var boundTags tagsv1alpha1.TagsLocationTagBindingList
	if err := r.List(ctx, &boundTags, client.InNamespace(req.Namespace), client.MatchingFields{tagBindingOwnerKey: ownerIndex}); err != nil {
		log.Error(err, "unable to list bound tags")
		return ctrl.Result{}, err
	}

	boundTagsMap := make(map[string]*tagsv1alpha1.TagsLocationTagBinding)
	for _, tag := range boundTags.Items {
		boundTagsMap[tag.Name] = &tag
	}

	var expectedTagValueRefs []string
	// TODO add option for allowlisting a subset of labels to be turned into tags (e.g. regex of label keys)
	for k, v := range resource.GetLabels() {
		value, err := r.TagsManager.LookupValue(ctx, projectID, k, v)
		if err != nil {
			return ctrl.Result{}, err
		}
		expectedTagValueRefs = append(expectedTagValueRefs, value.Name)
	}

	expectedResourceNames := make(map[string]bool)
	for _, ref := range expectedTagValueRefs {
		binding, err := r.generateBinding(resource, projectID, ref)
		if err != nil {
			return ctrl.Result{}, err
		}
		expectedResourceNames[binding.Name] = true

		if existingBinding, exists := boundTagsMap[binding.Name]; exists && existingBinding.ObjectMeta.DeletionTimestamp.IsZero() {
			if tagBindingChanged(binding, existingBinding) {
				// bindings are immutable, so we just always re-create
				if err := r.Delete(ctx, existingBinding); err != nil {
					return ctrl.Result{}, err
				}
				if err := r.Create(ctx, binding); err != nil {
					return ctrl.Result{}, err
				}
			}
		} else {
			if err := r.Create(ctx, binding); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	for _, item := range boundTags.Items {
		if _, exists := expectedResourceNames[item.Name]; !exists {
			if err := r.Delete(ctx, &item); err != nil {
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TaggableResourceReconciler[T, P, PT]) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(r.newPT()).
		Owns(&tagsv1alpha1.TagsLocationTagBinding{}).
		Complete(r)
}

func (r *TaggableResourceReconciler[T, P, PT]) newPT() PT {
	return (PT)(new(T))
}

func (r *TaggableResourceReconciler[T, P, PT]) determineProjectID(ctx context.Context, resource PT) string {
	log := log.FromContext(ctx)

	// TODO projectRef

	if projectID, exists := resource.GetAnnotations()[projectIDAnnotation]; exists {
		return projectID
	}

	var ns corev1.Namespace
	if err := r.Get(ctx, types.NamespacedName{Name: resource.GetNamespace()}, &ns); err != nil {
		log.Error(err, "unable to fetch namespace")
	}
	if projectID, exists := ns.ObjectMeta.Annotations[projectIDAnnotation]; exists {
		return projectID
	}

	return ns.Name
}

func (r *TaggableResourceReconciler[T, P, PT]) generateBinding(resource PT, projectID string, tagValueID string) (*tagsv1alpha1.TagsLocationTagBinding, error) {
	binding := &tagsv1alpha1.TagsLocationTagBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:        tagBindingResourceName(resource, tagValueID),
			Namespace:   resource.GetNamespace(),
			Annotations: map[string]string{},
		},
		Spec: tagsv1alpha1.TagsLocationTagBindingSpec{
			Location: r.MetadataProvider.GetResourceLocation(resource),
			ParentRef: ccv1alpha1.ResourceRef{
				External: r.MetadataProvider.GetResourceID(projectID, resource),
			},
			TagValueRef: ccv1alpha1.ResourceRef{
				External: tagValueID,
			},
		},
	}
	if projectID != "" {
		binding.Annotations[projectIDAnnotation] = projectID
	}
	if err := ctrl.SetControllerReference(resource, binding, r.Scheme); err != nil {
		return nil, err
	}

	return binding, nil
}

func tagBindingResourceName(owner client.Object, valueRef string) string {
	kind := strings.ToLower(owner.GetObjectKind().GroupVersionKind().Kind)
	prefix := fmt.Sprintf("%s-%s", kind, owner.GetName())
	valueID, _ := strings.CutPrefix(valueRef, "tagValues/")
	suffix := fmt.Sprintf("-%s", valueID)

	maxPrefixLen := 253 - len(suffix)
	if len(prefix) > maxPrefixLen {
		prefix = prefix[:maxPrefixLen]
	}

	return prefix + suffix
}

func tagBindingChanged(expected, actual *tagsv1alpha1.TagsLocationTagBinding) bool {
	if !equality.Semantic.DeepEqual(actual.Spec.TagValueRef, expected.Spec.TagValueRef) {
		return true
	}

	if !equality.Semantic.DeepEqual(actual.Spec.ParentRef, expected.Spec.ParentRef) {
		return true
	}

	if expected.Spec.Location != actual.Spec.Location {
		return true
	}

	expectedProjectID, _ := expected.ObjectMeta.Annotations[projectIDAnnotation]
	actualProjectID, _ := actual.ObjectMeta.Annotations[projectIDAnnotation]
	if expectedProjectID != actualProjectID {
		return true
	}

	return false
}

func SetupTagBindingIndex(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &tagsv1alpha1.TagsLocationTagBinding{}, tagBindingOwnerKey, func(rawObj client.Object) []string {
		// grab the tags location binding object, extract the owner...
		job := rawObj.(*tagsv1alpha1.TagsLocationTagBinding)
		owner := metav1.GetControllerOf(job)
		if owner == nil {
			return nil
		}

		// ...make sure it's a config connector resource...
		if !strings.Contains(owner.APIVersion, ".cnrm.cloud.google.com") {
			return nil
		}

		// ...and if so, return it
		return []string{ownerIndexValue(owner.APIVersion, owner.Kind, owner.Name)}
	}); err != nil {
		return err
	}

	return nil
}

func ownerIndexValue(apiVersion string, kind string, name string) string {
	return fmt.Sprintf("%s/%s/%s", apiVersion, kind, name)
}
