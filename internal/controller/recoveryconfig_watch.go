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
	"regexp"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kuberecoveryv1alpha1 "freepik.com/kuberecovery/api/v1alpha1"
	"freepik.com/kuberecovery/internal/globals"
	"freepik.com/kuberecovery/internal/pools"
)

func (r *RecoveryConfigReconciler) Watch(ctx context.Context, eventType watch.EventType,
	resource *kuberecoveryv1alpha1.RecoveryConfig) (err error) {

	logger := log.FromContext(ctx)

	newInformers := make(map[string]bool)

	// For each GVR included in the ResourcesIncluded section of the RecoveryConfig, we create an informer to watch
	//delete events on the resources
	// GV = GroupVersion (APIVersion) is a string, so just one group by ResourceIncluded is allowed
	// K = Kind is a string array, so multiple kinds by ResourceIncluded is allowed
	// N = Namespace is a string array, so multiple namespaces by ResourceIncluded is allowed
	for _, res := range resource.Spec.ResourcesIncluded {
		// Kind must be an array so, for each kind, we create an informer
		for _, rsc := range res.Resources {
			// If no namespace is specified or the wildcard is used, we watch all namespaces
			namespaces := res.Namespaces
			if len(namespaces) == 0 || (len(namespaces) == 1 && namespaces[0] == "*") {
				namespaces = []string{""}
			}

			// For each namespace, we create an informer
			for _, ns := range namespaces {

				// Key to store the informer in the pool
				resourceWatcherKey := fmt.Sprintf("%s/%s/%s/%s", resource.Name, res.APIVersion, rsc, ns)
				newInformers[resourceWatcherKey] = true

				// Check if the informer is already created and added to the pool
				resourceWatcher, exists := r.ResourceWatcherPool.Get(resourceWatcherKey)

				// If the informer is in the pool and the event type is Deleted, we remove it from the pool
				// and stop the informer
				if exists && eventType == watch.Deleted {
					logger.Info(fmt.Sprintf("Stopping watching %s/%s in namespace %s", res.APIVersion, rsc, ns))
					// Delete the resource watcher from the pool and close the channel
					r.ResourceWatcherPool.Delete(resourceWatcherKey)
					close(resourceWatcher.Chan)
					continue
				}

				// If the informer is not in the pool, we create it
				if !exists {

					// Log the resource we are going to watch
					logger.Info(fmt.Sprintf("Watching %s/%s in namespace %s", res.APIVersion, rsc, ns))

					// Create the resource watcher to add it to the pool
					resourceWatcher := &pools.ResourceWatcher{
						RecoveryConfigName: resource.Name,
						APIVersion:         res.APIVersion,
						Resource:           rsc,
						Namespace:          ns,
						Chan:               make(chan struct{}),
					}

					// Add the resource watcher to the pool and create the informer
					r.ResourceWatcherPool.Set(resourceWatcherKey, resourceWatcher)
					go r.createInformer(ctx, resourceWatcher, resource)
				}
			}
		}
	}

	// Get all the existing informers from the pool
	existingInformers := r.ResourceWatcherPool.GetAll()

	// For each informer in the pool, we check if it is not in the new resources list
	// If it is not in the new resources list, we stop the informer and remove it from the pool
	for key, watcher := range existingInformers {
		if _, exists := newInformers[key]; !exists {
			logger.Info(fmt.Sprintf("Stopping watching %s/%s in namespace %s", watcher.APIVersion, watcher.Resource, watcher.Namespace))
			close(watcher.Chan)
			r.ResourceWatcherPool.Delete(key)
		}
	}

	return nil
}

// createInformer creates an informer for the resource specified in the resourceWatcher
func (r *RecoveryConfigReconciler) createInformer(ctx context.Context, resourceWatcher *pools.ResourceWatcher,
	recoveryConfig *kuberecoveryv1alpha1.RecoveryConfig) {

	logger := log.FromContext(ctx)

	// Split the APIVersion into Group and Version if group is present
	group := ""
	apiVersion := resourceWatcher.APIVersion
	if idx := strings.Index(apiVersion, "/"); idx != -1 {
		group = apiVersion[:idx]
		apiVersion = apiVersion[idx+1:]
	}

	// Create the GVR for the resource
	gvr := &schema.GroupVersionResource{
		Group:    group,
		Version:  apiVersion,
		Resource: resourceWatcher.Resource,
	}

	// Creates the informer factory for the resource and the namespaces specified in the resourceWatcher
	// using the global dynamic client defined for the operator
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
		globals.Application.KubeRawClient,
		0,
		resourceWatcher.Namespace,
		nil,
	)

	// Creates the informer for the gvr defined
	informer := factory.ForResource(*gvr).Informer()

	// Add event handler to the informer and listen for delete events
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// Listen for delete events
		DeleteFunc: func(obj interface{}) {

			// Get the object deleted as unstructured object
			unstructuredObj, ok := obj.(*unstructured.Unstructured)
			if !ok {
				logger.Error(fmt.Errorf("failed to convert object to unstructured"),
					"Failed to delete resource")
				return
			}

			// Get the plural kind of the resource
			resource, err := getResourceFromKind(unstructuredObj.GroupVersionKind().Group,
				unstructuredObj.GroupVersionKind().Version, unstructuredObj.GroupVersionKind().Kind)
			if err != nil {
				logger.Error(err, "Failed to get plural kind")
				return
			}

			// Check if the resource is excluded to save it as RecoveryResource
			for _, excluded := range recoveryConfig.Spec.ResourcesExcluded {
				for _, excludedKind := range excluded.Resources {
					for _, excludedNamespace := range excluded.Namespaces {

						// kind and namespace can be regex
						kindMatched, err := regexp.MatchString(excludedKind, resource)
						if err != nil {
							logger.Error(err, "Failed to match kind")
							return
						}
						namespaceMatched, err := regexp.MatchString(excludedNamespace, unstructuredObj.GetNamespace())
						if err != nil {
							logger.Error(err, "Failed to match namespace")
							return
						}

						// Check if the resource is excluded, if true we do not save it as RecoveryResource
						if excluded.APIVersion == unstructuredObj.GetAPIVersion() && kindMatched && namespaceMatched {
							logger.Info(fmt.Sprintf("Resource %s/%s/%s is excluded from recovery",
								unstructuredObj.GetAPIVersion(), resource, unstructuredObj.GetNamespace()))
							return
						}
					}
				}
			}

			// Save the resource as RecoveryResource
			err = r.saveRecoveryResource(ctx, unstructuredObj, recoveryConfig)
			if err != nil {
				logger.Error(err, "Failed to save recovery resource")
			}
			logger.Info(fmt.Sprintf("Resource %s/%s/%s/%s saved as RecoveryResource",
				unstructuredObj.GetAPIVersion(), unstructuredObj.GetKind(), unstructuredObj.GetNamespace(),
				unstructuredObj.GetName()))

		},
	})
	if err != nil {
		logger.Error(err, "Failed to add event handler")
	}

	// Run the informer until the channel stored in the pool is closed
	informer.Run(resourceWatcher.Chan)
}

// saveRecoveryResource saves the resource as RecoveryResource in the cluster
func (r *RecoveryConfigReconciler) saveRecoveryResource(ctx context.Context, obj *unstructured.Unstructured,
	recoveryConfig *kuberecoveryv1alpha1.RecoveryConfig) (err error) {

	// Get the retention time for the RecoveryResource created
	retentionPeriod := recoveryConfig.Spec.Retention.Period
	parsedRetentionPeriod, err := time.ParseDuration(retentionPeriod)
	if err != nil {
		return fmt.Errorf("failed to parse retention period: %w", err)
	}

	// Calculate the retention time for the RecoveryResource created
	recoveryResourceName := fmt.Sprintf("%s-%s-%s-%s", recoveryConfig.Name, strings.ToLower(obj.GetKind()),
		obj.GetName(), metav1.Now().UTC().Format("20060102150405"))
	savedAt := metav1.Now().UTC().Format("2006-01-02T150405")
	retentionUntil := metav1.Now().UTC().Add(parsedRetentionPeriod).Format("2006-01-02T150405")

	// Create the RecoveryResource object
	recoveryObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": kuberecoveryv1alpha1.GroupVersion.String(),
			"kind":       recoveryResourceType,
			"metadata": map[string]interface{}{
				"name": recoveryResourceName,
				"labels": map[string]interface{}{
					recoveryResourceSavedAtLabel:        savedAt,
					recoveryResourceRetainUntilLabel:    retentionUntil,
					recoveryResourceRecoveryConfigLabel: recoveryConfig.Name,
				},
			},
			"spec": obj.Object,
		},
	}

	// Create the GVR for the RecoveryResource
	gvr := schema.GroupVersionResource{
		Group:    kuberecoveryv1alpha1.GroupVersion.Group,
		Version:  kuberecoveryv1alpha1.GroupVersion.Version,
		Resource: recoveryResourceTypePlural,
	}

	// Create the dynamic client for the RecoveryResource
	dynamicClient := globals.Application.KubeRawClient.Resource(gvr)

	// Save the RecoveryResource in the cluster
	_, err = dynamicClient.Create(ctx, recoveryObj, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating recoveryResource %s in the cluster: %w", recoveryObj.GetName(), err)
	}

	return nil
}
