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
	"encoding/json"
	"fmt"
	"k8s.io/client-go/dynamic"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kuberecoveryv1alpha1 "freepik.com/kuberecovery/api/v1alpha1"
	"freepik.com/kuberecovery/internal/globals"
)

// Sync checks if the resource is expired and deletes it if it is
// Also recreate the resource if it has a specific label
func (r *RecoveryResourceReconciler) Sync(ctx context.Context,
	resource *kuberecoveryv1alpha1.RecoveryResource) (err error) {

	logger := log.FromContext(ctx)

	// Add extra finalizer to the resource and just remove it if the retainUntil label has a past date
	if !controllerutil.ContainsFinalizer(resource, recoveryResourceExtraFinalizer) {
		controllerutil.AddFinalizer(resource, recoveryResourceExtraFinalizer)
		err = r.Update(ctx, resource)
		if err != nil {
			return fmt.Errorf(addExtraFinalizerError, resource.Name, err)
		}
	}

	// Get validUntil label from the resource
	validUntilLabel := resource.GetLabels()[recoveryResourceRetainUntilLabel]

	// Parse the validUntil label to get the time
	validUntil, err := time.Parse(timeParseFormat, validUntilLabel)
	if err != nil {
		return fmt.Errorf(timeParseError, err)
	}

	// Check if the resource is expired
	if time.Now().UTC().After(validUntil) {

		// If the resource is expired, delete it.
		// First remove the finalizer and then delete the resource
		logger.Info(fmt.Sprintf(resourceExpiredMessage, resource.Name))
		if controllerutil.ContainsFinalizer(resource, recoveryResourceExtraFinalizer) {
			controllerutil.RemoveFinalizer(resource, recoveryResourceExtraFinalizer)
			err = r.Update(ctx, resource)
			if err != nil {
				return fmt.Errorf(deleteExtraFinalizerError, resource.Name, err)
			}
		}
		err = r.Delete(ctx, resource)
		if err != nil {
			return fmt.Errorf(resourceDeleteError, resource.Name, err)
		}

		return nil
	}

	// Get restore label trigger. If it is present, restore the resource
	restoreTriggerLabel := resource.GetLabels()[recoveryResourceRestoreLabel]
	if restoreTriggerLabel == recoveryResourceRestoreLabelValue {
		logger.Info(fmt.Sprintf(resourceRestoreMessage, resource.Name))

		// Remove the restore label to avoid restoring the resource again
		defer func() {
			delete(resource.GetLabels(), recoveryResourceRestoreLabel)
			err = r.Update(ctx, resource)
			if err != nil {
				logger.Info(fmt.Sprintf(deleteRestoreLabelError, resource.Name, err))
			}
		}()

		// Unmarshal the RawExtension to the object to get unstructured.Unstructured
		var resourceToRestore unstructured.Unstructured
		if err := json.Unmarshal(resource.Spec.Raw, &resourceToRestore.Object); err != nil {
			return fmt.Errorf(deserializingRawExtensionError, err)
		}

		// Create the GVR for the RecoveryResource
		res, err := getResourceFromKind(resourceToRestore.GroupVersionKind().Group,
			resourceToRestore.GroupVersionKind().Version, resourceToRestore.GroupVersionKind().Kind)
		if err != nil {
			return fmt.Errorf(getResourceFromKindError, err)
		}
		gvr := schema.GroupVersionResource{
			Group:    resourceToRestore.GroupVersionKind().Group,
			Version:  resourceToRestore.GroupVersionKind().Version,
			Resource: res,
		}

		// Create the dynamic client for the RecoveryResource for namespaced and cluster-scoped resources
		var dynamicClient dynamic.ResourceInterface
		if resourceToRestore.GetNamespace() != "" {
			dynamicClient = globals.Application.KubeRawClient.Resource(gvr).Namespace(resourceToRestore.GetNamespace())
		} else {
			dynamicClient = globals.Application.KubeRawClient.Resource(gvr)
		}

		// Create the resource saved in the RecoveryResource spec
		_, err = dynamicClient.Create(ctx, &resourceToRestore, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf(createResourceError, resourceToRestore.GetName(), err)
		}

		logger.Info(fmt.Sprintf(resourceRestoredSuccessfullyMessage, resource.Name,
			resourceToRestore.GroupVersionKind().Group, resourceToRestore.GroupVersionKind().Version,
			resourceToRestore.GroupVersionKind().Kind, resourceToRestore.GetNamespace(), resourceToRestore.GetName()))
	}

	return nil
}
