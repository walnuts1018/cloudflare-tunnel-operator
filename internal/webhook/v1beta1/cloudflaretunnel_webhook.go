package v1beta1

import (
	"context"
	"fmt"
	"strconv"

	cftv1beta1 "github.com/walnuts1018/cloudflare-tunnel-operator/api/v1beta1"
	"github.com/walnuts1018/cloudflare-tunnel-operator/internal/consts"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
// var cloudflaretunnellog = logf.Log.WithName("cloudflaretunnel-resource")

// SetupCloudflareTunnelWebhookWithManager registers the webhook for CloudflareTunnel in the manager.
func SetupCloudflareTunnelWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&cftv1beta1.CloudflareTunnel{}).
		WithValidator(&CloudflareTunnelCustomValidator{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}).
		WithDefaulter(&CloudflareTunnelCustomDefaulter{}).
		Complete()
}

type CloudflareTunnelCustomDefaulter struct {
}

var _ webhook.CustomDefaulter = &CloudflareTunnelCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind CloudflareTunnel.
func (d *CloudflareTunnelCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	cloudflaretunnel, ok := obj.(*cftv1beta1.CloudflareTunnel)
	if !ok {
		return fmt.Errorf("expected an CloudflareTunnel object but got %T", obj)
	}

	if cloudflaretunnel.ObjectMeta.Labels == nil {
		cloudflaretunnel.ObjectMeta.Labels = make(map[string]string)
	}
	cloudflaretunnel.ObjectMeta.Labels[consts.DefaultLabelKey] = strconv.FormatBool(cloudflaretunnel.Spec.Default)

	return nil
}

// +kubebuilder:rbac:groups=cf-tunnel-operator.walnuts.dev,resources=cloudflaretunnels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:webhook:path=/validate-cf-tunnel-operator-walnuts-dev-v1beta1-cloudflaretunnel,mutating=false,failurePolicy=fail,sideEffects=None,groups=cf-tunnel-operator.walnuts.dev,resources=cloudflaretunnels,verbs=create;update,versions=v1beta1,name=vcloudflaretunnel-v1beta1.kb.io,admissionReviewVersions=v1

// CloudflareTunnelCustomValidator struct is responsible for validating the CloudflareTunnel resource
// when it is created, updated, or deleted.
type CloudflareTunnelCustomValidator struct {
	client.Client
	Scheme *runtime.Scheme
}

var _ webhook.CustomValidator = &CloudflareTunnelCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type CloudflareTunnel.
func (v *CloudflareTunnelCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cloudflaretunnel, ok := obj.(*cftv1beta1.CloudflareTunnel)
	if !ok {
		return nil, fmt.Errorf("expected a CloudflareTunnel object but got %T", obj)
	}
	return v.checkDefaultTunnel(ctx, *cloudflaretunnel)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type CloudflareTunnel.
func (v *CloudflareTunnelCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	cloudflaretunnel, ok := newObj.(*cftv1beta1.CloudflareTunnel)
	if !ok {
		return nil, fmt.Errorf("expected a CloudflareTunnel object for the newObj but got %T", newObj)
	}
	return v.checkDefaultTunnel(ctx, *cloudflaretunnel)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type CloudflareTunnel.
func (v *CloudflareTunnelCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// check that there are no multiple default tunnels
func (v *CloudflareTunnelCustomValidator) checkDefaultTunnel(ctx context.Context, cfTunnel cftv1beta1.CloudflareTunnel) (admission.Warnings, error) {
	// Defaultのトンネルではない
	if !cfTunnel.Spec.Default {
		return nil, nil
	}

	var defaultCFTunnels cftv1beta1.CloudflareTunnelList
	if err := v.List(ctx, &defaultCFTunnels, &client.ListOptions{
		Limit:         1,
		LabelSelector: labels.SelectorFromSet(map[string]string{consts.DefaultLabelKey: "true"}),
	}); err != nil {
		return nil, err
	}

	if len(defaultCFTunnels.Items) > 0 || defaultCFTunnels.Continue != "" {
		return admission.Warnings{fmt.Sprintf("default CloudflareTunnel already exists: %s/%s", defaultCFTunnels.Items[0].Namespace, defaultCFTunnels.Items[0].Name)}, nil
	}

	return nil, nil
}
