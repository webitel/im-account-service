package v1

import (
	"context"
	"log/slog"

	grpcsrv "github.com/webitel/im-account-service/infra/server/grpc"
	"github.com/webitel/im-account-service/internal/model"
	"github.com/webitel/im-account-service/internal/store"
	impb "github.com/webitel/im-account-service/proto/gen/im/service/admin/v1"
)

type ApplicationService struct {
	impb.UnimplementedApplicationsServer

	store  store.AppStore
	logger *slog.Logger
}

var _ impb.ApplicationsServer = (*ApplicationService)(nil)

func NewApplicationService(storage store.AppStore, logger *slog.Logger) *ApplicationService {
	return &ApplicationService{store: storage, logger: logger}
}

func RegisterApplicationService(server *grpcsrv.Server, handler *ApplicationService) {
	impb.RegisterApplicationsServer(server.Server, handler)
}

// ------------------------------- [API] v1 ---------------------------------------- //

// func (c *ApplicationService) mustEmbedUnimplementedApplicationsServer() {}

// Get Application(s) list
func (c *ApplicationService) SearchApps(ctx context.Context, req *impb.SearchAppRequest) (*impb.ApplicationList, error) {
	// return c.UnimplementedApplicationsServer.SearchApps(ctx, req)

	list, err := c.store.Search(store.SearchAppRequest{
		Context: ctx,
		Dc:      req.GetDc(),
		Id:      req.GetId(),
		Page:    int(req.GetPage()),
		Size:    int(req.GetSize()),
	})

	if err != nil {
		return nil, err
	}

	res := &impb.ApplicationList{
		Data: make([]*impb.Application, 0, len(list.Data)),
		Page: max(1, req.GetPage()),
		Next: (list.Next != nil),
	}

	for _, row := range list.Data {
		res.Data = append(res.Data, row.Proto())
	}

	return res, nil
}

func (c *ApplicationService) DeleteApps(ctx context.Context, req *impb.DeleteAppRequest) (*impb.ApplicationList, error) {
	return c.UnimplementedApplicationsServer.DeleteApps(ctx, req)
}

func (c *ApplicationService) CreateApp(ctx context.Context, req *impb.CreateAppRequest) (*impb.Application, error) {
	// return c.UnimplementedApplicationsServer.CreateApp(ctx, req)

	input := req.GetApp()
	src := model.NewApplication(input)

	// app := &impb.Application{
	// 	Dc:       input.GetDc(),
	// 	Id:       uuid.NewString(),
	// 	Name:     input.GetName(),
	// 	About:    input.GetAbout(),
	// 	Block:    nil, // &impb.Revocation{},
	// 	Client:   input.GetClient(),
	// 	Service:  input.GetService(), // LIMIT, UPDATES, PUSH
	// 	Account:  nil,                // &impb.Account{},
	// 	Contacts: input.GetContacts(),
	// }

	app, err := c.store.Create(
		store.CreateAppRequest{
			Context: ctx,
			App:     src,
		},
	)

	if err != nil {
		return nil, err
	}

	return app.Proto(), nil
}

func (c *ApplicationService) UpdateApp(ctx context.Context, req *impb.UpdateAppRequest) (*impb.Application, error) {
	return c.UnimplementedApplicationsServer.UpdateApp(ctx, req)
}
