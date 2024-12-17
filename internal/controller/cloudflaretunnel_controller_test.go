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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cftunneloperatorv1beta1 "github.com/walnuts1018/cloudflare-tunnel-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("CloudflareTunnel Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		const namespace = "namespace"

		ctx := context.Background()

		namespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: namespace,
		}
		cloudflareTunnel := &cftunneloperatorv1beta1.CloudflareTunnel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: namespace,
			},
			Spec: cftunneloperatorv1beta1.CloudflareTunnelSpec{
				Replicas: 1,
				Secret: cftunneloperatorv1beta1.CloudflareTunnelSecret{
					Name: "test-secret",
					Key:  "test-key",
				},
			},
		}

		BeforeEach(func() {
			By("creating the custom resource for the Kind CloudflareTunnel")
			err := k8sClient.Get(ctx, namespacedName, &cftunneloperatorv1beta1.CloudflareTunnel{})
			if apierrors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, cloudflareTunnel)).To(Succeed())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		})
		AfterEach(func() {
			By("Cleanup the specific resource instance CloudflareTunnel")
			Expect(k8sClient.Delete(ctx, cloudflareTunnel)).NotTo(HaveOccurred())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &CloudflareTunnelReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Check if Deployment is created")
			var dep *appsv1.Deployment
			Eventually(func() error {
				dep = &appsv1.Deployment{}
				return k8sClient.Get(ctx, client.ObjectKey{Name: resourceName, Namespace: namespace}, dep)
			}).Should(Succeed())
			Expect(dep.OwnerReferences).To(HaveLen(1))
			Expect(dep.Spec.Replicas).NotTo(BeNil())
			Expect(*dep.Spec.Replicas).To(Equal(cloudflareTunnel.Spec.Replicas))

		})
	})
})
