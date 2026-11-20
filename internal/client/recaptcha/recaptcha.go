package recaptcha

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultURL     = "https://www.google.com/recaptcha/api/siteverify"
	DefaultTimeout = 5 * time.Second
)

const maxBodySize = 64 << 10

// ErrUnavailable wraps every failure to obtain a siteverify answer:
// transport error, timeout, non-200 status or undecodable body.
var ErrUnavailable = errors.New("recaptcha: siteverify unavailable")

type Response struct {
	Success     bool     `json:"success"`
	Score       float64  `json:"score"`
	Action      string   `json:"action"`
	ChallengeTs string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes"` //nolint:tagliatelle // Google siteverify wire format
}

type Verifier interface {
	Verify(ctx context.Context, secret, token, remoteIP string) (*Response, error)
}

type Options struct {
	URL        string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type Client struct {
	url    string
	http   *http.Client
	logger *slog.Logger
}

var _ Verifier = (*Client)(nil)

// Never falls back to http.DefaultTransport: WBTL_LOG_DEBUG=http replaces it with a
// body-dumping transport, and the request body carries the secret.
func NewClient(logger *slog.Logger, opts Options) *Client {
	if logger == nil {
		logger = slog.Default()
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          16,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: timeout,
			},
		}
	}

	return &Client{
		url:    cmp.Or(opts.URL, DefaultURL),
		http:   httpClient,
		logger: logger,
	}
}

// A 200 answer is returned as-is, including success=false; the caller decides.
func (c *Client) Verify(ctx context.Context, secret, token, remoteIP string) (*Response, error) {
	form := url.Values{
		"secret":   {secret},
		"response": {token},
	}
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxBodySize))

		return nil, fmt.Errorf("%w: http status %d", ErrUnavailable, resp.StatusCode)
	}

	var out Response
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBodySize)).Decode(&out); err != nil {
		return nil, fmt.Errorf("%w: decode: %v", ErrUnavailable, err)
	}

	c.logger.DebugContext(ctx, "siteverify",
		"success", out.Success,
		"score", out.Score,
		"action", out.Action,
		"hostname", out.Hostname,
		"error-codes", out.ErrorCodes,
	)

	return &out, nil
}
