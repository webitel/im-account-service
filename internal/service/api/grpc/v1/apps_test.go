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

type fakeAppStore struct {
	created int
	last    *model.Application
	apps    []*model.Application
}

var _ store.AppStore = (*fakeAppStore)(nil)

func (f *fakeAppStore) Search(store.SearchAppRequest) (*model.ApplicationList, error) {
	return &model.ApplicationList{Data: f.apps}, nil
}

func (f *fakeAppStore) Create(req store.CreateAppRequest) (*model.Application, error) {
	f.created++
	f.last = req.App
	return req.App, nil
}

func (f *fakeAppStore) Update(store.UpdateAppRequest) (*model.Application, error) {
	panic("not expected")
}

func (f *fakeAppStore) Revoke(store.RevokeAppRequest) (*model.ApplicationList, error) {
	panic("not expected")
}

func TestSearchApps_MasksSecrets(t *testing.T) {
	stored := model.ProtoApplication(&impb.Application{
		Id:      "app-1",
		Captcha: &impb.AppCaptcha{Enabled: true, Secret: "captcha-secret", MinScore: 0.5},
		Service: &impb.AppService{
			Secret: "service-secret",
			PushService: &impb.PUSHServiceClient{
				Web: &impb.PushWebServiceClient{Token: []byte("push-token")},
			},
		},
	})
	fake := &fakeAppStore{apps: []*model.Application{stored}}
	svc := NewApplicationService(fake, slog.Default())

	resp, err := svc.SearchApps(context.Background(), &impb.SearchAppRequest{})
	if err != nil {
		t.Fatalf("SearchApps() error = %v", err)
	}
	if len(resp.GetData()) != 1 {
		t.Fatalf("SearchApps() returned %d apps, want 1", len(resp.GetData()))
	}

	got := resp.GetData()[0]
	if got.GetCaptcha().GetSecret() != "" {
		t.Errorf("captcha.secret = %q, want cleared", got.GetCaptcha().GetSecret())
	}
	if got.GetService().GetSecret() != "" {
		t.Errorf("service.secret = %q, want cleared", got.GetService().GetSecret())
	}
	if !got.GetCaptcha().GetEnabled() {
		t.Error("captcha.enabled must be kept")
	}
	// Push credentials are still read by im-delivery-service from SearchApps.
	if string(got.GetService().GetPushService().GetWeb().GetToken()) != "push-token" {
		t.Error("push credentials must not be masked")
	}

	// The stored application must not be mutated by masking.
	if stored.Captcha().GetSecret() != "captcha-secret" {
		t.Errorf("stored captcha.secret = %q, want unchanged", stored.Captcha().GetSecret())
	}
	if stored.Proto().GetService().GetSecret() != "service-secret" {
		t.Errorf("stored service.secret = %q, want unchanged", stored.Proto().GetService().GetSecret())
	}
}

func TestCreateApp_CaptchaValidation(t *testing.T) {
	t.Run("enabled captcha without secret is rejected before store.Create", func(t *testing.T) {
		fake := &fakeAppStore{}
		svc := NewApplicationService(fake, slog.Default())

		_, err := svc.CreateApp(context.Background(), &impb.CreateAppRequest{
			App: &impb.InputApp{
				Name:    "a",
				Captcha: &impb.AppCaptcha{Enabled: true},
			},
		})

		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("code = %v, want %v (err = %v)", code, codes.InvalidArgument, err)
		}
		if fake.created != 0 {
			t.Errorf("store.Create called %d times, want 0", fake.created)
		}
	})

	t.Run("disabled captcha without secret is accepted", func(t *testing.T) {
		fake := &fakeAppStore{}
		svc := NewApplicationService(fake, slog.Default())

		_, err := svc.CreateApp(context.Background(), &impb.CreateAppRequest{
			App: &impb.InputApp{
				Name:    "a",
				Captcha: &impb.AppCaptcha{Enabled: false},
			},
		})
		if err != nil {
			t.Fatalf("CreateApp() error = %v", err)
		}
		if fake.created != 1 {
			t.Errorf("store.Create called %d times, want 1", fake.created)
		}
	})
}

func TestCreateApp_MasksSecrets(t *testing.T) {
	fake := &fakeAppStore{}
	svc := NewApplicationService(fake, slog.Default())

	resp, err := svc.CreateApp(context.Background(), &impb.CreateAppRequest{
		App: &impb.InputApp{
			Name:    "a",
			Captcha: &impb.AppCaptcha{Enabled: true, Secret: "s3cret"},
			Service: &impb.AppService{Secret: "svc"},
		},
	})
	if err != nil {
		t.Fatalf("CreateApp() error = %v", err)
	}

	if got := resp.GetCaptcha().GetSecret(); got != "" {
		t.Errorf("response captcha.secret = %q, want empty", got)
	}
	if got := resp.GetService().GetSecret(); got != "" {
		t.Errorf("response service.secret = %q, want empty", got)
	}
	if got := fake.last.Proto().GetCaptcha().GetSecret(); got != "s3cret" {
		t.Errorf("stored captcha.secret = %q, want s3cret", got)
	}
	if got := fake.last.Proto().GetService().GetSecret(); got != "svc" {
		t.Errorf("stored service.secret = %q, want svc", got)
	}
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
