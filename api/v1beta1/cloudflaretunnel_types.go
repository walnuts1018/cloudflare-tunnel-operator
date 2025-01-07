package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CloudflareTunnelSpec defines the desired state of CloudflareTunnel.
type CloudflareTunnelSpec struct {
	// Default specifies whether this tunnel should be the default tunnel in the cluster.
	// +kubebuilder:default=false
	// +optional
	Default bool `json:"default"`

	// Replicas is the number of cloudflared pods.
	// +kubebuilder:default=1
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Specifies the resource requirements for code server pod.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Specifies the image to use for the cloudflared pod.
	// +kubebuilder:default="cloudflare/cloudflared:2025.1.0"
	// +optional
	Image string `json:"image,omitempty"`

	// Specifies the image pull policy for the cloudflared pod.
	// +optional
	ArgsOverride []string `json:"argsOverride,omitempty"`

	// Specifies the image pull policy for the cloudflared pod.
	// +optional
	ExtraEnv EnvVarApplyConfigurationList `json:"extraEnv,omitempty"`

	// Specifies the node selector for scheduling.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Specifies the tolerations for scheduling.
	// +optional
	Tolerations TolerationApplyConfigurationList `json:"tolerations,omitempty"`

	// Specifies the affinity for scheduling.
	// +optional
	Affinity *AffinityApplyConfiguration `json:"affinity,omitempty"`

	// Specifies the service account name.
	// +kubebuilder:default=true
	EnableServiceMonitor bool `json:"enableServiceMonitor,omitempty"`

	// +optional
	Settings CloudflareTunnelSettings `json:"settings,omitempty"`
}

type CloudflareTunnelSettings struct {
	// +kubebuilder:default="http_status:404"
	// +optional
	CatchAllRule string `json:"catchAllRule,omitempty"`

	// Path to the certificate authority (CA) for the certificate of your origin. This option should be used only if your certificate is not signed by Cloudflare.
	// +optional
	CAPool *string `json:"caPool,omitempty"`

	// Disables TLS verification of the certificate presented by your origin. Will allow any certificate from the origin to be accepted.
	// +kubebuilder:default=false
	// +optional
	NoTLSVerify bool `json:"noTLSVerify,omitempty"`

	// Timeout for completing a TLS handshake to your origin server, if you have chosen to connect Tunnel to an HTTPS server.
	// +kubebuilder:default=10
	// +optional
	TLSTimeoutSeconds int32 `json:"tlsTimeoutSeconds,omitempty"`

	// Attempt to connect to origin using HTTP2. Origin must be configured as https.
	// +kubebuilder:default=false
	// +optional
	HTTP2Origin bool `json:"http2Origin,omitempty"`

	// Disables chunked transfer encoding. Useful if you are running a WSGI server.
	// +kubebuilder:default=false
	// +optional
	DisableChunkedEncoding bool `json:"disableChunkedEncoding,omitempty"`

	// Timeout for establishing a new TCP connection to your origin server. This excludes the time taken to establish TLS, which is controlled by tlsTimeout.
	// +kubebuilder:default=30
	// +optional
	ConnectTimeoutSeconds int32 `json:"connectTimeoutSeconds,omitempty"`

	// Disable the “happy eyeballs” algorithm for IPv4/IPv6 fallback if your local network has misconfigured one of the protocols.
	// +kubebuilder:default=false
	// +optional
	NoHappyEyeballs bool `json:"noHappyEyeballs,omitempty"`

	// cloudflared starts a proxy server to translate HTTP traffic into TCP when proxying, for example, SSH or RDP. This configures what type of proxy will be started. Valid options are: "" for the regular proxy and "socks" for a SOCKS5 proxy.
	// +optional
	ProxyType string `json:"proxyType,omitempty"`

	// Timeout after which an idle keepalive connection can be discarded.
	// +kubebuilder:default=90
	// +optional
	KeepAliveTimeoutSeconds int32 `json:"keepAliveTimeoutSeconds,omitempty"`

	// Maximum number of idle keepalive connections between Tunnel and your origin. This does not restrict the total number of concurrent connections.
	// +kubebuilder:default=100
	// +optional
	KeepAliveConnections int32 `json:"keepAliveConnections,omitempty"`
}

// CloudflareTunnelStatus defines the observed state of CloudflareTunnel.
type CloudflareTunnelStatus struct {
	// Replicas is copied from the underlying Deployment's status.replicas.
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// +optional
	TunnelName string `json:"tunnelName"`

	// +optional
	TunnelID string `json:"tunnelID"`

	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

const (
	TypeCloudflareTunnelAvailable = "Available"
	TypeCloudflareTunnelDegraded  = "Degraded"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="REPLICAS",type="string",JSONPath=".spec.replicas",description="Replica Count"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="DEFAULT",type="boolean",JSONPath=".spec.default",description="Default Tunnel"

// CloudflareTunnel is the Schema for the cloudflaretunnels API.
type CloudflareTunnel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudflareTunnelSpec   `json:"spec,omitempty"`
	Status CloudflareTunnelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudflareTunnelList contains a list of CloudflareTunnel.
type CloudflareTunnelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudflareTunnel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudflareTunnel{}, &CloudflareTunnelList{})
}
