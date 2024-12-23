package v1beta1

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	cftv1beta1 "github.com/walnuts1018/cloudflare-tunnel-operator/api/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("CloudflareTunnel Webhook", func() {
	var (
		initial   *cftv1beta1.CloudflareTunnel
		obj       *cftv1beta1.CloudflareTunnel
		oldObj    *cftv1beta1.CloudflareTunnel
		validator CloudflareTunnelCustomValidator
		defaulter CloudflareTunnelCustomDefaulter
	)

	BeforeEach(func() {
		initial = &cftv1beta1.CloudflareTunnel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default-tunnel",
				Namespace: "default",
				Labels: map[string]string{
					defaultKey: "true",
				},
			},
			Spec: cftv1beta1.CloudflareTunnelSpec{
				Default: true,
			},
		}
		obj = &cftv1beta1.CloudflareTunnel{}
		oldObj = &cftv1beta1.CloudflareTunnel{}
		validator = CloudflareTunnelCustomValidator{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		defaulter = CloudflareTunnelCustomDefaulter{}
		Expect(defaulter).NotTo(BeNil(), "Expected defaulter to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
		Expect(initial).NotTo(BeNil(), "Expected initial to be initialized")
	})

	AfterEach(func() {
		err := k8sClient.Delete(ctx, initial)
		if !apierrors.IsNotFound(err) {
			Expect(err).NotTo(HaveOccurred())
		}
	})

	Context("When creating CloudflareTunnel under Defaulting Webhook", func() {
		It(fmt.Sprintf("Default Tunnelのとき、%s Labelに%vが設定される", defaultKey, true), func() {
			By("objectのdefaultフィールドをtrueに設定")
			obj.Spec.Default = true
			By("calling the Default method to apply defaults")
			defaulter.Default(ctx, obj)
			By(fmt.Sprintf("Labelsに%sが設定されていることを確認", defaultKey))
			Expect(obj.ObjectMeta.Labels[defaultKey]).To(Equal("true"))
		})

		It(fmt.Sprintf("Default Tunnelでないとき、%s Labelに%vが設定される", defaultKey, false), func() {
			By("objectのdefaultフィールドをfalseに設定")
			obj.Spec.Default = false
			By("calling the Default method to apply defaults")
			defaulter.Default(ctx, obj)
			By(fmt.Sprintf("Labelsに%sが設定されていることを確認", defaultKey))
			Expect(obj.ObjectMeta.Labels[defaultKey]).To(Equal("false"))
		})
	})

	Context("When creating or updating CloudflareTunnel under Validating Webhook", func() {
		It("Default Tunnelが既に存在するとき、新しいTunnelはDefaultにできない", func() {
			By("Default Tunnelを作成する")
			Expect(k8sClient.Create(ctx, initial)).To(Succeed())

			By("新しくDefault Tunnelを作成すると、エラーが発生する")
			obj.Spec.Default = true
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).Should(ContainElement(ContainSubstring("already exists")))
		})

		It("Default Tunnelが存在しない時、新しいTunnelはDefaultにできる", func() {
			By("新しくDefault Tunnelを作成する")
			obj.Spec.Default = true
			Expect(validator.ValidateCreate(ctx, obj)).To(BeNil())
		})

		It("そもそも新しいTunnelがDefaultでない場合、エラーが発生しない", func() {
			By("Default Tunnelを作成する")
			Expect(k8sClient.Create(ctx, initial)).To(Succeed())

			By("新しくTunnelを作成する")
			obj.Spec.Default = false
			Expect(validator.ValidateCreate(ctx, obj)).To(BeNil())
		})
	})

})
