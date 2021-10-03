/*
Copyright 2021.

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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OpenldapSpec defines the desired state of Openldap
type OpenldapSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Image to use
	Image string `json:"image"`

	// Size of the database storage in GB
	StorageSize resource.Quantity `json:"storage-size"`

	// Whether to delete the pvc
	DisposePVC bool `json:"dispose-pvc"`

	// +kubebuilder:validation:Pattern:=`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`
	LoadBalancerIPAddress string `json:"loadbalancer-ip-address"`

	// Stores the openldap configuration
	Config string `json:"config"`
}

// OpenldapStatus defines the observed state of Openldap
type OpenldapStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Node names of openldap pods
	Nodes []string `json:"nodes"`
}

// Openldap is the Schema for the openldaps API
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
type Openldap struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenldapSpec   `json:"spec,omitempty"`
	Status OpenldapStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenldapList contains a list of Openldap
type OpenldapList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Openldap `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Openldap{}, &OpenldapList{})
}
