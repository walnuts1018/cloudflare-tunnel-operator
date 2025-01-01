package domain

import (
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
	cftv1beta1 "github.com/walnuts1018/cloudflare-tunnel-operator/api/v1beta1"
	"k8s.io/utils/ptr"
)

func ToOriginRequestConfig(tunnelSettings cftv1beta1.CloudflareTunnelSettings, hostname string) *cloudflare.OriginRequestConfig {
	return &cloudflare.OriginRequestConfig{
		HTTPHostHeader:         ptr.To(hostname),
		OriginServerName:       ptr.To(hostname),
		CAPool:                 tunnelSettings.CAPool,
		NoTLSVerify:            ptr.To(tunnelSettings.NoTLSVerify),
		TLSTimeout:             ptr.To(cloudflare.TunnelDuration{Duration: time.Duration(tunnelSettings.TLSTimeoutSeconds) * time.Second}),
		Http2Origin:            ptr.To(tunnelSettings.HTTP2Origin),
		DisableChunkedEncoding: ptr.To(tunnelSettings.DisableChunkedEncoding),
		ConnectTimeout:         ptr.To(cloudflare.TunnelDuration{Duration: time.Duration(tunnelSettings.ConnectTimeoutSeconds) * time.Second}),
		NoHappyEyeballs:        ptr.To(tunnelSettings.NoHappyEyeballs),
		ProxyType:              ptr.To(tunnelSettings.ProxyType),
		KeepAliveTimeout:       ptr.To(cloudflare.TunnelDuration{Duration: time.Duration(tunnelSettings.KeepAliveTimeoutSeconds) * time.Second}),
		KeepAliveConnections:   ptr.To(int(tunnelSettings.KeepAliveConnections)),
	}
}
