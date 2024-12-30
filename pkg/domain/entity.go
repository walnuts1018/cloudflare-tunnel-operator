package domain

import "github.com/cloudflare/cloudflare-go"

type CloudflareTunnel struct {
	ID   string
	Name string
}

type CloudflareTunnelToken string

type TunnelConfiguration cloudflare.TunnelConfiguration
