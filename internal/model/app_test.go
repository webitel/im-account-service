package model

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	v1 "github.com/webitel/im-account-service/proto/gen/im/service/admin/v1"
)

func TestAppTypeOrDefault(t *testing.T) {
	tests := []struct {
		in   v1.AppType
		want v1.AppType
	}{
		{v1.AppType_APP_TYPE_UNSPECIFIED, v1.AppType_APP_TYPE_PORTAL},
		{v1.AppType_APP_TYPE_PORTAL, v1.AppType_APP_TYPE_PORTAL},
		{v1.AppType_APP_TYPE_WIDGET, v1.AppType_APP_TYPE_WIDGET},
	}

	for _, tt := range tests {
		if got := AppTypeOrDefault(tt.in); got != tt.want {
			t.Errorf("AppTypeOrDefault(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestNewApplication_Type(t *testing.T) {
	if got := NewApplication(&v1.InputApp{Name: "a"}).Proto().GetType(); got != v1.AppType_APP_TYPE_PORTAL {
		t.Errorf("unspecified input type = %v, want PORTAL", got)
	}

	if got := NewApplication(&v1.InputApp{Name: "a", Type: v1.AppType_APP_TYPE_WIDGET}).Proto().GetType(); got != v1.AppType_APP_TYPE_WIDGET {
		t.Errorf("widget input type = %v, want WIDGET", got)
	}
}

func TestProtoApplication_LegacyTypeIsPortal(t *testing.T) {
	src := &v1.Application{Id: "x"}
	if got := ProtoApplication(src).Proto().GetType(); got != v1.AppType_APP_TYPE_PORTAL {
		t.Errorf("legacy app type = %v, want PORTAL", got)
	}

	if src.GetType() != v1.AppType_APP_TYPE_UNSPECIFIED {
		t.Error("ProtoApplication must not mutate its source")
	}
}

func TestNewApplication_AllowContacts(t *testing.T) {
	t.Run("unset stays nil", func(t *testing.T) {
		if got := NewApplication(&v1.InputApp{Name: "a"}).Proto().GetAllowContacts(); got != nil {
			t.Errorf("allow_contacts = %v, want nil", got)
		}
	})

	t.Run("set but empty stays non-nil", func(t *testing.T) {
		got := NewApplication(&v1.InputApp{Name: "a", AllowContacts: &v1.ContactAllowList{}}).Proto().GetAllowContacts()
		if got == nil {
			t.Fatal("allow_contacts = nil, want non-nil empty list")
		}

		if len(got.GetRules()) != 0 {
			t.Errorf("rules len = %d, want 0", len(got.GetRules()))
		}
	})

	t.Run("rules are copied and contact ids lower-cased without mutating input", func(t *testing.T) {
		input := &v1.InputApp{Name: "a", AllowContacts: &v1.ContactAllowList{Rules: []*v1.ContactAllowRule{
			{ContactId: "0C9A9B1E-7D5F-4C1A-9B8E-3F2A1D4C5E6A", Default: true},
			{ContactId: contactB},
		}}}

		want := &v1.ContactAllowList{Rules: []*v1.ContactAllowRule{
			{ContactId: "0c9a9b1e-7d5f-4c1a-9b8e-3f2a1d4c5e6a", Default: true},
			{ContactId: contactB},
		}}

		got := NewApplication(input).Proto().GetAllowContacts()
		if !proto.Equal(got, want) {
			t.Errorf("allow_contacts = %v, want %v", got, want)
		}

		if input.GetAllowContacts().GetRules()[0].GetContactId() != "0C9A9B1E-7D5F-4C1A-9B8E-3F2A1D4C5E6A" {
			t.Error("NewApplication must not mutate the input message")
		}
	})
}

func TestApplication_AllowContactsPresenceSurvivesJSON(t *testing.T) {
	// Mirrors the jsonb storage path (internal/store/postgres/apps.go).
	marshal := protojson.MarshalOptions{UseProtoNames: true}
	unmarshal := protojson.UnmarshalOptions{DiscardUnknown: true}

	stored, err := marshal.Marshal(NewApplication(&v1.InputApp{
		Name:          "a",
		AllowContacts: &v1.ContactAllowList{},
	}).Proto())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if !strings.Contains(string(stored), `"allow_contacts"`) {
		t.Fatalf("stored json %s does not contain allow_contacts", stored)
	}

	var loaded v1.Application
	if err := unmarshal.Unmarshal(stored, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if loaded.GetAllowContacts() == nil {
		t.Errorf("allow_contacts presence lost after JSON round-trip: %s", stored)
	}
}
