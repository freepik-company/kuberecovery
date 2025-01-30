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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	kuberecoveryv1alpha1 "freepik.com/kuberecovery/api/v1alpha1"
)

// Sync checks if the resource is expired and deletes it if it is
// Also recreate the resource if it has a specific label
func (r *RecoveryResourceReconciler) Sync(ctx context.Context,
	resource *kuberecoveryv1alpha1.RecoveryResource) (err error) {

	logger := log.FromContext(ctx)

	// Get validUntil label from the resource
	validUnitlLabel := resource.GetLabels()[recoveryResourceRetainUntilLabel]

	// Parse the validUntil label to get the time
	validUntil, err := time.Parse("2006-01-02T150405", validUnitlLabel)
	if err != nil {
		return fmt.Errorf("error parsing valid until label %s: %w", validUnitlLabel, err)
	}

	// Check if the resource is expired
	if time.Now().UTC().After(validUntil) {

		// If the resource is expired, delete it.
		// First remove the finalizer and then delete the resource
		logger.Info(fmt.Sprintf("Resource %s is expired, deleting it", resource.Name))
		if controllerutil.ContainsFinalizer(resource, recoveryResourceExtraFinalizer) {
			controllerutil.RemoveFinalizer(resource, recoveryResourceExtraFinalizer)
			err = r.Update(ctx, resource)
			if err != nil {
				return fmt.Errorf("error deleting extra finalizer to resource %s: %w", resource.Name, err)
			}
		}
		err = r.Delete(ctx, resource)
		if err != nil {
			return fmt.Errorf("error deleting resource %s: %w", resource.Name, err)
		}
		return nil
	}

	// Add extra finalizer to the resource and just remove it if the retainUntil label has a past date
	if !controllerutil.ContainsFinalizer(resource, recoveryResourceExtraFinalizer) {
		controllerutil.AddFinalizer(resource, recoveryResourceExtraFinalizer)
		err = r.Update(ctx, resource)
		if err != nil {
			return fmt.Errorf("error adding extra finalizer to resource %s: %w", resource.Name, err)
		}
	}
	return nil
}
