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
			apiToken, ok := os.LookupEnv("CLOUDFLARE_API_TOKEN")
			if !ok {
				Fail("CLOUDFLARE_API_TOKEN is not set")
			}
			accountId, ok := os.LookupEnv("CLOUDFLARE_ACCOUNT_ID")
			if !ok {
				Fail("CLOUDFLARE_ACCOUNT_ID is not set")
			}

			zoneID, ok := os.LookupEnv("CLOUDFLARE_ZONE_ID")
			if !ok {
				Fail("CLOUDFLARE_ZONE_ID is not set")
			}

			var err error
			client, err = NewCloudflareTunnelClient(apiToken, accountId, zoneID, random.NewDummy())
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

			// By("Get Tunnel Token")
			// token, err := client.GetTunnelToken(ctx, tunnel.ID)
			// Expect(err).NotTo(HaveOccurred())
			// Expect(token).NotTo(BeEmpty())
		})
	})
})
