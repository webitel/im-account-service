package service

import (
	"context"

	"github.com/webitel/im-account-service/internal/model"
	"github.com/webitel/im-account-service/internal/store"
)

type SessionListOptions = store.ListSessionRequest
type SessionListOption func(req *SessionListOptions) error

func FindSessionDc(dc int64) SessionListOption {
	return func(req *SessionListOptions) error {
		req.Dc = max(dc, 0)
		return nil
	}
}

func FindSessionId(id string) SessionListOption {
	return func(req *SessionListOptions) error {
		req.Id = id
		return nil
	}
}

func (c *Manager) GetSession(ctx context.Context, lookup ...SessionListOption) (data *model.Authorization, err error) {
	
	// perform lookup by dc:iss/sub
	repo := c.opts.Sessions
	req := store.ListSessionRequest{
		Context: ctx,
		Page:    1,
		Size:    1,
	}

	for _, setup := range lookup {
		err := setup(&req)
		if err != nil {
			return nil, err
		}
	}


	for _, ref := range indexSessionRequest(&req) {
		data, _ = c.cache.Get(ref).(*model.Authorization)
		if data != nil {
			// Found cache[d] value ..
			return data, nil
		}
	}

	defer func() {
		c.Cache(data, err)
	} ()

	data, err = model.Get(repo.Search(req))

	if err != nil {
		return nil, err
	}

	return data, nil // not found ?
}

func (c *Manager) DelSession(ctx context.Context, session *model.Authorization) error {

	if session == nil {
		// [ OK ]; idempotent
		return nil
	}

	_ = c.DelCache(session)
	
	if session.Id == "" {
		// [ OK ]; idempotent
		return nil
	}
	
	store := c.opts.Sessions
	err := store.Delete(ctx, session.Id)
	if err != nil {
		// [ ERR ] something went wrong
		return err
	}
	// [ OK ]
	return nil
}

var TokenGen = model.GenerateOptions{
	Type:    "bearer",
	Length:  64,
	Expires: 0,   // never
	Refresh: nil, // no_refresh
	GenOpts: []model.GenerateOption{
		model.TokenNoRefresh(),
	},
}

func (c *Manager) RegisterDevice(req store.RegisterDeviceRequest) (*model.Authorization, error) {
	// for session Authorization
	session := req.Authorization
	// PERFORM: register for current session
	storage := c.Options().Sessions
	err := storage.RegisterDevice(req)

	if err != nil {
		return nil, err
	}
	// session.(Authorization).PUSH changed ; reload on demand ...
	_ = c.DelCache(session)
	
	// // publish UpdateNewDevice occured ..
	// updates, err := c.Updates()
	// if err != nil {
	// 	// Failed to publish UpdateNewDevice
	// 	return session, nil
	// }

	// changes := UpdateDevice{
	// 	Authorization: req.Authorization,
	// }

	// err = updates.PublishUpdateDevice(
	// 	req.Context, changes,
	// )
	
	// if err != nil {
	// 	c.Warn(req.Context, "Failed to publish UpdateDevice", "err", err)
	// }

	return session, nil
}

func (c *Manager) UnregisterDevice(req store.UnregisterDeviceRequest) (*model.Authorization, error) {
	// for session Authorization
	session := req.Authorization
	// PERFORM: register for current session
	storage := c.Options().Sessions
	err := storage.UnregisterDevice(req)

	if err != nil {
		return nil, err
	}
	// session.(Authorization).PUSH changed ; reload on demand ...
	_ = c.DelCache(session)

	// _ = c.PublishUpdate(&UpdateDeviceUnregister{
	// 	Authorization: authorizationFormProtoV1(session),
	// })
	// // [ OK ] ; sanitize ..
	// session.Device.Push = nil
	return session, nil
}