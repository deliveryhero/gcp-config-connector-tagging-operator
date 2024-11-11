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
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	ccv1alpha1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/k8s/v1alpha1"
	tagsv1alpha1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/tags/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/deliveryhero/gcp-config-connector-tagging-operator/internal/gcp"
)

const (
	projectIDAnnotation       = "cnrm.cloud.google.com/project-id"
	tagBindingOwnerKey        = ".metadata.controller"
	taggableResourceFinalizer = "gdp.deliveryhero.io/resource-tags"
)

var (
	setupLog = ctrl.Log.WithName("setup")
)

type ResourceMetadataProvider[R any] interface {
	GetResourceLocation(r *R) string
	GetResourceID(projectInfo *resourcemanagerpb.Project, r *R) string
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
	TagsManager      gcp.TagsManager
	MetadataProvider P
	LabelMatcher     func(map[string]string) map[string]string
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

	// Handle TagBinding deletion using finalizer
	if !resource.GetDeletionTimestamp().IsZero() {
		// Resource is being deleted
		if controllerutil.ContainsFinalizer(resource, taggableResourceFinalizer) {
			if err := r.handleTagBindingsDeletion(ctx, resource); err != nil {
				// If there's an error handling tag bindings, requeue for later
				return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, err
			}
			// Remove finalizer to allow Kubernetes to delete the resource
			controllerutil.RemoveFinalizer(resource, taggableResourceFinalizer)
			if err := r.Update(ctx, resource); err != nil {
				return ctrl.Result{}, err
			}
			log.Info("resource deletion request received trying to delete associated tagValue/tagKey if unused")
			projectID := r.determineProjectID(ctx, resource)
			labels := resource.GetLabels()
			for k, v := range r.LabelMatcher(labels) {
				// return tagValue.Name, tagKey.Name, nil
				valueID, keyID, err := r.getValueAndKeyID(ctx, projectID, k, v)
				if err != nil {
					return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, err
				}
				if err := r.TagsManager.DeleteValueIfUnused(ctx, projectID, keyID, valueID); err != nil {
					return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, err
				}
				if err := r.TagsManager.DeleteKeyIfUnused(ctx, projectID, keyID); err != nil {
					return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, err
				}
			}

		}
		// Stop reconciliation as the resource is being deleted
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(resource, taggableResourceFinalizer) {
		controllerutil.AddFinalizer(resource, taggableResourceFinalizer)
		if err := r.Update(ctx, resource); err != nil {
			return ctrl.Result{}, err
		}
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
	labels := resource.GetLabels()

	for k, v := range r.LabelMatcher(labels) {
		value, err := r.TagsManager.LookupValue(ctx, projectID, k, v)
		if err != nil {
			return ctrl.Result{}, err
		}
		expectedTagValueRefs = append(expectedTagValueRefs, value.Name)
	}

	projectInfo, err := r.TagsManager.GetProjectInfo(ctx, projectID)
	if err != nil {
		return ctrl.Result{}, err
	}

	expectedResourceNames := make(map[string]bool)
	for _, ref := range expectedTagValueRefs {
		binding, err := r.generateBinding(resource, projectInfo, ref)
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

func (r *TaggableResourceReconciler[T, P, PT]) generateBinding(resource PT, projectInfo *resourcemanagerpb.Project, tagValueID string) (*tagsv1alpha1.TagsLocationTagBinding, error) {
	binding := &tagsv1alpha1.TagsLocationTagBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:        tagBindingResourceName(resource, tagValueID),
			Namespace:   resource.GetNamespace(),
			Annotations: map[string]string{},
		},
		Spec: tagsv1alpha1.TagsLocationTagBindingSpec{
			Location: r.MetadataProvider.GetResourceLocation(resource),
			ParentRef: ccv1alpha1.ResourceRef{
				External: r.MetadataProvider.GetResourceID(projectInfo, resource),
			},
			TagValueRef: ccv1alpha1.ResourceRef{
				External: tagValueID,
			},
		},
	}

	binding.Annotations[projectIDAnnotation] = projectInfo.ProjectId

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

// handleTagBindingsDeletion handles the deletion of tag bindings when a resource is being deleted.
func (r *TaggableResourceReconciler[T, P, PT]) handleTagBindingsDeletion(ctx context.Context, resource PT) error {
	log := log.FromContext(ctx)

	gvk := resource.GetObjectKind().GroupVersionKind()
	ownerIndex := ownerIndexValue(gvk.GroupVersion().String(), gvk.Kind, resource.GetName())

	var boundTags tagsv1alpha1.TagsLocationTagBindingList
	if err := r.List(ctx, &boundTags, client.InNamespace(resource.GetNamespace()), client.MatchingFields{tagBindingOwnerKey: ownerIndex}); err != nil {
		log.Error(err, "unable to list bound tags")
		return err
	}

	for _, tagBinding := range boundTags.Items {
		// Delete the tag binding directly
		log.Info("deleting tag binding", "name", tagBinding.Name)
		if err := r.Delete(ctx, &tagBinding); err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf("error deleting tag binding %s: %w", tagBinding.Name, err)
			}
		}
		// Cleanup finalizer
		controllerutil.RemoveFinalizer(&tagBinding, "cnrm.cloud.google.com/finalizer")
		if err := r.Update(ctx, &tagBinding); err != nil {
			return fmt.Errorf("error removing finalizer from resource %s: %w", tagBinding.Name, err)
		}
	}

	return nil
}

func (r *TaggableResourceReconciler[T, P, PT]) getValueAndKeyID(ctx context.Context, projectID, key, value string) (string, string, error) {
	tagValue, err := r.TagsManager.LookupValue(ctx, projectID, key, value)
	if err != nil {
		return "", "", fmt.Errorf("failed to lookup tag value: %w", err)
	}

	tagKey, err := r.TagsManager.LookupKey(ctx, projectID, key)
	if err != nil {
		return "", "", fmt.Errorf("failed to lookup tag key: %w", err)
	}

	return tagValue.Name, tagKey.Name, nil
}

func CreateTaggableResourceController[T any, P ResourceMetadataProvider[T], PT ResourcePointer[T]](mgr ctrl.Manager, tagsManager gcp.TagsManager, provider P, labelMatcher func(map[string]string) map[string]string) {
	if err := (&TaggableResourceReconciler[T, P, PT]{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		TagsManager:      tagsManager,
		MetadataProvider: provider,
		LabelMatcher:     labelMatcher,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create taggable resource controller")
		os.Exit(1)
	}
}
