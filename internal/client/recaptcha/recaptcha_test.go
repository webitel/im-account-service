package recaptcha

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(t *testing.T, url string, timeout time.Duration) *Client {
	t.Helper()

	return NewClient(nil, Options{URL: url, Timeout: timeout})
}

func TestVerifySuccessPostsFormBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s", r.Method)
		}

		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("content-type: got %q", ct)
		}

		if r.URL.RawQuery != "" {
			t.Errorf("query must be empty, got %q", r.URL.RawQuery)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}

		if got := r.PostForm.Get("secret"); got != "s3cret" {
			t.Errorf("secret: got %q", got)
		}

		if got := r.PostForm.Get("response"); got != "tok" {
			t.Errorf("response: got %q", got)
		}

		if got := r.PostForm.Get("remoteip"); got != "10.0.0.1" {
			t.Errorf("remoteip: got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")

		_, _ = w.Write([]byte(`{"success":true,"score":0.9,"action":"login","hostname":"example.com","challenge_ts":"2026-11-18T10:00:00Z"}`))
	}))
	defer srv.Close()

	got, err := newTestClient(t, srv.URL, 0).Verify(context.Background(), "s3cret", "tok", "10.0.0.1")
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	if !got.Success || got.Score != 0.9 || got.Action != "login" || got.Hostname != "example.com" || got.ChallengeTs != "2026-11-18T10:00:00Z" {
		t.Errorf("unexpected response: %+v", got)
	}
}

func TestVerifyOmitsEmptyRemoteIP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}

		if _, ok := r.PostForm["remoteip"]; ok {
			t.Errorf("remoteip must be absent when empty")
		}

		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	if _, err := newTestClient(t, srv.URL, 0).Verify(context.Background(), "s", "t", ""); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestVerifyFailureAnswerIsNotError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"error-codes":["invalid-input-response"]}`))
	}))
	defer srv.Close()

	got, err := newTestClient(t, srv.URL, 0).Verify(context.Background(), "s", "t", "")
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	if got.Success {
		t.Errorf("Success: want false")
	}

	if len(got.ErrorCodes) != 1 || got.ErrorCodes[0] != "invalid-input-response" {
		t.Errorf("ErrorCodes: got %v", got.ErrorCodes)
	}
}

func TestVerifyHTTPStatusIsUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv.URL, 0).Verify(context.Background(), "s", "t", "")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("want ErrUnavailable, got %v", err)
	}
}

func TestVerifyInvalidBodyIsUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv.URL, 0).Verify(context.Background(), "s", "t", "")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("want ErrUnavailable, got %v", err)
	}
}

func TestVerifyTimeoutIsUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(200 * time.Millisecond):
		case <-r.Context().Done():
		}

		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv.URL, 50*time.Millisecond).Verify(context.Background(), "s", "t", "")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("want ErrUnavailable, got %v", err)
	}
}

func TestVerifyClosedServerIsUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	_, err := newTestClient(t, url, 0).Verify(context.Background(), "s", "t", "")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("want ErrUnavailable, got %v", err)
	}
}

func TestVerifyBypassesDefaultTransport(t *testing.T) {
	orig := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Error("http.DefaultTransport must not be used")

		return nil, errors.New("default transport used")
	})

	t.Cleanup(func() { http.DefaultTransport = orig })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(Response{Success: true})
	}))
	defer srv.Close()

	if _, err := newTestClient(t, srv.URL, 0).Verify(context.Background(), "s", "t", ""); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
