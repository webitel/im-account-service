package auth

import (
	"log/slog"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/hashicorp/golang-lru/v2/simplelru"
	v1pb "github.com/webitel/im-account-service/internal/client/webitel/proto/gen/auth"
	"github.com/webitel/webitel-go-kit/infra/discovery"

	"google.golang.org/grpc"
	_ "google.golang.org/grpc/balancer/grpclb" // consul: [DNS] SRV record(s)
	"google.golang.org/grpc/metadata"

	// "google.golang.org/grpc"
	client_grpc "github.com/webitel/im-account-service/infra/client/grpc"
)

// Webitel Authorization (Service) Client
type Client struct {
	authz v1pb.AuthClient
	cache simplelru.LRUCache[string, *v1pb.Userinfo]
	creds metadata.MD
}

func NewClient(logger *slog.Logger, registry discovery.DiscoveryProvider, opts ...grpc.DialOption) (*Client, error) {

	const serviceName = "go.webitel.app"
	client, err := client_grpc.NewServiceClient(
		logger, registry, serviceName, opts...,
	)

	if err != nil {
		return nil, err
	}

	return &Client{
		authz: v1pb.NewAuthClient(client),
		cache: expirable.NewLRU[string, *v1pb.Userinfo](0, nil, time.Minute),
		creds: serviceClientCredentials(),
	}, nil
}

func serviceClientCredentials() metadata.MD {

	// var (
	// 	// micro    *debug.Module
	// 	gomod, _ = debug.ReadBuildInfo()
	// 	// svhost   = micro_server.DefaultServer.Options()
	// )
	// resolve dependencies ..
	// for _, dep := range gomod.Deps {
	// 	if strings.HasPrefix(dep.Path, "go-micro.dev/v") {
	// 		micro = dep
	// 		break
	// 	}
	// }
	// cutprefix := func(prefix, vs string) string {
	// 	vs, _ = strings.CutPrefix(vs, prefix)
	// 	return vs
	// }
	return metadata.New(map[string]string{
		"from-service":    "im-account-service", // svhost.Name,
		"from-service-id": "development",        // svhost.Id,
		// "user-agent": fmt.Sprintf(
		// 	"%s/%s golang/%v micro/%s grpc/%s",
		// 	svhost.Name, svhost.Version,
		// 	cutprefix("go", gomod.GoVersion),
		// 	cutprefix("v", micro.Version),
		// 	grpc.Version,
		// ),
	})
}
