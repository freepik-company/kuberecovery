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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RetentionT TODO
type RetentionT struct {
	Period string `json:"period"`
}

// GvkResource TODO
type GvrResourceT struct {
	APIVersion string   `json:"apiVersion"`
	Resources  []string `json:"resources"`
	Namespaces []string `json:"namespaces,omitempty"`
}

// RecoveryConfigSpec defines the desired state of RecoveryConfig.
type RecoveryConfigSpec struct {
	ResourcesIncluded []GvrResourceT `json:"resourcesIncluded,omitempty"`
	ResourcesExcluded []GvrResourceT `json:"resourcesExcluded,omitempty"`
	Retention         RetentionT     `json:"retention"`
}

// RecoveryConfigStatus defines the observed state of RecoveryConfig.
type RecoveryConfigStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// RecoveryConfig is the Schema for the recoveryconfigs API.
type RecoveryConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RecoveryConfigSpec   `json:"spec,omitempty"`
	Status RecoveryConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RecoveryConfigList contains a list of RecoveryConfig.
type RecoveryConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RecoveryConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RecoveryConfig{}, &RecoveryConfigList{})
}
