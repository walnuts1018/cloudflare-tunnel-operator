package controller

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	cftv1beta1 "github.com/walnuts1018/cloudflare-tunnel-operator/api/v1beta1"
	"github.com/walnuts1018/cloudflare-tunnel-operator/pkg/domain"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	appsv1apply "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	appName        = "cloudflared"
	managerName    = "cloudflare-tunnel-operator"
	MetricsPort    = 60123
	tunnelTokenKey = "cloudflared-tunnel-token"
	finalizerName  = "cf-tunnel-operator.walnuts.dev/finalizer"
)

// CloudflareTunnelReconciler reconciles a CloudflareTunnel object
type CloudflareTunnelReconciler struct {
	client.Client
	Scheme                  *runtime.Scheme
	CloudflareTunnelManager CloudflareTunnelManager
}

// +kubebuilder:rbac:groups=cf-tunnel-operator.walnuts.dev,resources=cloudflaretunnels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cf-tunnel-operator.walnuts.dev,resources=cloudflaretunnels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cf-tunnel-operator.walnuts.dev,resources=cloudflaretunnels/finalizers,verbs=update

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;update;patch

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
	if apierrors.IsNotFound(err) {
		logger.Info("CloudflareTunnel has been deleted. Trying to delete its related resources.")
		return ctrl.Result{}, nil
	}
	if err != nil {
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get CloudflareTunnel.", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	if !cfTunnel.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&cfTunnel, finalizerName) {
			if err := r.CloudflareTunnelManager.DeleteAllDNS(ctx, cfTunnel.Status.TunnelID); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to delete CloudflareTunnel: %w", err)
			}

			if err := r.CloudflareTunnelManager.DeleteTunnel(ctx, cfTunnel.Status.TunnelID); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to delete Cloudflare Tunnel: %w", err)
			}

			controllerutil.RemoveFinalizer(&cfTunnel, finalizerName)
			err = r.Update(ctx, &cfTunnel)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(&cfTunnel, finalizerName) {
		controllerutil.AddFinalizer(&cfTunnel, finalizerName)
		err = r.Update(ctx, &cfTunnel)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	tunnel, token, err := r.reconcileTunnel(ctx, cfTunnel)
	if err != nil {
		result, err2 := r.updateStatus(ctx, cfTunnel)
		if err2 != nil {
			logger.Error(err2, "Failed to update CloudflareTunnel status.", "name", req.Name, "namespace", req.Namespace)
		}
		return result, err
	}

	secretName, err := r.reconcileSecret(ctx, cfTunnel, token)
	if err != nil {
		result, err2 := r.updateStatus(ctx, cfTunnel)
		if err2 != nil {
			logger.Error(err2, "Failed to update CloudflareTunnel status.", "name", req.Name, "namespace", req.Namespace)
		}
		return result, err
	}

	cfTunnel.Status.TunnelID = tunnel.ID
	cfTunnel.Status.TunnelName = tunnel.Name
	if err := r.Status().Update(ctx, &cfTunnel); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update CloudflareTunnel status: %w", err)
	}

	if err := r.reconcileDeployment(ctx, cfTunnel, secretName); err != nil {
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

func (r *CloudflareTunnelReconciler) reconcileTunnel(ctx context.Context, cfTunnel cftv1beta1.CloudflareTunnel) (domain.CloudflareTunnel, domain.CloudflareTunnelToken, error) {
	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{Namespace: cfTunnel.Namespace, Name: cfTunnel.Name}, &secret); err != nil {
		if !apierrors.IsNotFound(err) {
			return domain.CloudflareTunnel{}, "", fmt.Errorf("failed to get secret: %w", err)
		}
	}

	var token domain.CloudflareTunnelToken
	if secret.Data != nil {
		token = domain.CloudflareTunnelToken(secret.Data[tunnelTokenKey])
	}

	if cfTunnel.Status.TunnelID != "" {
		// TunnelIDがStatusに存在していて、Secretも存在している場合は、再利用する
		if token != "" {
			return domain.CloudflareTunnel{
				ID:   cfTunnel.Status.TunnelID,
				Name: cfTunnel.Name,
			}, token, nil
		} else {
			token, err := r.CloudflareTunnelManager.GetTunnelToken(ctx, cfTunnel.Status.TunnelID)
			if err == nil {
				return domain.CloudflareTunnel{
					ID:   cfTunnel.Status.TunnelID,
					Name: cfTunnel.Name,
				}, token, nil
			} else {
				if !errors.Is(err, ErrTunnelNotFound) {
					return domain.CloudflareTunnel{}, "", fmt.Errorf("failed to get tunnel: %w", err)
				}
			}
		}
	}

	// TunnelIDがStatusに存在していないので、Tunnelを作成する
	tunnel, err := r.CloudflareTunnelManager.CreateTunnel(ctx, cfTunnel.Name)
	if err != nil {
		return domain.CloudflareTunnel{}, "", fmt.Errorf("failed to create tunnel: %w", err)
	}

	token, err = r.CloudflareTunnelManager.GetTunnelToken(ctx, tunnel.ID)
	if err != nil {
		return domain.CloudflareTunnel{}, "", fmt.Errorf("failed to get tunnel token: %w", err)
	}

	return tunnel, token, nil
}

func (r *CloudflareTunnelReconciler) reconcileSecret(ctx context.Context, cfTunnel cftv1beta1.CloudflareTunnel, token domain.CloudflareTunnelToken) (types.NamespacedName, error) {
	logger := log.FromContext(ctx)

	namespacedName := types.NamespacedName{
		Namespace: cfTunnel.Namespace,
		Name:      cfTunnel.Name,
	}

	owner, err := controllerReference(cfTunnel, r.Scheme)
	if err != nil {
		return types.NamespacedName{}, fmt.Errorf("failed to create controller reference: %w", err)
	}

	labels := appLabels(cfTunnel)

	secret := corev1apply.Secret(namespacedName.Name, namespacedName.Namespace).
		WithLabels(labels).
		WithOwnerReferences(owner).
		WithData(
			map[string][]byte{
				tunnelTokenKey: []byte(token),
			},
		)

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(secret)
	if err != nil {
		return types.NamespacedName{}, fmt.Errorf("failed to convert secret to unstructured: %w", err)
	}

	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var current corev1.Service
	err = r.Client.Get(ctx, client.ObjectKey{Namespace: cfTunnel.Namespace, Name: cfTunnel.Name}, &current)
	if err != nil && !apierrors.IsNotFound(err) {
		return types.NamespacedName{}, fmt.Errorf("failed to get secret: %w", err)
	}

	currentApplyConfig, err := corev1apply.ExtractService(&current, managerName)
	if err != nil {
		return types.NamespacedName{}, fmt.Errorf("failed to extract apply configuration from secret: %w", err)
	}

	if equality.Semantic.DeepEqual(secret, currentApplyConfig) {
		return namespacedName, nil
	}

	if err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{FieldManager: managerName, Force: ptr.To(true)}); err != nil {
		return types.NamespacedName{}, fmt.Errorf("failed to apply secret: %w", err)
	}

	logger.Info("Secret has been reconciled.", "name", cfTunnel.Name, "namespace", cfTunnel.Namespace)

	return namespacedName, nil
}

func (r *CloudflareTunnelReconciler) reconcileDeployment(ctx context.Context, cfTunnel cftv1beta1.CloudflareTunnel, secretName types.NamespacedName) error {
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
					WithName(secretName.Name).
					WithKey(tunnelTokenKey),
				),
			),
	}
	envs = append(envs, cfTunnel.Spec.ExtraEnv...)

	resourceRequirements := corev1apply.ResourceRequirements()
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
		WithPodAntiAffinity(corev1apply.PodAntiAffinity().
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

	labels := appLabels(cfTunnel)
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

	if err != nil && !apierrors.IsNotFound(err) {
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

	labels := appLabels(cfTunnel)

	service := corev1apply.Service(cfTunnel.Name, cfTunnel.Namespace).
		WithLabels(labels).
		WithOwnerReferences(owner).
		WithSpec(corev1apply.ServiceSpec().
			WithPorts(corev1apply.ServicePort().
				WithName("metrics").
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
	if err != nil && !apierrors.IsNotFound(err) {
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
	logger := log.FromContext(ctx)

	if !cfTunnel.Spec.EnableServiceMonitor {
		return nil
	}

	sm := &monitoringv1.ServiceMonitor{}
	sm.SetNamespace(cfTunnel.Namespace)
	sm.SetName(cfTunnel.Name)
	sm.SetLabels(appLabels(cfTunnel))

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, sm, func() error {
		if sm.Spec.Selector.MatchLabels == nil {
			sm.Spec.Selector.MatchLabels = make(map[string]string)
		}
		for name, content := range appLabels(cfTunnel) {
			sm.Spec.Selector.MatchLabels[name] = content
		}

		if sm.Spec.Endpoints == nil {
			sm.Spec.Endpoints = make([]monitoringv1.Endpoint, 0)
		}

		sm.Spec.Endpoints = []monitoringv1.Endpoint{
			{
				Port:        "metrics",
				TargetPort:  ptr.To(intstr.FromString("metrics")),
				Path:        "/metrics",
				HonorLabels: false,
			},
		}

		sm.Spec.JobLabel = appName

		if sm.Spec.NamespaceSelector.MatchNames == nil {
			sm.Spec.NamespaceSelector.MatchNames = make([]string, 0, 1)
		}

		sm.Spec.NamespaceSelector.MatchNames = []string{cfTunnel.Namespace}

		return nil
	})

	if err != nil {
		logger.Error(err, "unable to create or update ServiceMonitor", "name", cfTunnel.Name, "namespace", cfTunnel.Namespace)
		return err
	}
	if op != controllerutil.OperationResultNone {
		logger.Info("reconcile ServiceMonitor", "operation", op)
	}
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

	if cfTunnel.Status.TunnelID == "" {
		meta.SetStatusCondition(&cfTunnel.Status.Conditions, metav1.Condition{
			Type:    cftv1beta1.TypeCloudflareTunnelDegraded,
			Status:  metav1.ConditionTrue,
			Reason:  "Reconciling",
			Message: "Cloudflare Tunnel is not ready yet",
		})
		meta.SetStatusCondition(&cfTunnel.Status.Conditions, metav1.Condition{
			Type:   cftv1beta1.TypeCloudflareTunnelAvailable,
			Status: metav1.ConditionFalse,
			Reason: "Reconciling",
		})
		return ctrl.Result{}, nil
	}

	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{Namespace: cfTunnel.Namespace, Name: cfTunnel.Name}, &secret); err != nil {
		if apierrors.IsNotFound(err) {
			meta.SetStatusCondition(&cfTunnel.Status.Conditions, metav1.Condition{
				Type:    cftv1beta1.TypeCloudflareTunnelDegraded,
				Status:  metav1.ConditionTrue,
				Reason:  "Reconciling",
				Message: "Secret not found",
			})
			meta.SetStatusCondition(&cfTunnel.Status.Conditions, metav1.Condition{
				Type:   cftv1beta1.TypeCloudflareTunnelAvailable,
				Status: metav1.ConditionFalse,
				Reason: "Reconciling",
			})
		} else {
			return ctrl.Result{}, err
		}
	}

	var svc corev1.Service
	if err := r.Get(ctx, client.ObjectKey{Namespace: cfTunnel.Namespace, Name: cfTunnel.Name}, &svc); err != nil {
		if apierrors.IsNotFound(err) {
			meta.SetStatusCondition(&cfTunnel.Status.Conditions, metav1.Condition{
				Type:    cftv1beta1.TypeCloudflareTunnelDegraded,
				Status:  metav1.ConditionTrue,
				Reason:  "Reconciling",
				Message: "Service not found",
			})
			meta.SetStatusCondition(&cfTunnel.Status.Conditions, metav1.Condition{
				Type:   cftv1beta1.TypeCloudflareTunnelAvailable,
				Status: metav1.ConditionFalse,
				Reason: "Reconciling",
			})
		} else {
			return ctrl.Result{}, err
		}
	}

	var dep appsv1.Deployment
	if err := r.Get(ctx, client.ObjectKey{Namespace: cfTunnel.Namespace, Name: cfTunnel.Name}, &dep); err != nil {
		if apierrors.IsNotFound(err) {
			meta.SetStatusCondition(&cfTunnel.Status.Conditions, metav1.Condition{
				Type:    cftv1beta1.TypeCloudflareTunnelDegraded,
				Status:  metav1.ConditionTrue,
				Reason:  "Reconciling",
				Message: "Deployment not found",
			})
			meta.SetStatusCondition(&cfTunnel.Status.Conditions, metav1.Condition{
				Type:   cftv1beta1.TypeCloudflareTunnelAvailable,
				Status: metav1.ConditionFalse,
				Reason: "Reconciling",
			})
		} else {
			return ctrl.Result{}, err
		}
	}

	result := ctrl.Result{}
	if dep.Status.AvailableReplicas == 0 {
		meta.SetStatusCondition(&cfTunnel.Status.Conditions, metav1.Condition{
			Type:    cftv1beta1.TypeCloudflareTunnelAvailable,
			Status:  metav1.ConditionFalse,
			Reason:  "Unavailable",
			Message: "AvailableReplicas is 0",
		})
		result = ctrl.Result{Requeue: true}
	}

	if err := r.Status().Update(ctx, &cfTunnel); err != nil {
		return ctrl.Result{}, err
	}

	return result, nil
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

func appLabels(cfTunnel cftv1beta1.CloudflareTunnel) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       appName,
		"app.kubernetes.io/instance":   cfTunnel.Name,
		"app.kubernetes.io/created-by": managerName,
	}
}
