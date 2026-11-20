package v1

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/webitel/im-account-service/internal/client/recaptcha"
	"github.com/webitel/im-account-service/internal/errors"
	"github.com/webitel/im-account-service/internal/model"
	adminpb "github.com/webitel/im-account-service/proto/gen/im/service/admin/v1"
)

const captchaMetadataKey = "captcha"

type captchaRecord struct {
	Score      float64   `json:"score"`
	Action     string    `json:"action,omitempty"`
	Hostname   string    `json:"hostname,omitempty"`
	VerifiedAt time.Time `json:"verified_at"`
	Iss        string    `json:"iss"`
	Sub        string    `json:"sub"`
}

func (r *captchaRecord) metadata() map[string]any {
	return map[string]any{
		"score":       r.Score,
		"action":      r.Action,
		"hostname":    r.Hostname,
		"verified_at": r.VerifiedAt.UTC().Format(time.RFC3339Nano),
		"iss":         r.Iss,
		"sub":         r.Sub,
	}
}

// The value is a map[string]any when loaded from the DB, but may be any Go value
// on a cached session, hence the JSON round-trip.
func sessionCaptchaRecord(session *model.Authorization) (*captchaRecord, bool) {
	if session == nil || session.Metadata == nil {
		return nil, false
	}
	value, ok := session.Metadata[captchaMetadataKey]
	if !ok {
		return nil, false
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}

	var rec captchaRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, false
	}
	return &rec, true
}

type captchaInput struct {
	Settings *adminpb.AppCaptcha
	Session  *model.Authorization // previous session for (dc, app_id, device_id), may be nil
	Iss, Sub string
	Token    string
	RemoteIP string
	Now      time.Time
}

// verifyCaptcha returns:
//   - (nil, nil, nil): nothing to store (captcha disabled, or a reusable result exists);
//   - (rec, res, nil): freshly verified, the caller must store rec on the session;
//   - (nil, res, err): login must be rejected with err.
func verifyCaptcha(ctx context.Context, verifier recaptcha.Verifier, in captchaInput) (*captchaRecord, *recaptcha.Response, error) {
	if !in.Settings.GetEnabled() {
		return nil, nil, nil
	}

	minScore := model.CaptchaMinScore(in.Settings)

	if prev, ok := sessionCaptchaRecord(in.Session); ok &&
		prev.Iss == in.Iss && prev.Sub == in.Sub &&
		prev.Score >= minScore &&
		!captchaExpired(prev, in.Settings.GetMaxAge(), in.Now) {
		return nil, nil, nil
	}

	token := strings.TrimSpace(in.Token)
	if token == "" {
		return nil, nil, errCaptchaRequired()
	}

	if verifier == nil {
		return nil, nil, errCaptchaUnavailable()
	}

	res, err := verifier.Verify(ctx, in.Settings.GetSecret(), token, in.RemoteIP)
	if err != nil || res == nil {
		// Fail closed: Google being down must not let logins through.
		return nil, nil, errCaptchaUnavailable()
	}

	// Misconfiguration of the application secret is a server-side error,
	// not a problem of the end-user token.
	if !res.Success && hasAnyCode(res.ErrorCodes, "missing-input-secret", "invalid-input-secret", "bad-request") {
		return nil, res, errCaptchaMisconfigured()
	}

	if !res.Success || res.Score < minScore {
		return nil, res, errCaptchaFailed()
	}

	if action := in.Settings.GetAction(); action != "" && res.Action != action {
		return nil, res, errCaptchaFailed()
	}

	if hosts := in.Settings.GetHostnames(); len(hosts) > 0 &&
		!slices.ContainsFunc(hosts, func(h string) bool {
			return strings.EqualFold(strings.TrimSpace(h), res.Hostname)
		}) {
		return nil, res, errCaptchaFailed()
	}

	return &captchaRecord{
		Score:      res.Score,
		Action:     res.Action,
		Hostname:   res.Hostname,
		VerifiedAt: in.Now.UTC(),
		Iss:        in.Iss,
		Sub:        in.Sub,
	}, res, nil
}

func captchaExpired(rec *captchaRecord, maxAge *durationpb.Duration, now time.Time) bool {
	if maxAge == nil {
		return false
	}
	limit := maxAge.AsDuration()
	if limit <= 0 {
		return false
	}

	if rec.VerifiedAt.IsZero() {
		return true
	}
	return now.Sub(rec.VerifiedAt) > limit
}

func hasAnyCode(codes []string, want ...string) bool {
	for _, code := range codes {
		if slices.Contains(want, code) {
			return true
		}
	}
	return false
}

func errCaptchaRequired() error {
	return errors.Unauthorized(
		errors.Status("CAPTCHA_REQUIRED"),
		errors.Message("captcha: required but missing"),
	)
}

func errCaptchaFailed() error {
	return errors.Unauthorized(
		errors.Status("CAPTCHA_FAILED"),
		errors.Message("captcha: verification failed"),
	)
}

func errCaptchaUnavailable() error {
	return errors.BadGateway(
		errors.Status("CAPTCHA_UNAVAILABLE"),
		errors.Message("captcha: verification service unavailable"),
	)
}

func errCaptchaMisconfigured() error {
	return errors.Internal(
		errors.Status("CAPTCHA_MISCONFIGURED"),
		errors.Message("captcha: verification misconfigured"),
	)
}
