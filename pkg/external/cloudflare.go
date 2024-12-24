package external

import (
	"context"
	"fmt"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/walnuts1018/cloudflare-tunnel-operator/pkg/domain"
)

type CloudflareTunnelClient struct {
	client    *cloudflare.API
	accountId string
}

func NewCloudflareTunnelClient(apiToken string, accountId string) (*CloudflareTunnelClient, error) {
	client, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloudflare provider: %v", err)
	}
	return &CloudflareTunnelClient{client: client, accountId: accountId}, nil
}

func (c *CloudflareTunnelClient) CreateTunnel(ctx context.Context, name string) (domain.CloudflareTunnel, error) {
	t, err := c.client.CreateTunnel(ctx, cloudflare.AccountIdentifier(c.accountId), cloudflare.TunnelCreateParams{
		Name:      name,
		ConfigSrc: "cloudflare",
	})
	if err != nil {
		return domain.CloudflareTunnel{}, fmt.Errorf("failed to create tunnel: %v", err)
	}
	return domain.CloudflareTunnel{
		ID:     t.ID,
		Name:   t.Name,
		Secret: t.Secret,
	}, nil
}

func (c *CloudflareTunnelClient) GetTunnel(ctx context.Context, id string) (domain.CloudflareTunnel, error) {
	t, err := c.client.GetTunnel(ctx, cloudflare.AccountIdentifier(c.accountId), id)
	if err != nil {
		return domain.CloudflareTunnel{}, fmt.Errorf("failed to get tunnel: %v", err)
	}
	return domain.CloudflareTunnel{
		ID:     t.ID,
		Name:   t.Name,
		Secret: t.Secret,
	}, nil
}
