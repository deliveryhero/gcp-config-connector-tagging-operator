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

package batch

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cri-api/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	tagsv1alpha1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/tags/v1alpha1"
	batchv1 "github.com/deliveryhero/gcp-config-connector-tagging-operator/api/batch/v1"
	"github.com/deliveryhero/gcp-config-connector-tagging-operator/internal/gcp"
)

// CronJobReconciler reconciles a CronJob object
type CronJobReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	TagsManager gcp.TagsManager
}

//+kubebuilder:rbac:groups=batch.gdp.deliveryhero.io,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch.gdp.deliveryhero.io,resources=cronjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batch.gdp.deliveryhero.io,resources=cronjobs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CronJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *CronJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Starting reconciliation to delete unused tags")

	unusedTags, err := r.identifyUnusedTags(ctx)
	if err != nil {
		log.Error(err, "Failed to identify unused tags")
		return ctrl.Result{}, err
	}

	if err := r.deleteUnusedTags(ctx, unusedTags); err != nil {
		log.Error(err, "Failed to delete unused tags")
		return ctrl.Result{}, err
	}

	log.Info("Reconciliation complete")
	return ctrl.Result{RequeueAfter: 24 * time.Hour}, nil // Requeue daily
}

// finds tags that have no active bindings
func (r *CronJobReconciler) identifyUnusedTags(ctx context.Context) ([]tagsv1alpha1.TagsLocationTagBinding, error) {
	var unusedTags []tagsv1alpha1.TagsLocationTagBinding
	var allTagBindings tagsv1alpha1.TagsLocationTagBindingList

	// List all tag bindings
	if err := r.List(ctx, &allTagBindings); err != nil {
		return nil, fmt.Errorf("failed to list tag bindings: %w", err)
	}

	// Identify bindings marked for deletion
	for _, tagBinding := range allTagBindings.Items {
		if tagBinding.ObjectMeta.DeletionTimestamp.IsZero() {
			// Tag in-use
			continue
		}
		unusedTags = append(unusedTags, tagBinding)
	}
	return unusedTags, nil
}

// remove the specified unused tags from the system
func (r *CronJobReconciler) deleteUnusedTags(ctx context.Context, unusedTags []tagsv1alpha1.TagsLocationTagBinding) error {
	for _, tag := range unusedTags {
		if err := r.Delete(ctx, &tag); err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete unused tag %s: %w", tag.Name, err)
			}
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CronJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.CronJob{}).
		Complete(r)
}
