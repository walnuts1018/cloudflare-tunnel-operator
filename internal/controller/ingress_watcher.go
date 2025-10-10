package controller

import (
	"context"
	"errors"
	"fmt"
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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	annotationPrefix   = "cf-tunnel-operator.walnuts.dev/"
	cfTunnelAnnotation = annotationPrefix + "cloudflare-tunnel"
	ignoreAnnotation   = annotationPrefix + "ignore"
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

// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;update

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
		if controllerutil.ContainsFinalizer(ingress, finalizerName) {
			if err := r.finalizeIngress(ctx, ingress); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to finalize Ingress: %w", err)
			}

			controllerutil.RemoveFinalizer(ingress, finalizerName)
			if err := r.Update(ctx, ingress); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(ingress, finalizerName) {
		controllerutil.AddFinalizer(ingress, finalizerName)
		if err := r.Update(ctx, ingress); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer to Ingress: %w", err)
		}
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

	{
		hosts := getHosts(*ingress)

		// Ignore Annotationがついていたらエントリを追加しない
		// 既に追加されていたら削除する
		if checkToBeIgnored(ingress.Annotations) {
			if err := r.removeCloudflareTunnelConfig(ctx, tunnelID, hosts, cfTunnel.Spec.Settings); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update Cloudflare Tunnel config: %w", err)
			}

			for _, host := range hosts {
				if err := r.removeDNSRecord(ctx, tunnelID, host); err != nil {
					return ctrl.Result{}, fmt.Errorf("failed to update Cloudflare DNS Record: %w", err)
				}
			}
		} else {
			if err := r.appendCloudflareTunnelConfig(ctx, tunnelID, hosts, ip, cfTunnel.Spec.Settings); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update Cloudflare Tunnel config: %w", err)
			}

			for _, host := range hosts {
				if err := r.appendDNSRecord(ctx, tunnelID, host); err != nil {
					return ctrl.Result{}, fmt.Errorf("failed to update Cloudflare DNS Record: %w", err)
				}
			}
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

func (r *IngressReconciler) appendCloudflareTunnelConfig(
	ctx context.Context,
	tunnelID string,
	hosts []host,
	ip netip.Addr,
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
			OriginRequest: domain.ToOriginRequestConfig(tunnelSettings, rule.Hostname),
		}
	}

	for _, host := range hosts {
		rules[host.Host] = cloudflare.UnvalidatedIngressRule{
			Hostname:      host.Host,
			Path:          "",
			Service:       createEndpoint(ip, host.TLS),
			OriginRequest: domain.ToOriginRequestConfig(tunnelSettings, host.Host),
		}
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

func (r *IngressReconciler) removeCloudflareTunnelConfig(
	ctx context.Context,
	tunnelID string,
	hosts []host,
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
			OriginRequest: domain.ToOriginRequestConfig(tunnelSettings, rule.Hostname),
		}
	}

	for _, host := range hosts {
		delete(rules, host.Host)
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

func (r *IngressReconciler) appendDNSRecord(ctx context.Context, tunnelID string, host host) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, err := r.CloudflareTunnelManager.GetDNS(ctx, tunnelID, host.Host)
	if err != nil {
		return fmt.Errorf("failed to get DNS record: %v", err)
	}

	if record.ID == "" {
		if err := r.CloudflareTunnelManager.AddDNS(ctx, tunnelID, host.Host); err != nil {
			return fmt.Errorf("failed to add DNS record: %v", err)
		}
		return nil
	} else if !record.Healthy(tunnelID) {
		if err := r.CloudflareTunnelManager.UpdateDNS(ctx, tunnelID, host.Host, record); err != nil {
			return fmt.Errorf("failed to update DNS record: %v", err)
		}
	}
	return nil
}

func (r *IngressReconciler) removeDNSRecord(ctx context.Context, tunnelID string, host host) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, err := r.CloudflareTunnelManager.GetDNS(ctx, tunnelID, host.Host)
	if err != nil {
		return fmt.Errorf("failed to get DNS record: %v", err)
	}

	if record.ID == "" {
		return nil
	} else {
		if err := r.CloudflareTunnelManager.DeleteDNS(ctx, tunnelID, record.ID); err != nil {
			return fmt.Errorf("failed to delete DNS record: %v", err)
		}
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

func getHosts(ingress networkingv1.Ingress) []host {
	var TLSHosts []string
	for _, tls := range ingress.Spec.TLS {
		TLSHosts = append(TLSHosts, tls.Hosts...)
	}

	hosts := make([]host, 0, len(ingress.Spec.Rules))
	for _, rule := range ingress.Spec.Rules {
		hosts = append(hosts, host{
			Host: rule.Host,
			TLS:  checkTLS(rule.Host, TLSHosts),
		})
	}

	return hosts
}

func checkTLS(host string, TLSHosts []string) bool {
	return slices.Contains(TLSHosts, host)
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

func checkToBeIgnored(annotations map[string]string) bool {
	v, ok := annotations[ignoreAnnotation]
	return ok && (strings.ToLower(v) != "false" && v != "0")
}

func (r *IngressReconciler) finalizeIngress(ctx context.Context, ingress *networkingv1.Ingress) error {
	logger := log.FromContext(ctx)
	cfTunnelName, err := detectCloudflareTunnelName(ingress.Annotations)
	if err != nil {
		return fmt.Errorf("failed to detect Cloudflare Tunnel name: %w", err)
	}

	cfTunnel, err := r.getCloudflareTunnel(ctx, cfTunnelName)
	if err != nil {
		if errors.Is(err, ErrDefaultCloudflareTunnelNotExists) {
			logger.Info("default Cloudflare Tunnel not exists")
			return nil
		}
		return fmt.Errorf("failed to get Cloudflare Tunnel: %w", err)
	}

	// CloudflareTunnelリソースの削除時にトンネルやレコードが削除されるので、何もせずに抜ける
	if !cfTunnel.DeletionTimestamp.IsZero() {
		logger.Info("cloudflare tunnel is being deleted, skip removing ingress from the tunnel")
		return nil
	}

	tunnelID := cfTunnel.Status.TunnelID
	if tunnelID == "" {
		return fmt.Errorf("tunnel ID is empty")
	}

	hosts := getHosts(*ingress)
	if err := r.removeCloudflareTunnelConfig(ctx, tunnelID, hosts, cfTunnel.Spec.Settings); err != nil {
		return fmt.Errorf("failed to remove Cloudflare Tunnel config: %w", err)
	}
	for _, host := range hosts {
		if err := r.removeDNSRecord(ctx, tunnelID, host); err != nil {
			return fmt.Errorf("failed to delete Cloudflare Tunnel: %w", err)
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.Ingress{}).
		Complete(r)
}
