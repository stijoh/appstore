/*
Copyright 2026.

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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AppDeploymentPhase represents the current phase of the deployment
type AppDeploymentPhase string

const (
	PhasePending      AppDeploymentPhase = "Pending"
	PhaseInstalling   AppDeploymentPhase = "Installing"
	PhaseUpgrading    AppDeploymentPhase = "Upgrading"
	PhaseDeployed     AppDeploymentPhase = "Deployed"
	PhaseFailed       AppDeploymentPhase = "Failed"
	PhaseUninstalling AppDeploymentPhase = "Uninstalling"
)

// ValuesReference references a ConfigMap or Secret for Helm values
type ValuesReference struct {
	// Kind of the values referent (ConfigMap or Secret)
	// +kubebuilder:validation:Enum=ConfigMap;Secret
	Kind string `json:"kind"`

	// Name of the referent
	Name string `json:"name"`

	// ValuesKey is the key in the referent to read
	// +kubebuilder:default=values.yaml
	// +optional
	ValuesKey string `json:"valuesKey,omitempty"`

	// Optional marks this reference as optional
	// +kubebuilder:default=false
	// +optional
	Optional bool `json:"optional,omitempty"`
}

// AppDeploymentSpec defines the desired state of AppDeployment
type AppDeploymentSpec struct {
	// AppName is the name of the application from the catalog (validated at runtime against available charts)
	// +kubebuilder:validation:MinLength=1
	AppName string `json:"appName"`

	// ChartVersion is the specific chart version to deploy (defaults to latest)
	// +optional
	ChartVersion string `json:"chartVersion,omitempty"`

	// TeamID identifies the team owning this deployment
	TeamID string `json:"teamId"`

	// RequestedBy is the user ID who requested the deployment
	// +optional
	RequestedBy string `json:"requestedBy,omitempty"`

	// ReleaseName is the Helm release name (auto-generated if not specified)
	// +kubebuilder:validation:MaxLength=53
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +optional
	ReleaseName string `json:"releaseName,omitempty"`

	// Values are custom Helm values to override defaults
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Values *apiextensionsv1.JSON `json:"values,omitempty"`

	// ValuesFrom references ConfigMaps/Secrets for values
	// +optional
	ValuesFrom []ValuesReference `json:"valuesFrom,omitempty"`

	// AutoUpgrade enables automatic upgrades to new chart versions
	// +kubebuilder:default=false
	// +optional
	AutoUpgrade bool `json:"autoUpgrade,omitempty"`

	// Suspend stops reconciliation of this deployment
	// +kubebuilder:default=false
	// +optional
	Suspend bool `json:"suspend,omitempty"`
}

// AppDeploymentStatus defines the observed state of AppDeployment
type AppDeploymentStatus struct {
	// Phase is the current deployment phase
	// +kubebuilder:validation:Enum=Pending;Installing;Upgrading;Deployed;Failed;Uninstalling
	Phase AppDeploymentPhase `json:"phase,omitempty"`

	// HelmReleaseName is the actual Helm release name
	HelmReleaseName string `json:"helmReleaseName,omitempty"`

	// HelmReleaseRevision is the current revision
	HelmReleaseRevision int `json:"helmReleaseRevision,omitempty"`

	// DeployedChartVersion is the currently deployed version
	DeployedChartVersion string `json:"deployedChartVersion,omitempty"`

	// LastAttemptedChartVersion is the version last attempted
	LastAttemptedChartVersion string `json:"lastAttemptedChartVersion,omitempty"`

	// LastAppliedValuesHash is a hash of the last applied values
	LastAppliedValuesHash string `json:"lastAppliedValuesHash,omitempty"`

	// Conditions represent the latest available observations
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastReconcileTime is when reconciliation last occurred
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`

	// ObservedGeneration is the last observed generation
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// FailureCount is the number of consecutive failures
	FailureCount int `json:"failureCount,omitempty"`

	// Message provides human-readable status information
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="App",type=string,JSONPath=`.spec.appName`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.chartVersion`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Release",type=string,JSONPath=`.status.helmReleaseName`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// AppDeployment is the Schema for the appdeployments API
type AppDeployment struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of AppDeployment
	// +required
	Spec AppDeploymentSpec `json:"spec"`

	// status defines the observed state of AppDeployment
	// +optional
	Status AppDeploymentStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// AppDeploymentList contains a list of AppDeployment
type AppDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []AppDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AppDeployment{}, &AppDeploymentList{})
}
