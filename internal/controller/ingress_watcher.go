package controller

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"maps"
	"net/netip"
	"slices"
	"strings"
	"sync"

	"github.com/cloudflare/cloudflare-go"
	cftv1beta1 "github.com/walnuts1018/cloudflare-tunnel-operator/api/v1beta1"
	"github.com/walnuts1018/cloudflare-tunnel-operator/internal/consts"
	"github.com/walnuts1018/cloudflare-tunnel-operator/pkg/domain"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	annotationPrefix   = "cf-tunnel-operator.walnuts.dev/"
	cfTunnelAnnotation = annotationPrefix + "cloudflare-tunnel"
)

var (
	ErrCloudflareTunnelNotFound         = errors.New("cloudflare tunnel not found")
	ErrDefaultCloudflareTunnelNotExists = errors.New("default cloudflare tunnel not exists")
)

// IngressReconciler reconciles a Ingress object
type IngressReconciler struct {
	client.Client
	Scheme                  *runtime.Scheme
	CloudflareTunnelManager CloudflareTunnelManager
	mu                      sync.Mutex
}

// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch

func (r *IngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	ingress := &networkingv1.Ingress{}
	if err := r.Get(ctx, req.NamespacedName, ingress); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("ingress resource not found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get Ingress: %w", err)
	}

	if !ingress.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	cfTunnelName, err := detectCloudflareTunnelName(ingress.Annotations)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to detect Cloudflare Tunnel name: %w", err)
	}

	cfTunnel, err := r.getCloudflareTunnel(ctx, cfTunnelName)
	if err != nil {
		if errors.Is(err, ErrDefaultCloudflareTunnelNotExists) {
			logger.Info("default Cloudflare Tunnel not exists")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get Cloudflare Tunnel: %w", err)
	}

	if !cfTunnel.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, fmt.Errorf("cloudflare tunnel is being deleted: %w", ErrCloudflareTunnelNotFound)
	}

	tunnelID := cfTunnel.Status.TunnelID
	if tunnelID == "" {
		return ctrl.Result{}, fmt.Errorf("tunnel ID is empty")
	}

	if len(ingress.Status.LoadBalancer.Ingress) == 0 {
		return ctrl.Result{}, fmt.Errorf("ingress status load balancer ingress is empty")
	}

	ip, err := netip.ParseAddr(ingress.Status.LoadBalancer.Ingress[0].IP)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to parse IP: %w", err)
	}

	for host := range getHosts(*ingress) {
		if err := r.updateCloudflareTunnelConfig(ctx, tunnelID, host, ip, cfTunnel.Spec.Settings); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update Cloudflare Tunnel config: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *IngressReconciler) getCloudflareTunnel(ctx context.Context, cfTunnelName types.NamespacedName) (cftv1beta1.CloudflareTunnel, error) {
	var cfTunnel cftv1beta1.CloudflareTunnel

	// use default Cloudflare Tunnel
	if cfTunnelName.Name == "" || cfTunnelName.Namespace == "" {
		var defaultCFTunnels cftv1beta1.CloudflareTunnelList
		if err := r.List(ctx, &defaultCFTunnels, &client.ListOptions{
			Limit:         1,
			LabelSelector: labels.SelectorFromSet(map[string]string{consts.DefaultLabelKey: "true"}),
		}); err != nil {
			return cftv1beta1.CloudflareTunnel{}, fmt.Errorf("failed to list default Cloudflare Tunnel: %w", err)
		}

		if len(defaultCFTunnels.Items) == 0 {
			return cftv1beta1.CloudflareTunnel{}, ErrDefaultCloudflareTunnelNotExists
		}

		cfTunnel = defaultCFTunnels.Items[0]
	} else {
		if err := r.Get(ctx, cfTunnelName, &cfTunnel); err != nil {
			if apierrors.IsNotFound(err) {
				return cftv1beta1.CloudflareTunnel{}, ErrCloudflareTunnelNotFound
			}

			return cftv1beta1.CloudflareTunnel{}, fmt.Errorf("failed to get Cloudflare Tunnel: %w", err)
		}
	}

	return cfTunnel, nil
}

func (r *IngressReconciler) updateCloudflareTunnelConfig(
	ctx context.Context,
	tunnelID string,
	host host,
	IP netip.Addr,
	tunnelSettings cftv1beta1.CloudflareTunnelSettings,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	config, err := r.CloudflareTunnelManager.GetTunnelConfiguration(ctx, tunnelID)
	if err != nil {
		return fmt.Errorf("failed to get tunnel configs: %v", err)
	}
	rules := make(map[string]cloudflare.UnvalidatedIngressRule, len(config.Ingress))
	for _, rule := range config.Ingress {
		rules[rule.Hostname] = cloudflare.UnvalidatedIngressRule{
			Hostname:      rule.Hostname,
			Path:          rule.Path,
			Service:       rule.Service,
			OriginRequest: domain.ToOriginRequestConfig(tunnelSettings),
		}
	}

	rules[host.Host] = cloudflare.UnvalidatedIngressRule{
		Hostname:      host.Host,
		Path:          "",
		Service:       createEndpoint(IP, host.TLS),
		OriginRequest: domain.ToOriginRequestConfig(tunnelSettings),
	}

	rules[""] = cloudflare.UnvalidatedIngressRule{
		Hostname:      "",
		Path:          "",
		Service:       tunnelSettings.CatchAllRule,
		OriginRequest: nil,
	}

	config.Ingress = slices.SortedFunc(maps.Values(rules), func(a, b cloudflare.UnvalidatedIngressRule) int {
		switch {
		case a.Hostname == "":
			return 1
		case b.Hostname == "":
			return -1
		default:
			return strings.Compare(a.Hostname, b.Hostname)
		}
	})

	if err := r.CloudflareTunnelManager.UpdateTunnelConfiguration(ctx, tunnelID, config); err != nil {
		return fmt.Errorf("failed to update tunnel configs: %v", err)
	}

	return nil
}

func createEndpoint(IP netip.Addr, TLS bool) string {
	if TLS {
		return "https://" + IP.String() + ":443"
	} else {
		return "http://" + IP.String() + ":80"
	}
}

type host struct {
	Host string
	TLS  bool
}

func getHosts(ingress networkingv1.Ingress) iter.Seq[host] {
	var TLSHosts []string
	for _, tls := range ingress.Spec.TLS {
		TLSHosts = append(TLSHosts, tls.Hosts...)
	}

	return func(yield func(host) bool) {
		for _, rule := range ingress.Spec.Rules {
			if !yield(host{
				Host: rule.Host,
				TLS:  checkTLS(rule.Host, TLSHosts),
			}) {
				break
			}
		}
	}
}

func checkTLS(host string, TLSHosts []string) bool {
	for _, tlshost := range TLSHosts {
		if host == tlshost {
			return true
		}
	}
	return false
}

func detectCloudflareTunnelName(annotations map[string]string) (types.NamespacedName, error) {
	v := annotations[cfTunnelAnnotation]
	if v == "" {
		return types.NamespacedName{}, nil
	}

	values := strings.Split(v, "/")

	if len(values) != 2 {
		return types.NamespacedName{}, fmt.Errorf("invalid annotation value: %s, expected format: namespace/name", annotations[cfTunnelAnnotation])
	}

	return types.NamespacedName{
		Namespace: values[0],
		Name:      values[1],
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.Ingress{}).
		Complete(r)
}
