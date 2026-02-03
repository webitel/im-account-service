package grpc

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/webitel/im-account-service/infra/discovery/resolver"
	"github.com/webitel/webitel-go-kit/infra/discovery"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	// _ "google.golang.org/grpc/balancer/grpclb" // consul: [DNS] SRV record(s) ; port not involved  =((
	// _ "google.golang.org/grpc/balancer/roundrobin"
)

const (
	// see https://github.com/grpc/grpc/blob/master/doc/service_config.md to know more about service config
	retryPolicy string = ` {
		"loadBalancingPolicy": "round_robin",
		"loadBalancingConfig": [ { "round_robin": {} } ],
		"methodConfig": [
			{
				"timeout": "5.000000001s",
				"waitForReady": true,
				"retryPolicy": {
					"MaxAttempts": 4,
					"InitialBackoff": ".01s",
					"MaxBackoff": ".01s",
					"BackoffMultiplier": 1.0,
					"RetryableStatusCodes": [ "UNAVAILABLE" ]
				}
			}
		]
	}`
)

func NewServiceClient(logger *slog.Logger, registry discovery.DiscoveryProvider, secure *tls.Config, service string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {

	const scheme = "discovery://"

	target := service
	if !strings.HasPrefix(target, scheme) {
		// discovery:///service
		target = scheme + "/" + target
	}

	// target := "discovery:///" + service

	logger.Info(fmt.Sprintf("NewClient( %s )", target), slog.String("target", target))

	creds := insecure.NewCredentials()
	if secure != nil {
		creds = credentials.NewTLS(secure)
	}

	opts = append([]grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultServiceConfig(retryPolicy),
		grpc.WithResolvers(resolver.NewBuilder(
			registry,
			resolver.WithInsecure(true),
			resolver.PrintDebugLog(false),
			resolver.WithTimeout((time.Second * 5)),
		)),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}, opts...)

	client, err := grpc.NewClient(target, opts...)

	if err != nil {
		return nil, err
	}

	return client, nil
}
