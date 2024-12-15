package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="REPLICAS",type="string",JSONPath=".spec.replicas",description="Replica Count"
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// ClusterCloudflareTunnel is the Schema for the clustercloudflaretunnels API.
type ClusterCloudflareTunnel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudflareTunnelSpec   `json:"spec,omitempty"`
	Status CloudflareTunnelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterCloudflareTunnelList contains a list of ClusterCloudflareTunnel.
type ClusterCloudflareTunnelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterCloudflareTunnel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterCloudflareTunnel{}, &ClusterCloudflareTunnelList{})
}
