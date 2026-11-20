package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/webitel/im-account-service/internal/client/recaptcha"
	"github.com/webitel/im-account-service/internal/errors"
	"github.com/webitel/im-account-service/internal/model"
	adminpb "github.com/webitel/im-account-service/proto/gen/im/service/admin/v1"
)

type fakeVerifier struct {
	res               *recaptcha.Response
	err               error
	calls             int
	secret, token, ip string
}

var _ recaptcha.Verifier = (*fakeVerifier)(nil)

func (f *fakeVerifier) Verify(_ context.Context, secret, token, remoteIP string) (*recaptcha.Response, error) {
	f.calls++
	f.secret, f.token, f.ip = secret, token, remoteIP
	return f.res, f.err
}

var captchaNow = time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)

func enabledCaptcha() *adminpb.AppCaptcha {
	return &adminpb.AppCaptcha{Enabled: true, Secret: "secret-value"}
}

// storedSession simulates a session loaded from the DB: Metadata holds plain map[string]any values.
func storedSession(t *testing.T, rec *captchaRecord) *model.Authorization {
	t.Helper()
	raw, err := json.Marshal(map[string]any{captchaMetadataKey: rec.metadata()})
	if err != nil {
		t.Fatal(err)
	}

	var meta map[string]any
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatal(err)
	}
	return &model.Authorization{Metadata: meta}
}

func assertCaptchaErr(t *testing.T, err error, code codes.Code, status string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want error %s, got nil", status)
	}
	if got := grpcCode(err); got != code {
		t.Fatalf("grpc code: want %v, got %v (err: %v)", code, got, err)
	}
	e, ok := errors.FromError(err)
	if !ok || e.Status != status {
		t.Fatalf("status: want %s, got %+v", status, e)
	}
}

func grpcCode(err error) codes.Code {
	return status.Code(err)
}

func TestVerifyCaptcha_Disabled(t *testing.T) {
	v := &fakeVerifier{}
	for _, settings := range []*adminpb.AppCaptcha{nil, {Enabled: false}} {
		rec, res, err := verifyCaptcha(context.Background(), v, captchaInput{Settings: settings, Now: captchaNow})
		if rec != nil || res != nil || err != nil {
			t.Fatalf("disabled: want (nil, nil, nil), got (%v, %v, %v)", rec, res, err)
		}
	}
	if v.calls != 0 {
		t.Fatalf("disabled captcha must not call Google, calls=%d", v.calls)
	}
}

func TestVerifyCaptcha_Required(t *testing.T) {
	v := &fakeVerifier{}
	for _, token := range []string{"", "   "} {
		_, _, err := verifyCaptcha(context.Background(), v, captchaInput{
			Settings: enabledCaptcha(), Token: token, Now: captchaNow,
		})
		assertCaptchaErr(t, err, codes.Unauthenticated, "CAPTCHA_REQUIRED")
	}
	if v.calls != 0 {
		t.Fatalf("missing token must not call Google, calls=%d", v.calls)
	}
}

func TestVerifyCaptcha_Unavailable(t *testing.T) {
	in := captchaInput{Settings: enabledCaptcha(), Token: "tok", Now: captchaNow}

	v := &fakeVerifier{err: fmt.Errorf("%w: timeout", recaptcha.ErrUnavailable)}
	_, _, err := verifyCaptcha(context.Background(), v, in)
	assertCaptchaErr(t, err, codes.Unavailable, "CAPTCHA_UNAVAILABLE")

	_, _, err = verifyCaptcha(context.Background(), nil, in)
	assertCaptchaErr(t, err, codes.Unavailable, "CAPTCHA_UNAVAILABLE")
}

func TestVerifyCaptcha_Misconfigured(t *testing.T) {
	for _, code := range []string{"missing-input-secret", "invalid-input-secret", "bad-request"} {
		v := &fakeVerifier{res: &recaptcha.Response{Success: false, ErrorCodes: []string{code}}}
		_, _, err := verifyCaptcha(context.Background(), v, captchaInput{
			Settings: enabledCaptcha(), Token: "tok", Now: captchaNow,
		})
		assertCaptchaErr(t, err, codes.Internal, "CAPTCHA_MISCONFIGURED")
	}
}

func TestVerifyCaptcha_TokenErrorsAreFailed(t *testing.T) {
	for _, code := range []string{"invalid-input-response", "timeout-or-duplicate"} {
		v := &fakeVerifier{res: &recaptcha.Response{Success: false, ErrorCodes: []string{code}}}
		_, _, err := verifyCaptcha(context.Background(), v, captchaInput{
			Settings: enabledCaptcha(), Token: "tok", Now: captchaNow,
		})
		assertCaptchaErr(t, err, codes.Unauthenticated, "CAPTCHA_FAILED")
	}
}

func TestVerifyCaptcha_Decision(t *testing.T) {
	ok := recaptcha.Response{Success: true, Score: 0.9, Action: "login", Hostname: "Example.com"}

	tests := []struct {
		name     string
		settings *adminpb.AppCaptcha
		res      recaptcha.Response
		wantErr  string // "" = success
	}{
		{
			name:     "success",
			settings: enabledCaptcha(),
			res:      ok,
		},
		{
			name:     "not success",
			settings: enabledCaptcha(),
			res:      recaptcha.Response{Success: false, Score: 0.9},
			wantErr:  "CAPTCHA_FAILED",
		},
		{
			name:     "score below default min 0.5",
			settings: enabledCaptcha(),
			res:      recaptcha.Response{Success: true, Score: 0.4},
			wantErr:  "CAPTCHA_FAILED",
		},
		{
			name:     "score 0.5 meets default min",
			settings: enabledCaptcha(),
			res:      recaptcha.Response{Success: true, Score: 0.5},
		},
		{
			name:     "score exactly at min 0.3 passes",
			settings: &adminpb.AppCaptcha{Enabled: true, Secret: "s", MinScore: 0.3},
			res:      recaptcha.Response{Success: true, Score: 0.3},
		},
		{
			name:     "score exactly at min 0.1 passes",
			settings: &adminpb.AppCaptcha{Enabled: true, Secret: "s", MinScore: 0.1},
			res:      recaptcha.Response{Success: true, Score: 0.1},
		},
		{
			name:     "action mismatch",
			settings: &adminpb.AppCaptcha{Enabled: true, Secret: "s", Action: "signup"},
			res:      ok,
			wantErr:  "CAPTCHA_FAILED",
		},
		{
			name:     "action not checked when empty in settings",
			settings: &adminpb.AppCaptcha{Enabled: true, Secret: "s"},
			res:      recaptcha.Response{Success: true, Score: 0.9, Action: "anything"},
		},
		{
			name:     "hostname not in list",
			settings: &adminpb.AppCaptcha{Enabled: true, Secret: "s", Hostnames: []string{"other.org"}},
			res:      ok,
			wantErr:  "CAPTCHA_FAILED",
		},
		{
			name:     "hostname matches case-insensitively",
			settings: &adminpb.AppCaptcha{Enabled: true, Secret: "s", Hostnames: []string{" example.COM "}},
			res:      ok,
		},
		{
			name:     "hostnames not checked when empty",
			settings: &adminpb.AppCaptcha{Enabled: true, Secret: "s"},
			res:      ok,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.res
			v := &fakeVerifier{res: &res}

			rec, _, err := verifyCaptcha(context.Background(), v, captchaInput{
				Settings: tt.settings,
				Iss:      "iss-1",
				Sub:      "sub-1",
				Token:    "  tok  ",
				RemoteIP: "10.0.0.1",
				Now:      captchaNow,
			})
			if tt.wantErr != "" {
				assertCaptchaErr(t, err, codes.Unauthenticated, tt.wantErr)
				if rec != nil {
					t.Fatalf("rejected login must not return a record: %+v", rec)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rec == nil {
				t.Fatal("want a record on success")
			}
			if v.secret != tt.settings.GetSecret() || v.token != "tok" || v.ip != "10.0.0.1" {
				t.Fatalf("verifier args: secret=%q token=%q ip=%q", v.secret, v.token, v.ip)
			}
		})
	}
}

func TestVerifyCaptcha_SuccessRecord(t *testing.T) {
	v := &fakeVerifier{res: &recaptcha.Response{Success: true, Score: 0.8, Action: "login", Hostname: "example.com"}}
	settings := &adminpb.AppCaptcha{Enabled: true, Secret: "s", Action: "login", Hostnames: []string{"example.com"}}

	rec, res, err := verifyCaptcha(context.Background(), v, captchaInput{
		Settings: settings, Iss: "iss-1", Sub: "sub-1", Token: "tok", Now: captchaNow,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || rec == nil {
		t.Fatalf("want record and response, got rec=%v res=%v", rec, res)
	}
	if rec.Score != 0.8 || rec.Action != "login" || rec.Hostname != "example.com" ||
		rec.Iss != "iss-1" || rec.Sub != "sub-1" || !rec.VerifiedAt.Equal(captchaNow) {
		t.Fatalf("unexpected record: %+v", rec)
	}
}

func TestVerifyCaptcha_Reuse(t *testing.T) {
	prev := &captchaRecord{Score: 0.9, Iss: "iss-1", Sub: "sub-1", VerifiedAt: captchaNow.Add(-10 * time.Minute)}

	tests := []struct {
		name     string
		settings *adminpb.AppCaptcha
		rec      *captchaRecord
		iss, sub string
		session  bool
		wantErr  string
	}{
		{name: "same identity, no max_age", settings: enabledCaptcha(), rec: prev, iss: "iss-1", sub: "sub-1", session: true},
		{name: "different sub requires token", settings: enabledCaptcha(), rec: prev, iss: "iss-1", sub: "sub-2", session: true, wantErr: "CAPTCHA_REQUIRED"},
		{name: "different iss requires token", settings: enabledCaptcha(), rec: prev, iss: "iss-2", sub: "sub-1", session: true, wantErr: "CAPTCHA_REQUIRED"},
		{
			name:     "raised min_score requires token",
			settings: &adminpb.AppCaptcha{Enabled: true, Secret: "s", MinScore: 0.95},
			rec:      prev, iss: "iss-1", sub: "sub-1", session: true, wantErr: "CAPTCHA_REQUIRED",
		},
		{
			name:     "max_age passed requires token",
			settings: &adminpb.AppCaptcha{Enabled: true, Secret: "s", MaxAge: durationpb.New(time.Hour)},
			rec:      &captchaRecord{Score: 0.9, Iss: "iss-1", Sub: "sub-1", VerifiedAt: captchaNow.Add(-2 * time.Hour)},
			iss:      "iss-1", sub: "sub-1", session: true, wantErr: "CAPTCHA_REQUIRED",
		},
		{
			name:     "within max_age reused",
			settings: &adminpb.AppCaptcha{Enabled: true, Secret: "s", MaxAge: durationpb.New(time.Hour)},
			rec:      &captchaRecord{Score: 0.9, Iss: "iss-1", Sub: "sub-1", VerifiedAt: captchaNow.Add(-30 * time.Minute)},
			iss:      "iss-1", sub: "sub-1", session: true,
		},
		{
			name:     "zero max_age never expires",
			settings: &adminpb.AppCaptcha{Enabled: true, Secret: "s", MaxAge: durationpb.New(0)},
			rec:      &captchaRecord{Score: 0.9, Iss: "iss-1", Sub: "sub-1", VerifiedAt: captchaNow.Add(-1000 * time.Hour)},
			iss:      "iss-1", sub: "sub-1", session: true,
		},
		{
			name:     "stored score exactly at min 0.3 reused",
			settings: &adminpb.AppCaptcha{Enabled: true, Secret: "s", MinScore: 0.3},
			rec:      &captchaRecord{Score: 0.3, Iss: "iss-1", Sub: "sub-1", VerifiedAt: captchaNow.Add(-time.Minute)},
			iss:      "iss-1", sub: "sub-1", session: true,
		},
		{name: "no previous session requires token", settings: enabledCaptcha(), iss: "iss-1", sub: "sub-1", session: false, wantErr: "CAPTCHA_REQUIRED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &fakeVerifier{}

			var sess *model.Authorization
			if tt.session {
				sess = storedSession(t, tt.rec)
			}
			rec, res, err := verifyCaptcha(context.Background(), v, captchaInput{
				Settings: tt.settings, Session: sess, Iss: tt.iss, Sub: tt.sub, Now: captchaNow,
			})
			if tt.wantErr != "" {
				assertCaptchaErr(t, err, codes.Unauthenticated, tt.wantErr)
				return
			}
			if err != nil || rec != nil || res != nil {
				t.Fatalf("reuse: want (nil, nil, nil), got (%v, %v, %v)", rec, res, err)
			}
			if v.calls != 0 {
				t.Fatalf("reuse must not call Google, calls=%d", v.calls)
			}
		})
	}
}

func TestVerifyCaptcha_GarbageMetadata(t *testing.T) {
	sess := &model.Authorization{Metadata: map[string]any{captchaMetadataKey: "x"}}
	v := &fakeVerifier{}
	_, _, err := verifyCaptcha(context.Background(), v, captchaInput{
		Settings: enabledCaptcha(), Session: sess, Iss: "iss-1", Sub: "sub-1", Now: captchaNow,
	})
	assertCaptchaErr(t, err, codes.Unauthenticated, "CAPTCHA_REQUIRED")
}

func TestCaptchaRecord_MetadataRoundTrip(t *testing.T) {
	rec := &captchaRecord{
		Score:      0.7,
		Action:     "login",
		Hostname:   "example.com",
		VerifiedAt: captchaNow,
		Iss:        "iss-1",
		Sub:        "sub-1",
	}

	got, ok := sessionCaptchaRecord(storedSession(t, rec))
	if !ok {
		t.Fatal("record not found after round-trip")
	}
	if got.Score != rec.Score || got.Action != rec.Action || got.Hostname != rec.Hostname ||
		got.Iss != rec.Iss || got.Sub != rec.Sub || !got.VerifiedAt.Equal(rec.VerifiedAt) {
		t.Fatalf("round-trip mismatch: want %+v, got %+v", rec, got)
	}
}
