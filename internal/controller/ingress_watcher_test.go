package controller

import (
	"context"
	"net/netip"
	"reflect"
	"slices"
	"sync"
	"testing"

	"github.com/cloudflare/cloudflare-go"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	cftv1beta1 "github.com/walnuts1018/cloudflare-tunnel-operator/api/v1beta1"
	mock_controller "github.com/walnuts1018/cloudflare-tunnel-operator/internal/controller/mock"
	"github.com/walnuts1018/cloudflare-tunnel-operator/pkg/domain"
	"go.uber.org/mock/gomock"
	networkingv1 "k8s.io/api/networking/v1"
)

var _ = Describe("Ingress Controller", func() {
	Context("When reconciling a resource", func() {

		It("should successfully reconcile the resource", func() {

			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})

func Test_getHosts(t *testing.T) {
	type args struct {
		ingress networkingv1.Ingress
	}
	tests := []struct {
		name string
		args args
		want []host
	}{
		{
			name: "normal",
			args: args{
				ingress: networkingv1.Ingress{
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{
								Host: "example1.walnuts.dev",
							},
							{
								Host: "example2.walnuts.dev",
							},
						},
						TLS: []networkingv1.IngressTLS{
							{
								Hosts: []string{"example.com"},
							},
							{
								Hosts: []string{"example1.walnuts.dev"},
							},
							{
								Hosts: []string{"test.example2.walnuts.dev"},
							},
						},
					},
				},
			},
			want: []host{
				{
					Host: "example1.walnuts.dev",
					TLS:  true,
				},
				{
					Host: "example2.walnuts.dev",
					TLS:  false,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slices.Collect(getHosts(tt.args.ingress))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getHosts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIngressReconciler_updateCloudflareTunnelConfig(t *testing.T) {
	ctx := context.Background()
	type state struct {
		Ingress []cloudflare.UnvalidatedIngressRule
	}

	type args struct {
		tunnelID string
		host     host
		IP       netip.Addr
	}
	tests := []struct {
		name      string
		state     state
		args      args
		wantState state
	}{
		{
			name: "normal",
			state: state{
				Ingress: []cloudflare.UnvalidatedIngressRule{
					{
						Hostname: "example1.walnuts.dev",
						Service:  "https://example1:80",
					},
					{
						Hostname: "example2.walnuts.dev",
						Service:  "https://example2:80",
					},
					{
						Hostname: "",
						Service:  "CatchAll",
					},
				},
			},
			args: args{
				tunnelID: "test",
				host: host{
					Host: "example3.walnuts.dev",
					TLS:  true,
				},
				IP: netip.MustParseAddr("192.168.0.1"),
			},
			wantState: state{
				Ingress: []cloudflare.UnvalidatedIngressRule{
					{
						Hostname: "example1.walnuts.dev",
						Service:  "https://example1:80",
					},
					{
						Hostname: "example2.walnuts.dev",
						Service:  "https://example2:80",
					},
					{
						Hostname: "example3.walnuts.dev",
						Service:  "https://192.168.0.1:443",
					},
					{
						Hostname: "",
						Service:  "CatchAll",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomockctrl := gomock.NewController(t)
			defer gomockctrl.Finish()

			mockCloudflareTunnelManager := mock_controller.NewMockCloudflareTunnelManager(gomockctrl)
			r := &IngressReconciler{
				CloudflareTunnelManager: mockCloudflareTunnelManager,
				mu:                      sync.Mutex{},
			}
			mockCloudflareTunnelManager.EXPECT().GetTunnelConfiguration(ctx, tt.args.tunnelID).Return(domain.TunnelConfiguration{
				Ingress: tt.state.Ingress,
			}, nil)
			mockCloudflareTunnelManager.EXPECT().UpdateTunnelConfiguration(ctx, tt.args.tunnelID, gomock.Any()).Return(nil).Do(
				func(ctx context.Context, tunnelID string, config domain.TunnelConfiguration) {
					for i, rule := range config.Ingress {
						assert.Equal(t, tt.wantState.Ingress[i].Hostname, rule.Hostname)
						assert.Equal(t, tt.wantState.Ingress[i].Service, rule.Service)
					}
				},
			)

			if err := r.updateCloudflareTunnelConfig(ctx, tt.args.tunnelID, tt.args.host, tt.args.IP, cftv1beta1.CloudflareTunnelSettings{
				CatchAllRule: "CatchAll",
			}); err != nil {
				t.Errorf("IngressReconciler.updateCloudflareTunnelConfig() error = %v", err)
			}
		})
	}
}
