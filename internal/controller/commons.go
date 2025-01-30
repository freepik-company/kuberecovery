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
	"freepik.com/kuberecovery/internal/globals"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/restmapper"
)

const (

	// Resource types
	recoveryConfigType         = "RecoveryConfig"
	recoveryResourceType       = "RecoveryResource"
	recoveryResourceTypePlural = "recoveryresources"

	// Sync interval to check if secrets of SearchRuleAction and SearchRuleQueryConnector are up to date
	defaultSyncInterval = "1m"

	// Formats
	recoveryResourceNameFormat = "%s-%s-%s-%s"
	timeParseFormat            = "2006-01-02T150405"
	timeParseFormatName        = "20060102150405"

	// Error messages
	resourceNotFoundError          = "%s '%s' resource not found. Ignoring since object must be deleted."
	resourceFinalizersUpdateError  = "Failed to update finalizer of %s '%s': %s"
	resourceConditionUpdateError   = "Failed to update the condition on %s '%s': %s"
	resourceSyncTimeRetrievalError = "can not get synchronization time from the %s '%s': %s"
	syncTargetError                = "can not sync the target for the %s '%s': %s"
	timeParseError                 = "error parsing time: %v"
	deleteExtraFinalizerError      = "error deleting extra finalizer to resource %s: %v"
	resourceDeleteError            = "error deleting resource %s: %v"
	addExtraFinalizerError         = "error adding extra finalizer to resource %s: %v"
	deserializingRawExtensionError = "error deserializing RawExtension: %v"
	getResourceFromKindError       = "error getting resource from kind: %v"
	createResourceError            = "error creating resource %s in the cluster: %v"
	deleteRestoreLabelError        = "error deleting restore label from resource %s: %v"
	convertToUnstructuredError     = "failed to convert object to unstructured object: %v"
	regexResourceError             = "failed to regex resource to exclude %s: %v"
	regexNamespaceError            = "failed to regex namespace to exclude %s: %v"
	saveRecoveryResourceError      = "Failed to save resource %s/%s/%s/%s as RecoveryResource: %v"
	resourceWatcherError           = "error creating event handler for resource %s/%s: %v"
	recoveryResourceCreationError  = "error creating recoveryResource %s in the cluster: %w"

	// Info messages
	resourceExpiredMessage              = "Resource %s is expired, deleting it"
	resourceRestoreMessage              = "Resource %s has restore label, restoring it"
	stopWatchingResourceMessage         = "Stopping watching %s/%s in namespace %s"
	startWatchingResourceMessage        = "Watching %s/%s in namespace %s"
	resourceExcludedFromRecoveryMessage = "Resource %s/%s/%s is excluded from recovery"
	recoveryResourceSavedMessage        = "Resource %s/%s/%s/%s saved as RecoveryResource %s"

	// Finalizer
	resourceFinalizer              = "kuberecovery.freepik.com/finalizer"
	recoveryResourceExtraFinalizer = "kuberecovery.freepik.com/protectFinalizer"

	// Labels
	recoveryResourceRetainUntilLabel    = "kuberecovery.freepik.com/retentionUntil"
	recoveryResourceSavedAtLabel        = "kuberecovery.freepik.com/savedAt"
	recoveryResourceRecoveryConfigLabel = "kuberecovery.freepik.com/recoveryConfig"
	recoveryResourceRestoreLabel        = "kuberecovery.freepik.com/restore"
	recoveryResourceRestoreLabelValue   = "true"
)

// getResourceFromKind returns the resource name from the group, version and kind
func getResourceFromKind(group, version, kind string) (string, error) {

	// Get the discovery client
	discoveryClient := globals.Application.KubeRawCoreClient.Discovery()

	// Get the API group resources
	groupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return "", err
	}

	// Get the REST mapper
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

	// Get the group version kind
	gvk := schema.GroupVersionKind{Group: group, Version: version, Kind: kind}

	// Get the REST mapping for the group version kind
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return "", err
	}

	// Return the resource
	return mapping.Resource.Resource, nil
}
