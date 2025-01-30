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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/restmapper"

	"freepik.com/kuberecovery/internal/globals"
)

const (

	// Resource types
	recoveryConfigType         = "RecoveryConfig"
	recoveryResourceType       = "RecoveryResource"
	recoveryResourceTypePlural = "recoveryresources"

	// Sync interval to check if secrets of SearchRuleAction and SearchRuleQueryConnector are up to date
	defaultSyncInterval = "1m"

	// Error messages
	resourceNotFoundError          = "%s '%s' resource not found. Ignoring since object must be deleted."
	resourceFinalizersUpdateError  = "Failed to update finalizer of %s '%s': %s"
	resourceConditionUpdateError   = "Failed to update the condition on %s '%s': %s"
	resourceSyncTimeRetrievalError = "can not get synchronization time from the %s '%s': %s"
	syncTargetError                = "can not sync the target for the %s '%s': %s"

	// Finalizer
	resourceFinalizer              = "kuberecovery.freepik.com/finalizer"
	recoveryResourceExtraFinalizer = "kuberecovery.freepik.com/protectFinalizer"

	// Labels
	recoveryResourceRetainUntilLabel    = "kuberecovery.freepik.com/retentionUntil"
	recoveryResourceSavedAtLabel        = "kuberecovery.freepik.com/savedAt"
	recoveryResourceRecoveryConfigLabel = "kuberecovery.freepik.com/recoveryConfig"
)

// getPluralKind
func getPluralKind(group, version, kind string) (string, error) {

	//
	discoveryClient := globals.Application.KubeRawCoreClient.Discovery()

	groupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return "", err
	}

	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

	gvk := schema.GroupVersionKind{Group: group, Version: version, Kind: kind}

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return "", err
	}

	return mapping.Resource.Resource, nil
}
