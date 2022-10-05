/*
Copyright 2022 The Flux authors

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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SpecDefaultOpened string = "opened"
	SpecDefaultClosed string = "closed"
)

const (
	OpenedCondition string = "Opened"
)

const (
	ReconciliationSucceededReason string = "ReconciliationSucceeded"
	GateClosedReason              string = "GateClosed"
)

// GateSpec defines the desired state of Gate
type GateSpec struct {
	// Interval at which to check the Gate for updates.
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern="^([0-9]+(\\.[0-9]+)?(ms|s|m|h))+$"
	// +required
	Interval metav1.Duration `json:"interval"`

	// Default specifies the status of the Gate opened or closed.
	// Defaults to 'closed', valid values are ('opened', 'closed').
	// +kubebuilder:validation:Enum=opened;closed
	// +required
	Default string `json:"default"`

	// Window specifies the window automatically closes from
	// the time the Gate has been opened.
	// +kubebuilder:validation:Pattern="^([0-9]+(\\.[0-9]+)?(ms|s|m|h))+$"
	// +required
	Window string `json:"window"`
}

// GateStatus defines the observed state of Gate
type GateStatus struct {
	// RequestedAt specifies to close or open the gate ahead of its schedule.
	// +kubebuilder:validation:Format=date-time
	// +optional
	RequestedAt string `json:"requestedAt,omitempty"`

	// ResetToDefaultAt specifies to reset the status of the gate ahead of its schedule.
	// +kubebuilder:validation:Format=date-time
	// +optional
	ResetToDefaultAt string `json:"ResetToDefaultAt,omitempty"`

	// Conditions holds the conditions for the GitRepository.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// GetConditions returns the status conditions of the object.
func (in Gate) GetConditions() []metav1.Condition {
	return in.Status.Conditions
}

// SetConditions sets the status conditions on the object.
func (in *Gate) SetConditions(conditions []metav1.Condition) {
	in.Status.Conditions = conditions
}

// GetRequeueAfter returns the duration after which the Gate must be
// reconciled again.
func (in Gate) GetRequeueAfter() time.Duration {
	return in.Spec.Interval.Duration
}

//+genclient
//+genclient:Namespaced
//+kubebuilder:storageversion
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=gt
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description=""
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status",description=""
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].message",description=""

// Gate is the Schema for the gates API
type Gate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GateSpec   `json:"spec,omitempty"`
	Status GateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GateList contains a list of Gate
type GateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Gate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Gate{}, &GateList{})
}
