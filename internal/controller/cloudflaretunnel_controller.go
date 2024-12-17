/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"strconv"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	cftv1beta1 "github.com/walnuts1018/cloudflare-tunnel-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	appsv1apply "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	appName     = "cloudflared"
	managerName = "cloudflare-tunnel-operator"
	MetricsPort = 60123
)

// CloudflareTunnelReconciler reconciles a CloudflareTunnel object
type CloudflareTunnelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cf-tunnel-operator.walnuts.dev.cf-tunnel-operator.walnuts.dev,resources=cloudflaretunnels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cf-tunnel-operator.walnuts.dev.cf-tunnel-operator.walnuts.dev,resources=cloudflaretunnels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cf-tunnel-operator.walnuts.dev.cf-tunnel-operator.walnuts.dev,resources=cloudflaretunnels/finalizers,verbs=update

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitor,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CloudflareTunnel object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *CloudflareTunnelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var cfTunnel cftv1beta1.CloudflareTunnel

	err := r.Client.Get(ctx, req.NamespacedName, &cfTunnel)
	if errors.IsNotFound(err) {
		logger.Info("CloudflareTunnel has been deleted. Trying to delete its related resources.")
		return ctrl.Result{}, nil
	}
	if err != nil {
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get CloudflareTunnel.", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	if !cfTunnel.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	if err := r.reconcileDeployment(ctx, cfTunnel); err != nil {
		result, err2 := r.updateStatus(ctx, cfTunnel)
		if err2 != nil {
			logger.Error(err2, "Failed to update CloudflareTunnel status.", "name", req.Name, "namespace", req.Namespace)
		}
		return result, err
	}

	if err := r.reconcileService(ctx, cfTunnel); err != nil {
		result, err2 := r.updateStatus(ctx, cfTunnel)
		if err2 != nil {
			logger.Error(err2, "Failed to update CloudflareTunnel status.", "name", req.Name, "namespace", req.Namespace)
		}
		return result, err
	}

	if err := r.reconcileServiceMonitor(ctx, cfTunnel); err != nil {
		result, err2 := r.updateStatus(ctx, cfTunnel)
		if err2 != nil {
			logger.Error(err2, "Failed to update CloudflareTunnel status.", "name", req.Name, "namespace", req.Namespace)
		}
		return result, err
	}

	return r.updateStatus(ctx, cfTunnel)
}

func (r *CloudflareTunnelReconciler) reconcileDeployment(ctx context.Context, cfTunnel cftv1beta1.CloudflareTunnel) error {
	logger := log.FromContext(ctx)

	owner, err := controllerReference(cfTunnel, r.Scheme)
	if err != nil {
		return fmt.Errorf("failed to create controller reference: %w", err)
	}

	envs := []*corev1apply.EnvVarApplyConfiguration{
		corev1apply.EnvVar().
			WithName("TUNNEL_TOKEN").
			WithValueFrom(corev1apply.EnvVarSource().
				WithSecretKeyRef(corev1apply.SecretKeySelector().
					WithName(cfTunnel.Spec.Secret.Key).
					WithKey(cfTunnel.Spec.Secret.Name),
				),
			),
	}
	envs = append(envs, cfTunnel.Spec.ExtraEnv...)

	var resourceRequirements *corev1apply.ResourceRequirementsApplyConfiguration
	if cfTunnel.Spec.Resources.Limits != nil {
		cpu, ok := cfTunnel.Spec.Resources.Limits[corev1.ResourceCPU]
		if !ok {
			cpu = resource.Quantity{}
		}

		mem, ok := cfTunnel.Spec.Resources.Limits[corev1.ResourceMemory]
		if !ok {
			mem = resource.Quantity{}
		}

		resourceRequirements = resourceRequirements.WithLimits(corev1.ResourceList{
			corev1.ResourceCPU:    cpu,
			corev1.ResourceMemory: mem,
		})
	}

	if cfTunnel.Spec.Resources.Requests != nil {
		cpu, ok := cfTunnel.Spec.Resources.Requests[corev1.ResourceCPU]
		if !ok {
			cpu = resource.Quantity{}
		}

		mem, ok := cfTunnel.Spec.Resources.Requests[corev1.ResourceMemory]
		if !ok {
			mem = resource.Quantity{}
		}

		resourceRequirements = resourceRequirements.WithRequests(corev1.ResourceList{
			corev1.ResourceCPU:    cpu,
			corev1.ResourceMemory: mem,
		})
	}

	affinity := corev1apply.Affinity().
		WithPodAffinity(corev1apply.PodAffinity().
			WithPreferredDuringSchedulingIgnoredDuringExecution(corev1apply.WeightedPodAffinityTerm().
				WithWeight(100).
				WithPodAffinityTerm(corev1apply.PodAffinityTerm().
					WithTopologyKey("kubernetes.io/hostname").
					WithLabelSelector(metav1apply.LabelSelector().
						WithMatchLabels(map[string]string{
							"app.kubernetes.io/name": appName,
						}),
					),
				),
			),
		)

	if cfTunnel.Spec.Affinity != nil {
		affinity = cfTunnel.Spec.Affinity.Ref()
	}

	imagePullSecrets := make([]*corev1apply.LocalObjectReferenceApplyConfiguration, 0, len(cfTunnel.Spec.ImagePullSecrets))
	for _, secret := range cfTunnel.Spec.ImagePullSecrets {
		imagePullSecrets = append(imagePullSecrets, corev1apply.LocalObjectReference().
			WithName(secret.Name),
		)
	}

	args := []string{
		"--no-autoupdate",
		"--metrics=0.0.0.0:" + strconv.Itoa(MetricsPort),
		"tunnel",
		"run",
	}
	if cfTunnel.Spec.ArgsOverride != nil {
		args = cfTunnel.Spec.ArgsOverride
	}

	labels := labels(cfTunnel)
	deployment := appsv1apply.Deployment(cfTunnel.Name, cfTunnel.Namespace).
		WithLabels(labels).
		WithOwnerReferences(owner).
		WithSpec(appsv1apply.DeploymentSpec().
			WithReplicas(cfTunnel.Spec.Replicas).
			WithSelector(metav1apply.LabelSelector().WithMatchLabels(labels)).
			WithTemplate(corev1apply.PodTemplateSpec().
				WithLabels(labels).
				WithSpec(corev1apply.PodSpec().
					WithSecurityContext(corev1apply.PodSecurityContext().
						WithSysctls(corev1apply.Sysctl().
							WithName("net.ipv4.ping_group_range").
							WithValue("0 2147483647"),
						),
					).
					WithImagePullSecrets(imagePullSecrets...).
					WithContainers(corev1apply.Container().
						WithName("cloudflared").
						WithImage(cfTunnel.Spec.Image).
						WithImagePullPolicy(corev1.PullIfNotPresent).
						WithArgs(args...).
						WithEnv(envs...).
						WithPorts(corev1apply.ContainerPort().
							WithName("metrics").
							WithProtocol(corev1.ProtocolTCP).
							WithContainerPort(MetricsPort),
						).
						WithResources(resourceRequirements).
						WithSecurityContext(corev1apply.SecurityContext().
							WithReadOnlyRootFilesystem(true),
						).
						WithLivenessProbe(corev1apply.Probe().
							WithHTTPGet(corev1apply.HTTPGetAction().
								WithPath("/ready").
								WithPort(intstr.FromString("metrics")),
							).
							WithFailureThreshold(1).
							WithInitialDelaySeconds(10).
							WithPeriodSeconds(10),
						),
					).
					WithNodeSelector(cfTunnel.Spec.NodeSelector).
					WithTolerations(cfTunnel.Spec.Tolerations...).
					WithAffinity(affinity),
				),
			),
		)

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(deployment)
	if err != nil {
		return fmt.Errorf("failed to convert deployment to unstructured: %w", err)
	}

	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var current appsv1.Deployment
	err = r.Client.Get(ctx, client.ObjectKey{Namespace: cfTunnel.Namespace, Name: cfTunnel.Name}, &current)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	currentApplyConfig, err := appsv1apply.ExtractDeployment(&current, managerName)
	if err != nil {
		return fmt.Errorf("failed to extract apply configuration from deployment: %w", err)
	}

	if equality.Semantic.DeepEqual(deployment, currentApplyConfig) {
		return nil
	}

	if err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{FieldManager: managerName, Force: ptr.To(true)}); err != nil {
		return fmt.Errorf("failed to apply deployment: %w", err)
	}

	logger.Info("Deployment has been reconciled.", "name", cfTunnel.Name, "namespace", cfTunnel.Namespace)

	return nil
}

func (r *CloudflareTunnelReconciler) reconcileService(ctx context.Context, cfTunnel cftv1beta1.CloudflareTunnel) error {
	logger := log.FromContext(ctx)

	owner, err := controllerReference(cfTunnel, r.Scheme)
	if err != nil {
		return fmt.Errorf("failed to create controller reference: %w", err)
	}

	labels := labels(cfTunnel)

	service := corev1apply.Service(cfTunnel.Name, cfTunnel.Namespace).
		WithLabels(labels).
		WithOwnerReferences(owner).
		WithSpec(corev1apply.ServiceSpec().
			WithPorts(corev1apply.ServicePort().
				WithName("http").
				WithProtocol(corev1.ProtocolTCP).
				WithPort(MetricsPort).
				WithTargetPort(intstr.FromString("metrics")),
			).
			WithSelector(labels).
			WithType(corev1.ServiceTypeClusterIP),
		)

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(service)
	if err != nil {
		return fmt.Errorf("failed to convert service to unstructured: %w", err)
	}

	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var current corev1.Service
	err = r.Client.Get(ctx, client.ObjectKey{Namespace: cfTunnel.Namespace, Name: cfTunnel.Name}, &current)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get service: %w", err)
	}

	currentApplyConfig, err := corev1apply.ExtractService(&current, managerName)
	if err != nil {
		return fmt.Errorf("failed to extract apply configuration from service: %w", err)
	}

	if equality.Semantic.DeepEqual(service, currentApplyConfig) {
		return nil
	}

	if err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{FieldManager: managerName, Force: ptr.To(true)}); err != nil {
		return fmt.Errorf("failed to apply service: %w", err)
	}

	logger.Info("Service has been reconciled.", "name", cfTunnel.Name, "namespace", cfTunnel.Namespace)

	return nil
}

func (r *CloudflareTunnelReconciler) reconcileServiceMonitor(ctx context.Context, cfTunnel cftv1beta1.CloudflareTunnel) error {
	return nil
}

func (r *CloudflareTunnelReconciler) updateStatus(ctx context.Context, cfTunnel cftv1beta1.CloudflareTunnel) (ctrl.Result, error) {
	meta.SetStatusCondition(&cfTunnel.Status.Conditions, metav1.Condition{
		Type:   cftv1beta1.TypeCloudflareTunnelAvailable,
		Status: metav1.ConditionTrue,
		Reason: "OK",
	})
	meta.SetStatusCondition(&cfTunnel.Status.Conditions, metav1.Condition{
		Type:   cftv1beta1.TypeCloudflareTunnelDegraded,
		Status: metav1.ConditionFalse,
		Reason: "OK",
	})

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudflareTunnelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cftv1beta1.CloudflareTunnel{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&monitoringv1.ServiceMonitor{}).
		Named("cloudflaretunnel").
		Complete(r)
}

func controllerReference(cfTunnel cftv1beta1.CloudflareTunnel, scheme *runtime.Scheme) (*metav1apply.OwnerReferenceApplyConfiguration, error) {
	gvk, err := apiutil.GVKForObject(&cfTunnel, scheme)
	if err != nil {
		return nil, err
	}
	ref := metav1apply.OwnerReference().
		WithAPIVersion(gvk.GroupVersion().String()).
		WithKind(gvk.Kind).
		WithName(cfTunnel.Name).
		WithUID(cfTunnel.GetUID()).
		WithBlockOwnerDeletion(true).
		WithController(true)
	return ref, nil
}

func labels(cfTunnel cftv1beta1.CloudflareTunnel) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       appName,
		"app.kubernetes.io/instance":   cfTunnel.Name,
		"app.kubernetes.io/created-by": managerName,
	}
}
