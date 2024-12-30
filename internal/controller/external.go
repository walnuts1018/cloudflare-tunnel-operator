package controller

import (
	"context"
	"errors"

	"github.com/walnuts1018/cloudflare-tunnel-operator/pkg/domain"
)

var (
	ErrTunnelNotFound = errors.New("tunnel not found")
)

//go:generate go run -mod=mod go.uber.org/mock/mockgen -source=external.go -destination=mock/mock.go

type CloudflareTunnelManager interface {
	CreateTunnel(ctx context.Context, Name string) (domain.CloudflareTunnel, error)
	DeleteTunnel(ctx context.Context, id string) error
	GetTunnel(ctx context.Context, ID string) (domain.CloudflareTunnel, error)
	GetTunnelToken(ctx context.Context, tunnelId string) (domain.CloudflareTunnelToken, error)
}
