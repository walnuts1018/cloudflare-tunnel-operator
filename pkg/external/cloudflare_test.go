package external

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/walnuts1018/cloudflare-tunnel-operator/pkg/utils/random"
)

var _ = Describe("Cloudflare", func() {
	Context("Default", func() {
		ctx := context.Background()

		var client *CloudflareTunnelClient

		BeforeEach(func() {
			apiToken, ok := os.LookupEnv("CF_API_TOKEN")
			if !ok {
				Fail("CF_API_TOKEN is not set")
			}
			accountId, ok := os.LookupEnv("CF_ACCOUNT_ID")
			if !ok {
				Fail("CF_ACCOUNT_ID is not set")
			}

			var err error
			client, err = NewCloudflareTunnelClient(apiToken, accountId, random.NewDummy())
			Expect(err).NotTo(HaveOccurred())
		})

		It("Normal", func() {
			By("Create Tunnel")
			tunnel, err := client.CreateTunnel(ctx, "cloudflare-tunnel-operator-test")
			Expect(err).NotTo(HaveOccurred())
			Expect(tunnel.ID).NotTo(BeEmpty())

			By("Get Tunnel")
			gotTunnel, err := client.GetTunnel(ctx, tunnel.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotTunnel.ID).To(Equal(tunnel.ID))
			Expect(gotTunnel.Name).To(Equal(tunnel.Name))

			By("Get Tunnel Token")
			token, err := client.GetTunnelToken(ctx, tunnel.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())
		})
	})
})
