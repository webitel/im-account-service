package store

import (
	"context"

	"github.com/webitel/im-account-service/internal/model"
)

type AppStore interface {
	Search(SearchAppRequest) (*model.ApplicationList, error)
	Create(CreateAppRequest) (*model.Application, error)
	Update(UpdateAppRequest) (*model.Application, error)
	Revoke(RevokeAppRequest) (*model.ApplicationList, error)
	// Delete()
}

type SearchAppRequest struct {
	context.Context
	Dc int64  // domain_id
	Id string // client_id

	Page int // offset
	Size int // limit, per page
}

type CreateAppRequest struct {
	context.Context
	App *model.Application
}

type UpdateAppRequest struct {
	context.Context
	App *model.Application
}

type RevokeAppRequest struct {
	context.Context
	Id     []string
	Reason error
	Delete bool
}
