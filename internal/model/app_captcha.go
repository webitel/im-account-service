package model

import (
	"math"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/webitel/im-account-service/internal/errors"
	v1 "github.com/webitel/im-account-service/proto/gen/im/service/admin/v1"
)

// proto3 cannot tell an unset min_score from 0, so 0 always means this default.
const DefaultCaptchaMinScore float32 = 0.5

func ValidateAppCaptcha(c *v1.AppCaptcha) error {
	if c == nil {
		return nil
	}

	score := c.GetMinScore()
	if math.IsNaN(float64(score)) || score < 0 || score > 1 {
		return errors.BadRequest(
			errors.Status("BAD_APP_CAPTCHA"),
			errors.Message("app: captcha.min_score( %v ); must be within [0.0, 1.0]", score),
		)
	}

	if maxAge := c.GetMaxAge(); maxAge != nil {
		if err := maxAge.CheckValid(); err != nil || maxAge.AsDuration() < 0 {
			return errors.BadRequest(
				errors.Status("BAD_APP_CAPTCHA"),
				errors.Message("app: captcha.max_age; invalid duration, must be non-negative"),
			)
		}
	}

	if c.GetEnabled() && strings.TrimSpace(c.GetSecret()) == "" {
		return errors.BadRequest(
			errors.Status("BAD_APP_CAPTCHA"),
			errors.Message("app: captcha.secret; required when captcha is enabled"),
		)
	}

	return nil
}

func normalizeAppCaptcha(c *v1.AppCaptcha) *v1.AppCaptcha {
	if c == nil {
		return nil
	}

	out := proto.CloneOf(c)
	out.Secret = strings.TrimSpace(out.GetSecret())
	out.Action = strings.TrimSpace(out.GetAction())

	if out.GetMinScore() == 0 {
		out.MinScore = DefaultCaptchaMinScore
	}

	var (
		seen      = make(map[string]struct{}, len(out.GetHostnames()))
		hostnames = make([]string, 0, len(out.GetHostnames()))
	)

	for _, host := range out.GetHostnames() {
		host = strings.ToLower(strings.TrimSpace(host))
		if host == "" {
			continue
		}

		if _, dup := seen[host]; dup {
			continue
		}

		seen[host] = struct{}{}
		hostnames = append(hostnames, host)
	}

	out.Hostnames = hostnames

	return out
}

func CaptchaMinScore(c *v1.AppCaptcha) float64 {
	s := c.GetMinScore()
	if s <= 0 {
		s = DefaultCaptchaMinScore
	}

	return float32ToFloat64(s)
}

// float64(float32(0.3)) is 0.30000001192..., which would reject a Google score of
// exactly 0.3. Going through the shortest decimal form of the float32 yields 0.3.
func float32ToFloat64(f float32) float64 {
	v, _ := strconv.ParseFloat(strconv.FormatFloat(float64(f), 'g', -1, 32), 64)

	return v
}

// Returns the cached application's own message: callers must not mutate it.
func (app *Application) Captcha() *v1.AppCaptcha {
	return app.opts.GetCaptcha()
}
