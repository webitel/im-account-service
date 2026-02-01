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
	"github.com/webitel/im-account-service/infra/pubsub"
)

// Webitel Authorization (Service) Client
type Client struct {
	// options
	logger *slog.Logger
	broker pubsub.Provider
	// private
	cache simplelru.LRUCache[string, *v1pb.Userinfo]
	creds metadata.MD
	authz v1pb.AuthClient
}

func NewClient(

	logger *slog.Logger,
	registry discovery.DiscoveryProvider,
	broker pubsub.Provider,
	opts ...grpc.DialOption,

) (

	*Client, error,

) {

	const serviceName = "go.webitel.app"
	conn, err := client_grpc.NewServiceClient(
		logger, registry, serviceName, opts...,
	)

	if err != nil {
		return nil, err
	}

	// err = subscribeUpdates(broker)
	// if err != nil {
	// 	return nil, err
	// }

	client := &Client{
		logger: logger,
		broker: broker,
		cache:  expirable.NewLRU[string, *v1pb.Userinfo](0, nil, time.Minute),
		creds:  serviceClientCredentials(),
		authz:  v1pb.NewAuthClient(conn),
	}

	return client, client.Subscribe(broker)
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
