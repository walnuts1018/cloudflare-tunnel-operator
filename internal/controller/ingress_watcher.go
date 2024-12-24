package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"

	cftv1beta1 "github.com/walnuts1018/cloudflare-tunnel-operator/api/v1beta1"
	"github.com/walnuts1018/cloudflare-tunnel-operator/internal/consts"
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
	ErrCloudflareTunnelNotFound = errors.New("Cloudflare Tunnel not found")
)

// IngressReconciler reconciles a Ingress object
type IngressReconciler struct {
	client.Client
	Scheme                  *runtime.Scheme
	CloudflareTunnelManager CloudflareTunnelManager
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
		return ctrl.Result{}, fmt.Errorf("failed to get Cloudflare Tunnel: %w", err)
	}

	if !cfTunnel.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, fmt.Errorf("cloudflare tunnel is being deleted: %w", ErrCloudflareTunnelNotFound)
	}

	if err := r.updateCloudflareTunnelConfig(ctx, cfTunnel, *ingress); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update Cloudflare Tunnel config: %w", err)
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
			return cftv1beta1.CloudflareTunnel{}, ErrCloudflareTunnelNotFound
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

func (r *IngressReconciler) updateCloudflareTunnelConfig(ctx context.Context, cfTunnel cftv1beta1.CloudflareTunnel, ingress networkingv1.Ingress) error {
	tunnelID := cfTunnel.Status.TunnelID
	if tunnelID == "" {
		return fmt.Errorf("tunnel ID is empty: %w", ErrCloudflareTunnelNotFound)
	}

	



	return nil
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
