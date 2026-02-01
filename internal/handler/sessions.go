package handler

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

func (srv *Service) GetSession(ctx context.Context, lookup ...SessionListOption) (*model.Authorization, error) {

	// perform lookup by dc:iss/sub
	repo := srv.opts.Sessions
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

	got, err := model.Get(repo.Search(req))

	if err != nil {
		return nil, err
	}

	return got, nil // not found ?
}
