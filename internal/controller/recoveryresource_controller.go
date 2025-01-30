/*
Copyright 2025.

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
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kuberecoveryv1alpha1 "freepik.com/kuberecovery/api/v1alpha1"
)

// RecoveryResourceReconciler reconciles a RecoveryResource object
type RecoveryResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kuberecovery.freepik.com,resources=recoveryresources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kuberecovery.freepik.com,resources=recoveryresources/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kuberecovery.freepik.com,resources=recoveryresources/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *RecoveryResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	// 1. Get the content of the Patch
	kubeRecoveryResource := &kuberecoveryv1alpha1.RecoveryResource{}
	err = r.Get(ctx, req.NamespacedName, kubeRecoveryResource)

	// 2. Check existence on the cluster
	if err != nil {

		// 2.1 It does NOT exist: manage removal
		if err = client.IgnoreNotFound(err); err == nil {
			logger.Info(fmt.Sprintf(resourceNotFoundError, recoveryResourceType, req.NamespacedName))
			return result, err
		}

		// 2.2 Failed to get the resource, requeue the request
		logger.Info(fmt.Sprintf(resourceSyncTimeRetrievalError, recoveryResourceType, req.NamespacedName, err.Error()))
		return result, err
	}

	// 3. Check if the RecoveryResource resource is marked to be deleted: indicated by the deletion timestamp being set
	if !kubeRecoveryResource.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(kubeRecoveryResource, resourceFinalizer) {

			// 3.1 Delete the resources associated with the QueryConnector
			err = r.Sync(ctx, watch.Deleted, kubeRecoveryResource)

			// Remove the finalizers on Patch CR
			controllerutil.RemoveFinalizer(kubeRecoveryResource, resourceFinalizer)
			err = r.Update(ctx, kubeRecoveryResource)
			if err != nil {
				logger.Info(fmt.Sprintf(resourceFinalizersUpdateError, recoveryResourceType, req.NamespacedName, err.Error()))
			}
		}

		result = ctrl.Result{}
		err = nil
		return result, err
	}

	// 4. Add finalizer to the RecoveryResource CR
	if !kubeRecoveryResource.DeletionTimestamp.IsZero() {
		controllerutil.AddFinalizer(kubeRecoveryResource, resourceFinalizer)
		err = r.Update(ctx, kubeRecoveryResource)
		if err != nil {
			return result, err
		}
	}

	// 5. Update the status before the requeue
	defer func() {
		err = r.Status().Update(ctx, kubeRecoveryResource)
		if err != nil {
			logger.Info(fmt.Sprintf(resourceConditionUpdateError, recoveryResourceType, req.NamespacedName, err.Error()))
		}
	}()

	// 6. Schedule periodical request
	RequeueTime, err := time.ParseDuration("1m")
	if err != nil {
		logger.Info(fmt.Sprintf(resourceSyncTimeRetrievalError, recoveryResourceType, req.NamespacedName, err.Error()))
		return result, err
	}
	result = ctrl.Result{
		RequeueAfter: RequeueTime,
	}

	// 7. Check if the resource can be deleted
	err = r.Sync(ctx, watch.Modified, kubeRecoveryResource)
	if err != nil {
		r.UpdateConditionKubernetesApiCallFailure(kubeRecoveryResource)
		logger.Info(fmt.Sprintf(syncTargetError, recoveryResourceType, req.NamespacedName, err.Error()))
		return result, err
	}

	// 8. Success, update the status
	r.UpdateConditionSuccess(kubeRecoveryResource)

	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *RecoveryResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kuberecoveryv1alpha1.RecoveryResource{}).
		Named("recoveryresource").
		Complete(r)
}
