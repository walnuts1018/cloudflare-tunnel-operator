package domain

import (
	"fmt"

	"github.com/cloudflare/cloudflare-go"
)

type CloudflareTunnel struct {
	ID   string
	Name string
}

type CloudflareTunnelToken string

type TunnelConfiguration cloudflare.TunnelConfiguration

type DNSRecord cloudflare.DNSRecord

func (d DNSRecord) Healthy(tunnelID string) bool {
	return d.ID != "" && d.Type == "CNAME" && d.Content == fmt.Sprintf("%v.cfargotunnel.com", tunnelID)
}
