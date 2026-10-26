package model

import (
	"testing"

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
