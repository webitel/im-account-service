package v1

import (
	"context"
	"log/slog"
	"testing"

	"github.com/webitel/im-account-service/internal/model"
	"github.com/webitel/im-account-service/internal/store"
	impb "github.com/webitel/im-account-service/proto/gen/im/service/admin/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// fakeAppStore records Create calls; it never touches a database.
type fakeAppStore struct {
	created int
}

var _ store.AppStore = (*fakeAppStore)(nil)

func (f *fakeAppStore) Search(store.SearchAppRequest) (*model.ApplicationList, error) {
	return &model.ApplicationList{}, nil
}

func (f *fakeAppStore) Create(req store.CreateAppRequest) (*model.Application, error) {
	f.created++
	return req.App, nil
}

func (f *fakeAppStore) Update(store.UpdateAppRequest) (*model.Application, error) {
	panic("not expected")
}

func (f *fakeAppStore) Revoke(store.RevokeAppRequest) (*model.ApplicationList, error) {
	panic("not expected")
}

func TestCreateApp_AllowContactsValidation(t *testing.T) {
	t.Run("two default rules are rejected before store.Create", func(t *testing.T) {
		fake := &fakeAppStore{}
		svc := NewApplicationService(fake, slog.Default())

		_, err := svc.CreateApp(context.Background(), &impb.CreateAppRequest{
			App: &impb.InputApp{
				Name: "a",
				AllowContacts: &impb.ContactAllowList{Rules: []*impb.ContactAllowRule{
					{ContactId: "0c9a9b1e-7d5f-4c1a-9b8e-3f2a1d4c5e6a", Default: true},
					{ContactId: "6f1e2d3c-4b5a-4987-8a7b-6c5d4e3f2a1b", Default: true},
				}},
			},
		})

		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("code = %v, want %v (err = %v)", code, codes.InvalidArgument, err)
		}
		if fake.created != 0 {
			t.Errorf("store.Create called %d times, want 0", fake.created)
		}
	})

	t.Run("valid list is persisted with lower-cased contact ids", func(t *testing.T) {
		fake := &fakeAppStore{}
		svc := NewApplicationService(fake, slog.Default())

		resp, err := svc.CreateApp(context.Background(), &impb.CreateAppRequest{
			App: &impb.InputApp{
				Name: "a",
				AllowContacts: &impb.ContactAllowList{Rules: []*impb.ContactAllowRule{
					{ContactId: "0C9A9B1E-7D5F-4C1A-9B8E-3F2A1D4C5E6A", Default: true},
					{ContactId: "6f1e2d3c-4b5a-4987-8a7b-6c5d4e3f2a1b"},
				}},
			},
		})
		if err != nil {
			t.Fatalf("CreateApp() error = %v", err)
		}
		if fake.created != 1 {
			t.Errorf("store.Create called %d times, want 1", fake.created)
		}

		want := &impb.ContactAllowList{Rules: []*impb.ContactAllowRule{
			{ContactId: "0c9a9b1e-7d5f-4c1a-9b8e-3f2a1d4c5e6a", Default: true},
			{ContactId: "6f1e2d3c-4b5a-4987-8a7b-6c5d4e3f2a1b"},
		}}
		if got := resp.GetAllowContacts(); !proto.Equal(got, want) {
			t.Errorf("allow_contacts = %v, want %v", got, want)
		}
	})
}
