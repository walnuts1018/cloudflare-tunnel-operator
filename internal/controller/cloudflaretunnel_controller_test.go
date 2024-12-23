package controller

import (
	"context"
	"embed"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	cftunneloperatorv1beta1 "github.com/walnuts1018/cloudflare-tunnel-operator/api/v1beta1"
	. "github.com/walnuts1018/cloudflare-tunnel-operator/test/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

//go:embed testdata
var content embed.FS

var _ = Describe("CloudflareTunnel Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()

		var namespacedName types.NamespacedName

		cloudflareTunnel := &cftunneloperatorv1beta1.CloudflareTunnel{}

		desiredDeployment := &appsv1.Deployment{}
		desiredService := &corev1.Service{}
		desiredServiceMonitor := &monitoringv1.ServiceMonitor{}

		BeforeEach(func() {
			By("Reading the CloudflareTunnel manifest & decoding it")
			cloudflareTunnelManifest, err := content.ReadFile("testdata/cf-tunnel-operator_v1beta1_cloudflaretunnel.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(yaml.Unmarshal(cloudflareTunnelManifest, cloudflareTunnel)).To(Succeed())

			namespacedName = types.NamespacedName{
				Name:      cloudflareTunnel.Name,
				Namespace: cloudflareTunnel.Namespace,
			}

			By("Reading the desired Deployment manifest & decoding it")
			desiredDeploymentManifest, err := content.ReadFile("testdata/deployment.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(yaml.Unmarshal(desiredDeploymentManifest, desiredDeployment)).To(Succeed())

			By("Reading the desired Service manifest & decoding it")
			desiredServiceManifest, err := content.ReadFile("testdata/service.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(yaml.Unmarshal(desiredServiceManifest, desiredService)).To(Succeed())

			By("Reading the desired ServiceMonitor manifest & decoding it")
			desiredServiceMonitorManifest, err := content.ReadFile("testdata/servicemonitor.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(yaml.Unmarshal(desiredServiceMonitorManifest, desiredServiceMonitor)).To(Succeed())

			By("creating the custom resource for the Kind CloudflareTunnel")
			err = k8sClient.Get(ctx, namespacedName, &cftunneloperatorv1beta1.CloudflareTunnel{})
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
			var gotDeployment *appsv1.Deployment
			Eventually(func() error {
				gotDeployment = &appsv1.Deployment{}
				return k8sClient.Get(ctx, namespacedName, gotDeployment)
			}).Should(Succeed())
			Expect(*gotDeployment).Should(HaveFields(*desiredDeployment))

			By("Check if Service is created")
			var gotService *corev1.Service
			Eventually(func() error {
				gotService = &corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Service",
					},
				}
				return k8sClient.Get(ctx, namespacedName, gotService)
			}).Should(Succeed())
			Expect(*gotService).Should(HaveFields(*desiredService))

			By("Check if ServiceMonitor is created")
			var gotServiceMonitor *monitoringv1.ServiceMonitor
			Eventually(func() error {
				gotServiceMonitor = &monitoringv1.ServiceMonitor{}
				return k8sClient.Get(ctx, namespacedName, gotServiceMonitor)
			}).Should(Succeed())
			Expect(*gotServiceMonitor).Should(HaveFields(*desiredServiceMonitor))
		})
	})
})
