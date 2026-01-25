package handler

import (
	"context"

	"github.com/webitel/im-account-service/internal/model"
	"github.com/webitel/im-account-service/internal/store"
)

// GetApplication by given global [client_id] identifier
func (srv *Service) GetApplication(ctx context.Context, clientId string) (*model.Application, error) {

	// TODO: cache[ing]

	apps := srv.opts.Apps
	app, err := model.Get(apps.Search(
		store.SearchAppRequest{
			Context: ctx,
			Dc:      0,
			Id:      clientId,
			Page:    1,
			Size:    1,
		},
	))

	if err != nil {
		return nil, err
	}

	// Make sure the result satisfies the requested [client_id]
	if app != nil && app.ClientId() != clientId {
		app = nil // sanitize ; invalid [client_id] ; NOT Found !
	}

	return app, nil
}
