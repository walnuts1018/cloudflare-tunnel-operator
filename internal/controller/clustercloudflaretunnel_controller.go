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

	cftunneloperatorwalnutsdevv1beta1 "github.com/walnuts1018/cloudflare-tunnel-operator/api/v1beta1"
	"gorm.io/gorm/logger"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ClusterCloudflareTunnelReconciler reconciles a ClusterCloudflareTunnel object
type ClusterCloudflareTunnelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cf-tunnel-operator.walnuts.dev.cf-tunnel-operator.walnuts.dev,resources=clustercloudflaretunnels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cf-tunnel-operator.walnuts.dev.cf-tunnel-operator.walnuts.dev,resources=clustercloudflaretunnels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cf-tunnel-operator.walnuts.dev.cf-tunnel-operator.walnuts.dev,resources=clustercloudflaretunnels/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterCloudflareTunnel object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *ClusterCloudflareTunnelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var clusterCfTunnel cftunneloperatorwalnutsdevv1beta1.ClusterCloudflareTunnel

	err := r.Client.Get(ctx, req.NamespacedName, &clusterCfTunnel)
	if errors.IsNotFound(err) {
		logger.Info("ClusterCloudflareTunnel has been deleted. Trying to delete its related resources.")
		return ctrl.Result{}, nil
	}
	if err != nil {
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get ClusterCloudflareTunnel.", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, err
	}
	if !clusterCfTunnel.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	return r.updateStatus(ctx, clusterCfTunnel)
}

const secretKey = "token"

func (r *ClusterCloudflareTunnelReconciler) reconcileSecret(ctx context.Context, clusterCfTunnel cftunneloperatorwalnutsdevv1beta1.ClusterCloudflareTunnel) error {
	logger := log.FromContext(ctx)

	secret := &corev1.Secret{}
	secret.SetName(clusterCfTunnel.Name)

	token, err := r.getAPIToken(ctx, clusterCfTunnel.Spec.Secret)
	if err != nil {
		return fmt.Errorf("failed to get API token: %w", err)
	}

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, secret, func() error {
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}

		if _, ok := secret.Data[secretKey]; !ok {
			secret.Data[secretKey] = token
		}
		return ctrl.SetControllerReference(&clusterCfTunnel, secret, r.Scheme)
	})

	if err != nil {
		return fmt.Errorf("failed to reconcile secret: %w", err)
	}

	if op != controllerutil.OperationResultNone {
		logger.Info("Secret has been reconciled.", "name", secret.Name, "namespace", secret.Namespace)
	}

	return nil
}

// Managerと同じNamespaceに存在するSecretから、Cloudflare API Tokenを取得する
func (r *ClusterCloudflareTunnelReconciler) getAPIToken(ctx context.Context, cfTunnelSecretConfig cftunneloperatorwalnutsdevv1beta1.CloudflareTunnelSecret) ([]byte, error) {
	logger := log.FromContext(ctx)

	secret := &corev1.Secret{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: cfTunnelSecretConfig.Name}, secret); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Secret not found.", "name", cfTunnelSecretConfig.Name)
			return []byte{}, nil
		}
		return []byte{}, fmt.Errorf("failed to get secret: %w", err)
	}

	if secret.Data == nil {
		return []byte{}, fmt.Errorf("Secret %s has no data", cfTunnelSecretConfig.Name)
	}

	token, ok := secret.Data[cfTunnelSecretConfig.Key]
	if !ok {
		return []byte{}, fmt.Errorf("API Token not found in secret %s", cfTunnelSecretConfig.Name)
	}

	return token, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterCloudflareTunnelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cftunneloperatorwalnutsdevv1beta1.ClusterCloudflareTunnel{}).
		Named("clustercloudflaretunnel").
		Complete(r)
}
