package service

import (
	"context"

	"github.com/webitel/im-account-service/internal/model"
	"github.com/webitel/im-account-service/internal/store"
)

// GetApplication by given global [client_id] identifier
func (c *Manager) GetApplication(ctx context.Context, clientId string) (app *model.Application, err error) {

	// TODO: cache[ing]
	app, _ = c.cache.Get(_ClientId(clientId)).(*model.Application)
	if app != nil {
		return app, nil
	}
	defer func() {
		c.Cache(app, err)
	} ()

	apps := c.opts.Apps
	app, err = model.Get(apps.Search(
		store.SearchAppRequest{
			Context: ctx,
			Dc:      0,
			Id:      clientId,
			Page:    1,
			Size:    1,
		},
	))

	// Make sure the result satisfies the requested [client_id]
	if app != nil && app.ClientId() != clientId {
		app = nil // sanitize ; invalid [client_id] ; NOT Found !
	}

	if err != nil {
		return nil, err
	}

	return app, nil
}