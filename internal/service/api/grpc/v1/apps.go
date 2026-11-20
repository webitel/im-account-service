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
		res.Data = append(res.Data, maskAppSecrets(row.Proto()))
	}

	return res, nil
}

// Mutates app in place, so it must be a copy (row.Proto() clones).
// Push credentials and send_update.token are kept on purpose: im-delivery-service
// reads them from SearchApps to deliver notifications.
func maskAppSecrets(app *impb.Application) *impb.Application {
	if c := app.GetCaptcha(); c != nil {
		c.Secret = ""
	}

	if s := app.GetService(); s != nil {
		s.Secret = ""
	}

	return app
}

func (c *ApplicationService) DeleteApps(ctx context.Context, req *impb.DeleteAppRequest) (*impb.ApplicationList, error) {
	return c.UnimplementedApplicationsServer.DeleteApps(ctx, req)
}

func (c *ApplicationService) CreateApp(ctx context.Context, req *impb.CreateAppRequest) (*impb.Application, error) {
	// return c.UnimplementedApplicationsServer.CreateApp(ctx, req)

	input := req.GetApp()
	if err := model.ValidateInputApp(input); err != nil {
		return nil, err
	}
	app := model.NewApplication(input)

	app, err := c.store.Create(
		store.CreateAppRequest{
			Context: ctx,
			App:     app,
		},
	)

	if err != nil {
		return nil, err
	}

	return maskAppSecrets(app.Proto()), nil
}

// NOTE: when implemented, must call model.ValidateInputApp(req.GetApp()) before persisting.
func (c *ApplicationService) UpdateApp(ctx context.Context, req *impb.UpdateAppRequest) (*impb.Application, error) {
	return c.UnimplementedApplicationsServer.UpdateApp(ctx, req)
}
