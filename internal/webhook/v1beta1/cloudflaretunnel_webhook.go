package v1beta1

import (
	"context"
	"fmt"

	cftunneloperatorwalnutsdevv1beta1 "github.com/walnuts1018/cloudflare-tunnel-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// nolint:unused
// log is for logging in this package.
var cloudflaretunnellog = logf.Log.WithName("cloudflaretunnel-resource")

// SetupCloudflareTunnelWebhookWithManager registers the webhook for CloudflareTunnel in the manager.
func SetupCloudflareTunnelWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&cftunneloperatorwalnutsdevv1beta1.CloudflareTunnel{}).
		WithValidator(&CloudflareTunnelCustomValidator{}).
		WithDefaulter(&CloudflareTunnelCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-cf-tunnel-operator-walnuts-dev-cf-tunnel-operator-walnuts-dev-v1beta1-cloudflaretunnel,mutating=true,failurePolicy=fail,sideEffects=None,groups=cf-tunnel-operator.walnuts.dev.cf-tunnel-operator.walnuts.dev,resources=cloudflaretunnels,verbs=create;update,versions=v1beta1,name=mcloudflaretunnel-v1beta1.kb.io,admissionReviewVersions=v1

// CloudflareTunnelCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind CloudflareTunnel when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type CloudflareTunnelCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &CloudflareTunnelCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind CloudflareTunnel.
func (d *CloudflareTunnelCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	cloudflaretunnel, ok := obj.(*cftunneloperatorwalnutsdevv1beta1.CloudflareTunnel)

	if !ok {
		return fmt.Errorf("expected an CloudflareTunnel object but got %T", obj)
	}
	cloudflaretunnellog.Info("Defaulting for CloudflareTunnel", "name", cloudflaretunnel.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-cf-tunnel-operator-walnuts-dev-cf-tunnel-operator-walnuts-dev-v1beta1-cloudflaretunnel,mutating=false,failurePolicy=fail,sideEffects=None,groups=cf-tunnel-operator.walnuts.dev.cf-tunnel-operator.walnuts.dev,resources=cloudflaretunnels,verbs=create;update,versions=v1beta1,name=vcloudflaretunnel-v1beta1.kb.io,admissionReviewVersions=v1

// CloudflareTunnelCustomValidator struct is responsible for validating the CloudflareTunnel resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type CloudflareTunnelCustomValidator struct {
	//TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &CloudflareTunnelCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type CloudflareTunnel.
func (v *CloudflareTunnelCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cloudflaretunnel, ok := obj.(*cftunneloperatorwalnutsdevv1beta1.CloudflareTunnel)
	if !ok {
		return nil, fmt.Errorf("expected a CloudflareTunnel object but got %T", obj)
	}
	cloudflaretunnellog.Info("Validation for CloudflareTunnel upon creation", "name", cloudflaretunnel.GetName())

	

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type CloudflareTunnel.
func (v *CloudflareTunnelCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	cloudflaretunnel, ok := newObj.(*cftunneloperatorwalnutsdevv1beta1.CloudflareTunnel)
	if !ok {
		return nil, fmt.Errorf("expected a CloudflareTunnel object for the newObj but got %T", newObj)
	}
	cloudflaretunnellog.Info("Validation for CloudflareTunnel upon update", "name", cloudflaretunnel.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type CloudflareTunnel.
func (v *CloudflareTunnelCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cloudflaretunnel, ok := obj.(*cftunneloperatorwalnutsdevv1beta1.CloudflareTunnel)
	if !ok {
		return nil, fmt.Errorf("expected a CloudflareTunnel object but got %T", obj)
	}
	cloudflaretunnellog.Info("Validation for CloudflareTunnel upon deletion", "name", cloudflaretunnel.GetName())

	return nil, nil
}
