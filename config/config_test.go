package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/pflag"

	"github.com/webitel/webitel-go-kit/appconfig/secret"

	"github.com/webitel/im-account-service/config"
)

// startAsSystemdUnit makes Load* see what a systemd unit with
// LoadCredential*= sees: no secret env vars, only $CREDENTIALS_DIRECTORY
// holding the given credentials. It also resets the global flag set, since
// the loaders register into pflag.CommandLine and parse os.Args.
func startAsSystemdUnit(t *testing.T, creds map[string]string) string {
	t.Helper()

	for _, env := range []string{
		"CONFIG_FILE",
		"POSTGRES_DSN", "POSTGRES_DSN_FILE",
		"REDIS_PASSWORD", "REDIS_PASSWORD_FILE",
		"PUBSUB_URL", "PUBSUB_URL_FILE",
		"SERVICE_CONN_KEY", "SERVICE_CONN_CLIENT_KEY",
		"RECAPTCHA_URL", "RECAPTCHA_TIMEOUT",
	} {
		t.Setenv(env, "")
		os.Unsetenv(env)
	}

	dir := t.TempDir()
	for name, value := range creds {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value+"\n"), 0o400); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv("CREDENTIALS_DIRECTORY", dir)

	args, flags := os.Args, pflag.CommandLine
	os.Args = []string{"server"}
	pflag.CommandLine = pflag.NewFlagSet("server", pflag.ContinueOnError)

	t.Cleanup(func() { os.Args, pflag.CommandLine = args, flags })

	return dir
}

func TestLoadServerConfigSecretsFromCredentials(t *testing.T) {
	dir := startAsSystemdUnit(t, map[string]string{
		"postgres.dsn":     "postgres://db.example.com/im_test",
		"redis.password":   "test-redis-password",
		"pubsub.url":       "amqp://mq.example.com/",
		"service.conn.key": "test-only-key",
	})
	t.Setenv("SERVICE_CONN_CA", "/etc/test/ca.crt")
	t.Setenv("SERVICE_CONN_CERT", "/etc/test/service.crt")

	cfg, err := config.LoadServerConfig()
	if err != nil {
		t.Fatalf("LoadServerConfig: %v", err)
	}

	for name, got := range map[string]string{
		"postgres.dsn":     cfg.Postgres.DSN,
		"redis.password":   cfg.Redis.Password,
		"pubsub.url":       cfg.Pubsub.URL,
		"service.conn.key": cfg.Service.Connection.Key,
	} {
		want := map[string]string{
			"postgres.dsn":     "postgres://db.example.com/im_test",
			"redis.password":   "test-redis-password",
			"pubsub.url":       "amqp://mq.example.com/",
			"service.conn.key": filepath.Join(dir, "service.conn.key"),
		}[name]
		if got != want {
			t.Errorf("%s: got %q, want %q", name, got, want)
		}
	}
}

func TestLoadServerConfigPlainSecretShadowingCredentialFails(t *testing.T) {
	startAsSystemdUnit(t, map[string]string{"postgres.dsn": "postgres://db.example.com/new"})
	t.Setenv("POSTGRES_DSN", "postgres://db.example.com/old")

	_, err := config.LoadServerConfig()
	if !errors.Is(err, secret.ErrConflict) {
		t.Fatalf("want secret.ErrConflict, got %v", err)
	}
}

func TestLoadServerConfigRecaptchaDefaults(t *testing.T) {
	startAsSystemdUnit(t, map[string]string{
		"postgres.dsn":     "postgres://db.example.com/im_test",
		"pubsub.url":       "amqp://mq.example.com/",
		"service.conn.key": "test-only-key",
	})
	t.Setenv("SERVICE_CONN_CA", "/etc/test/ca.crt")
	t.Setenv("SERVICE_CONN_CERT", "/etc/test/service.crt")

	cfg, err := config.LoadServerConfig()
	if err != nil {
		t.Fatalf("LoadServerConfig: %v", err)
	}
	if want := "https://www.google.com/recaptcha/api/siteverify"; cfg.Recaptcha.URL != want {
		t.Errorf("recaptcha.url: got %q, want %q", cfg.Recaptcha.URL, want)
	}
	if cfg.Recaptcha.Timeout != 5*time.Second {
		t.Errorf("recaptcha.timeout: got %v, want 5s", cfg.Recaptcha.Timeout)
	}
}

func TestLoadServerConfigRecaptchaFromEnv(t *testing.T) {
	startAsSystemdUnit(t, map[string]string{
		"postgres.dsn":     "postgres://db.example.com/im_test",
		"pubsub.url":       "amqp://mq.example.com/",
		"service.conn.key": "test-only-key",
	})
	t.Setenv("SERVICE_CONN_CA", "/etc/test/ca.crt")
	t.Setenv("SERVICE_CONN_CERT", "/etc/test/service.crt")
	t.Setenv("RECAPTCHA_URL", "http://127.0.0.1:9/verify")
	t.Setenv("RECAPTCHA_TIMEOUT", "2s")

	cfg, err := config.LoadServerConfig()
	if err != nil {
		t.Fatalf("LoadServerConfig: %v", err)
	}
	if cfg.Recaptcha.URL != "http://127.0.0.1:9/verify" {
		t.Errorf("recaptcha.url: got %q", cfg.Recaptcha.URL)
	}
	if cfg.Recaptcha.Timeout != 2*time.Second {
		t.Errorf("recaptcha.timeout: got %v, want 2s", cfg.Recaptcha.Timeout)
	}
}
