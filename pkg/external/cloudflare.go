package external

import (
	"context"
	"encoding/json"
	"fmt"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/walnuts1018/cloudflare-tunnel-operator/pkg/domain"
	"github.com/walnuts1018/cloudflare-tunnel-operator/pkg/utils/random"
	"k8s.io/utils/ptr"
)

type CloudflareTunnelClient struct {
	client    *cloudflare.API
	accountId string
	zoneID    string
	random    random.Random
}

func NewCloudflareTunnelClient(apiToken string, accountId string, zoneID string, random random.Random) (*CloudflareTunnelClient, error) {
	client, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloudflare provider: %v", err)
	}
	return &CloudflareTunnelClient{
		client:    client,
		accountId: accountId,
		zoneID:    zoneID,
		random:    random,
	}, nil
}

func (c *CloudflareTunnelClient) CreateTunnel(ctx context.Context, name string) (domain.CloudflareTunnel, error) {
	secret, err := c.random.SecureString(32, random.Alphanumeric)
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

func (c *CloudflareTunnelClient) GetTunnelToken(ctx context.Context, tunnelID string) (domain.CloudflareTunnelToken, error) {
	t, err := c.client.GetTunnelToken(ctx, cloudflare.AccountIdentifier(c.accountId), tunnelID)
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

func (c *CloudflareTunnelClient) DeleteToken(ctx context.Context, tunnelID string) error {
	return nil
}

func (c *CloudflareTunnelClient) GetTunnelConfiguration(ctx context.Context, tunnelID string) (domain.TunnelConfiguration, error) {
	result, err := c.client.GetTunnelConfiguration(ctx, cloudflare.AccountIdentifier(c.accountId), tunnelID)
	if err != nil {
		return domain.TunnelConfiguration{}, fmt.Errorf("failed to get tunnel configs: %v", err)
	}
	return domain.TunnelConfiguration(result.Config), nil
}

func (c *CloudflareTunnelClient) UpdateTunnelConfiguration(ctx context.Context, tunnelID string, config domain.TunnelConfiguration) error {
	_, err := c.client.UpdateTunnelConfiguration(ctx, cloudflare.AccountIdentifier(c.accountId), cloudflare.TunnelConfigurationParams{
		TunnelID: tunnelID,
		Config:   cloudflare.TunnelConfiguration(config),
	})
	if err != nil {
		return fmt.Errorf("failed to update tunnel configs: %v", err)
	}
	return nil
}

const managedBy = "cloudflare-tunnel-operator"

type DNSComment struct {
	ManagedBy string `json:"managed-by"`
	TunnelID  string `json:"tunnelID"`
}

func (c *CloudflareTunnelClient) AddDNS(ctx context.Context, tunnelID string, hostname string) error {
	comment, err := json.Marshal(DNSComment{
		ManagedBy: managedBy,
		TunnelID:  tunnelID,
	})
	if err != nil {
		return fmt.Errorf("failed to generate comment: %v", err)
	}

	if _, err := c.client.CreateDNSRecord(ctx, cloudflare.ZoneIdentifier(c.zoneID), cloudflare.CreateDNSRecordParams{
		Name:    hostname,
		TTL:     1, // auto
		Proxied: ptr.To(true),
		Type:    "CNAME",
		Content: fmt.Sprintf("%v.cfargotunnel.com", tunnelID),
		Comment: string(comment),
	}); err != nil {
		return fmt.Errorf("failed to create DNS record: %v", err)
	}
	return nil
}

func (c *CloudflareTunnelClient) GetDNS(ctx context.Context, tunnelID string, hostname string) (domain.DNSRecord, error) {
	records, _, err := c.client.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(c.zoneID), cloudflare.ListDNSRecordsParams{
		Name: hostname,
		Type: "CNAME",
	})
	if err != nil {
		return domain.DNSRecord{}, fmt.Errorf("failed to get DNS record: %v", err)
	}

	if len(records) == 0 {
		return domain.DNSRecord{}, nil
	}

	return domain.DNSRecord(records[0]), nil
}

func (c *CloudflareTunnelClient) UpdateDNS(ctx context.Context, tunnelID string, hostname string, current domain.DNSRecord) error {
	comment, err := json.Marshal(DNSComment{
		ManagedBy: managedBy,
		TunnelID:  tunnelID,
	})
	if err != nil {
		return fmt.Errorf("failed to generate comment: %v", err)
	}

	if _, err := c.client.UpdateDNSRecord(ctx, cloudflare.ZoneIdentifier(c.zoneID), cloudflare.UpdateDNSRecordParams{
		ID:      current.ID,
		Name:    hostname,
		TTL:     1, // auto
		Proxied: ptr.To(true),
		Type:    "CNAME",
		Comment: ptr.To(string(comment)),
		Content: fmt.Sprintf("%v.cfargotunnel.com", tunnelID),
	}); err != nil {
		return fmt.Errorf("failed to update DNS record: %v", err)
	}
	return nil
}

func (c *CloudflareTunnelClient) DeleteDNS(ctx context.Context, tunnelID string, record domain.DNSRecord) error {
	if err := c.client.DeleteDNSRecord(ctx, cloudflare.ZoneIdentifier(c.zoneID), record.ID); err != nil {
		return fmt.Errorf("failed to delete DNS record: %v", err)
	}
	return nil
}

func (c *CloudflareTunnelClient) DeleteAllDNS(ctx context.Context, tunnelID string) error {
	comment, err := json.Marshal(DNSComment{
		ManagedBy: managedBy,
		TunnelID:  tunnelID,
	})
	if err != nil {
		return fmt.Errorf("failed to generate comment: %v", err)
	}

	records, _, err := c.client.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(c.zoneID), cloudflare.ListDNSRecordsParams{
		Type:    "CNAME",
		Comment: string(comment),
	})
	if err != nil {
		return fmt.Errorf("failed to get DNS record: %v", err)
	}

	if len(records) == 0 {
		return nil
	}

	for _, record := range records {
		if err := c.client.DeleteDNSRecord(ctx, cloudflare.ZoneIdentifier(c.zoneID), record.ID); err != nil {
			return fmt.Errorf("failed to update tunnel configs: %v", err)
		}
	}
	return nil
}
