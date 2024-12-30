package controller

import (
	"reflect"
	"testing"

	. "github.com/onsi/ginkgo/v2"
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
		name    string
		args    args
		want    []host
		wantErr bool
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
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getHosts(tt.args.ingress)
			if (err != nil) != tt.wantErr {
				t.Errorf("getHosts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getHosts() = %v, want %v", got, tt.want)
			}
		})
	}
}
