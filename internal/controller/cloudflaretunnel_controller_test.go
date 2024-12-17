package controller

import (
	"context"
	"embed"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cftunneloperatorv1beta1 "github.com/walnuts1018/cloudflare-tunnel-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		BeforeEach(func() {
			By("Reading the CloudflareTunnel manifest & decoding it")
			cloudflareTunnelManifest, err := content.ReadFile("testdata/cf-tunnel-operator_v1beta1_cloudflaretunnel.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(yaml.Unmarshal(cloudflareTunnelManifest, cloudflareTunnel)).To(Succeed())

			namespacedName = types.NamespacedName{
				Name:      cloudflareTunnel.Name,
				Namespace: cloudflareTunnel.Namespace,
			}

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
			var dep *appsv1.Deployment
			Eventually(func() error {
				dep = &appsv1.Deployment{}
				return k8sClient.Get(ctx, client.ObjectKey{Name: namespacedName.Name, Namespace: namespacedName.Namespace}, dep)
			}).Should(Succeed())
			Expect(dep.OwnerReferences).To(HaveLen(1))
			Expect(dep.Spec.Replicas).NotTo(BeNil())
			Expect(*dep.Spec.Replicas).To(Equal(cloudflareTunnel.Spec.Replicas))
		})
	})
})
