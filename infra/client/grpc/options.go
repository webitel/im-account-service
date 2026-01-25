package grpc

import (
	"context"
	"log/slog"

	"github.com/webitel/webitel-go-kit/infra/discovery"
	"go.uber.org/fx"
	"google.golang.org/grpc"
)

// Service gRPC Client Options
type Options struct {
	fx.In    // Params
	Target   string
	Logger   *slog.Logger
	Registry discovery.DiscoveryProvider
	Options  []grpc.DialOption
	Context  context.Context
}
