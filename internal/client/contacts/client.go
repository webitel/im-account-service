package contacts

import (

	// "google.golang.org/grpc"
	"log/slog"

	client_grpc "github.com/webitel/im-account-service/infra/client/grpc"
	"github.com/webitel/webitel-go-kit/infra/discovery"
	"google.golang.org/grpc"

	v1pb "github.com/webitel/im-account-service/proto/gen/im/service/contact/v1"
)

func NewClient(logger *slog.Logger, registry discovery.DiscoveryProvider, opts ...grpc.DialOption) (v1pb.ContactsClient, error) {

	const serviceName = "im-contact-service"

	client, err := client_grpc.NewServiceClient(
		logger, registry, serviceName, opts...,
	)

	if err != nil {
		return nil, err
	}

	return v1pb.NewContactsClient(client), nil
}
