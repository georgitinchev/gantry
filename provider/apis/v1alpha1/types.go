package v1alpha1

import (
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ---- Network ----

type NetworkParameters struct {
	// CIDR to use. Leave empty to get the next free /24 from the cloud's pool.
	// +optional
	// +kubebuilder:validation:Pattern=`^([0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2}$`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="cidr is immutable"
	CIDR string `json:"cidr,omitempty"`
}

type NetworkObservation struct {
	ID   string `json:"id,omitempty"`
	CIDR string `json:"cidr,omitempty"`
}

type NetworkSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              NetworkParameters `json:"forProvider"`
}

type NetworkStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`
	AtProvider                 NetworkObservation `json:"atProvider,omitempty"`
}

// A Network is a private network in gantry-cloud.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gantry}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="CIDR",type="string",JSONPath=".status.atProvider.cidr"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
type Network struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              NetworkSpec   `json:"spec"`
	Status            NetworkStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type NetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Network `json:"items"`
}

// ---- Instance ----

type InstanceParameters struct {
	// Size of the instance. Changing it resizes the instance in place.
	// +kubebuilder:validation:Enum=small;medium;large
	Size string `json:"size"`

	// NetworkRef names a Network in the same namespace.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="networkRef is immutable"
	NetworkRef LocalRef `json:"networkRef"`
}

type LocalRef struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

type InstanceObservation struct {
	ID        string `json:"id,omitempty"`
	IP        string `json:"ip,omitempty"`
	State     string `json:"state,omitempty"`
	NetworkID string `json:"networkId,omitempty"`
	Size      string `json:"size,omitempty"`
}

type InstanceSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              InstanceParameters `json:"forProvider"`
}

type InstanceStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`
	AtProvider                 InstanceObservation `json:"atProvider,omitempty"`
}

// An Instance is a virtual machine in gantry-cloud.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,gantry}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="IP",type="string",JSONPath=".status.atProvider.ip"
type Instance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              InstanceSpec   `json:"spec"`
	Status            InstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Instance `json:"items"`
}

var (
	NetworkKind              = "Network"
	NetworkGroupVersionKind  = SchemeGroupVersion.WithKind(NetworkKind)
	InstanceKind             = "Instance"
	InstanceGroupVersionKind = SchemeGroupVersion.WithKind(InstanceKind)
)

func init() {
	SchemeBuilder.Register(&Network{}, &NetworkList{}, &Instance{}, &InstanceList{})
}
