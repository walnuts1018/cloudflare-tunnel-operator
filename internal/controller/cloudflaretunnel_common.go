package controller

import (
	"context"
	"fmt"

	cftunneloperatorwalnutsdevv1beta1 "github.com/walnuts1018/cloudflare-tunnel-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func updateStatus(ctx context.Context, codeServer cftunneloperatorwalnutsdevv1beta1.CloudflareTunnelStatus) (ctrl.Result, error) {
	var dep appsv1.Deployment
	err := r.Get(ctx, client.ObjectKey{Name: codeServer.Name, Namespace: codeServer.Namespace}, &dep)
	if err != nil {
		return ctrl.Result{}, err
	}

	var status csv1alpha2.CodeServerStatus
	if dep.Status.AvailableReplicas == 0 {
		status = csv1alpha2.CodeServerNotReady
	} else {
		status = csv1alpha2.CodeServerReady
	}

	if codeServer.Status != status {
		codeServer.Status = status
		err = r.Status().Update(ctx, &codeServer)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if codeServer.Status == csv1alpha2.CodeServerNotReady {
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{}, nil
}
