package model

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	v1 "github.com/webitel/im-account-service/proto/gen/im/service/admin/v1"
)

func TestValidateAppCaptcha(t *testing.T) {
	tests := []struct {
		name    string
		captcha *v1.AppCaptcha
		wantErr bool
	}{
		{name: "nil is valid (not configured)", captcha: nil},
		{name: "disabled without secret is valid", captcha: &v1.AppCaptcha{Enabled: false}},
		{name: "enabled with secret is valid", captcha: &v1.AppCaptcha{Enabled: true, Secret: "s"}},
		{name: "enabled with empty secret is rejected", captcha: &v1.AppCaptcha{Enabled: true}, wantErr: true},
		{name: "enabled with blank secret is rejected", captcha: &v1.AppCaptcha{Enabled: true, Secret: "  "}, wantErr: true},
		{name: "min_score 0 is valid (default)", captcha: &v1.AppCaptcha{MinScore: 0}},
		{name: "min_score 1 is valid", captcha: &v1.AppCaptcha{MinScore: 1}},
		{name: "min_score below 0 is rejected", captcha: &v1.AppCaptcha{MinScore: -0.1}, wantErr: true},
		{name: "min_score above 1 is rejected", captcha: &v1.AppCaptcha{MinScore: 1.1}, wantErr: true},
		{name: "min_score NaN is rejected", captcha: &v1.AppCaptcha{MinScore: float32(math.NaN())}, wantErr: true},
		{name: "max_age unset is valid", captcha: &v1.AppCaptcha{MaxAge: nil}},
		{name: "max_age positive is valid", captcha: &v1.AppCaptcha{MaxAge: durationpb.New(time.Hour)}},
		{name: "max_age negative is rejected", captcha: &v1.AppCaptcha{MaxAge: durationpb.New(-time.Second)}, wantErr: true},
		{name: "max_age with mixed signs is rejected", captcha: &v1.AppCaptcha{MaxAge: &durationpb.Duration{Seconds: 1, Nanos: -1}}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAppCaptcha(tt.captcha)
			if tt.wantErr {
				if code := status.Code(err); code != codes.InvalidArgument {
					t.Errorf("code = %v, want %v (err = %v)", code, codes.InvalidArgument, err)
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateInputApp_Captcha(t *testing.T) {
	if code := status.Code(ValidateInputApp(&v1.InputApp{Captcha: &v1.AppCaptcha{Enabled: true}})); code != codes.InvalidArgument {
		t.Errorf("enabled captcha without secret: code = %v, want %v", code, codes.InvalidArgument)
	}

	if err := ValidateInputApp(&v1.InputApp{Captcha: &v1.AppCaptcha{Enabled: false}}); err != nil {
		t.Errorf("disabled captcha: unexpected error: %v", err)
	}
}

func TestCaptchaMinScore(t *testing.T) {
	tests := []struct {
		name    string
		captcha *v1.AppCaptcha
		want    float64
	}{
		{name: "nil uses default", captcha: nil, want: 0.5},
		{name: "zero uses default", captcha: &v1.AppCaptcha{MinScore: 0}, want: 0.5},
		{name: "explicit value is used", captcha: &v1.AppCaptcha{MinScore: 0.7}, want: 0.7},
		{name: "0.1 has no float32 error", captcha: &v1.AppCaptcha{MinScore: 0.1}, want: 0.1},
		{name: "0.3 has no float32 error", captcha: &v1.AppCaptcha{MinScore: 0.3}, want: 0.3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CaptchaMinScore(tt.captcha); got != tt.want {
				t.Errorf("CaptchaMinScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewApplication_Captcha(t *testing.T) {
	t.Run("copies captcha, defaults min_score and normalizes hostnames", func(t *testing.T) {
		input := &v1.InputApp{
			Name: "a",
			Captcha: &v1.AppCaptcha{
				Enabled:   true,
				Secret:    "  s3cret  ",
				Action:    " login ",
				Hostnames: []string{" Example.COM ", "example.com", "", "widget.example.com"},
				MaxAge:    durationpb.New(time.Minute),
			},
		}

		app := NewApplication(input)
		got := app.Captcha()

		if got == nil {
			t.Fatal("Captcha() = nil, want settings")
		}

		if got.GetSecret() != "s3cret" {
			t.Errorf("secret = %q, want trimmed %q", got.GetSecret(), "s3cret")
		}

		if got.GetAction() != "login" {
			t.Errorf("action = %q, want %q", got.GetAction(), "login")
		}

		if got.GetMinScore() != DefaultCaptchaMinScore {
			t.Errorf("min_score = %v, want default %v", got.GetMinScore(), DefaultCaptchaMinScore)
		}

		wantHosts := []string{"example.com", "widget.example.com"}
		if !reflect.DeepEqual(got.GetHostnames(), wantHosts) {
			t.Errorf("hostnames = %v, want %v", got.GetHostnames(), wantHosts)
		}

		if got.GetMaxAge().AsDuration() != time.Minute {
			t.Errorf("max_age = %v, want %v", got.GetMaxAge().AsDuration(), time.Minute)
		}

		if input.GetCaptcha().GetSecret() != "  s3cret  " || len(input.GetCaptcha().GetHostnames()) != 4 {
			t.Error("NewApplication must not mutate the input message")
		}
	})

	t.Run("absent captcha stays absent", func(t *testing.T) {
		if got := NewApplication(&v1.InputApp{Name: "a"}).Captcha(); got != nil {
			t.Errorf("Captcha() = %v, want nil", got)
		}
	})

	t.Run("accepted for widget apps", func(t *testing.T) {
		app := NewApplication(&v1.InputApp{
			Name:    "w",
			Type:    v1.AppType_APP_TYPE_WIDGET,
			Captcha: &v1.AppCaptcha{Enabled: true, Secret: "s"},
		})
		if app.Captcha() == nil || app.Proto().GetType() != v1.AppType_APP_TYPE_WIDGET {
			t.Error("widget application must keep captcha settings")
		}
	})
}

func TestApplication_CaptchaPresenceSurvivesJSON(t *testing.T) {
	// Mirrors the jsonb storage path (internal/store/postgres/apps.go).
	marshal := protojson.MarshalOptions{UseProtoNames: true}
	unmarshal := protojson.UnmarshalOptions{DiscardUnknown: true}

	stored, err := marshal.Marshal(NewApplication(&v1.InputApp{
		Name: "a",
		Captcha: &v1.AppCaptcha{
			Enabled: true,
			Secret:  "s",
			MaxAge:  durationpb.New(time.Hour),
		},
	}).Proto())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if !strings.Contains(string(stored), `"captcha"`) {
		t.Fatalf("stored json %s does not contain captcha", stored)
	}

	var loaded v1.Application
	if err := unmarshal.Unmarshal(stored, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if loaded.GetCaptcha() == nil {
		t.Fatalf("captcha presence lost after JSON round-trip: %s", stored)
	}

	if !loaded.GetCaptcha().GetEnabled() || loaded.GetCaptcha().GetSecret() != "s" {
		t.Errorf("captcha fields lost after JSON round-trip: %v", loaded.GetCaptcha())
	}

	if !proto.Equal(loaded.GetCaptcha().GetMaxAge(), durationpb.New(time.Hour)) {
		t.Errorf("max_age lost after JSON round-trip: %v", loaded.GetCaptcha().GetMaxAge())
	}
}
