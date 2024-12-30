package external

import (
	"context"
	"fmt"

	cloudflare "github.com/cloudflare/cloudflare-go"
	cftv1beta1 "github.com/walnuts1018/cloudflare-tunnel-operator/api/v1beta1"
	"github.com/walnuts1018/cloudflare-tunnel-operator/pkg/domain"
	"github.com/walnuts1018/cloudflare-tunnel-operator/pkg/utils/random"
)

type CloudflareTunnelClient struct {
	client    *cloudflare.API
	accountId string

	random random.Random
}

func NewCloudflareTunnelClient(apiToken string, accountId string, random random.Random) (*CloudflareTunnelClient, error) {
	client, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloudflare provider: %v", err)
	}
	return &CloudflareTunnelClient{
		client:    client,
		accountId: accountId,
		random:    random,
	}, nil
}

func (c *CloudflareTunnelClient) CreateTunnel(ctx context.Context, name string) (domain.CloudflareTunnel, error) {
	secret, err := c.random.String(32, random.Alphanumeric)
	if err != nil {
		return domain.CloudflareTunnel{}, fmt.Errorf("failed to generate secret: %v", err)
	}

	t, err := c.client.CreateTunnel(ctx, cloudflare.AccountIdentifier(c.accountId), cloudflare.TunnelCreateParams{
		Name:      name,
		ConfigSrc: "cloudflare",
		Secret:    secret,
	})
	if err != nil {
		return domain.CloudflareTunnel{}, fmt.Errorf("failed to create tunnel: %v", err)
	}

	return domain.CloudflareTunnel{
		ID:   t.ID,
		Name: t.Name,
	}, nil
}

func (c *CloudflareTunnelClient) DeleteTunnel(ctx context.Context, id string) error {
	if err := c.client.DeleteTunnel(ctx, cloudflare.AccountIdentifier(c.accountId), id); err != nil {
		return fmt.Errorf("failed to delete tunnel: %v", err)
	}
	return nil
}

func (c *CloudflareTunnelClient) GetTunnelToken(ctx context.Context, tunnelId string) (domain.CloudflareTunnelToken, error) {
	t, err := c.client.GetTunnelToken(ctx, cloudflare.AccountIdentifier(c.accountId), tunnelId)
	if err != nil {
		return "", fmt.Errorf("failed to get tunnel token: %v", err)
	}
	return domain.CloudflareTunnelToken(t), nil
}

func (c *CloudflareTunnelClient) GetTunnel(ctx context.Context, id string) (domain.CloudflareTunnel, error) {
	t, err := c.client.GetTunnel(ctx, cloudflare.AccountIdentifier(c.accountId), id)
	if err != nil {
		return domain.CloudflareTunnel{}, fmt.Errorf("failed to get tunnel: %v", err)
	}

	return domain.CloudflareTunnel{
		ID:   t.ID,
		Name: t.Name,
	}, nil
}

func (c *CloudflareTunnelClient) DeleteToken(ctx context.Context, tunnelId string) error {
	return nil
}

func (c *CloudflareTunnelClient) AddPublicHost(ctx context.Context, tunnelId string, host string, tunnelSettings cftv1beta1.CloudflareTunnelSettings) error {
	return nil
}
