package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dlf-dds/goat-translator/internal/adapter"
	_ "github.com/dlf-dds/goat-translator/adapters/echo"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	// Don't reset the registry — echo registers itself at init.
	if _, err := adapter.Get("echo"); err != nil {
		t.Fatalf("echo adapter not registered (init failed?): %v", err)
	}
	return New(Config{
		Listen:       "127.0.0.1:0",
		MaxBodyBytes: 1 << 16,
		Logger:       logger,
	})
}

func TestHealthz(t *testing.T) {
	h := newTestHandler(t)
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestFormatsListsEcho(t *testing.T) {
	h := newTestHandler(t)
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/v1/formats")
	if err != nil {
		t.Fatalf("GET /v1/formats: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"name":"echo"`) {
		t.Fatalf("formats missing echo: %s", body)
	}
}

func TestTranslateEchoRoundTrip(t *testing.T) {
	h := newTestHandler(t)
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/v1/translate?from=echo&to=echo", "application/octet-stream", bytes.NewBufferString("hello"))
	if err != nil {
		t.Fatalf("POST /v1/translate: %v", err)
	}
	defer resp.Body.Close()
	// echo→echo should be rejected as same-format.
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected non-200 for echo→echo (same format), got 200")
	}
}

func TestTranslateEchoToCanonical(t *testing.T) {
	h := newTestHandler(t)
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/v1/translate?from=echo&to=canonical", "application/octet-stream", bytes.NewBufferString("hello"))
	if err != nil {
		t.Fatalf("POST /v1/translate: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"source_format": "echo"`) {
		t.Fatalf("canonical body missing source_format: %s", body)
	}
}

func TestTranslateMissingParams(t *testing.T) {
	h := newTestHandler(t)
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/v1/translate", "application/octet-stream", bytes.NewBufferString("hello"))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestDetect(t *testing.T) {
	h := newTestHandler(t)
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/v1/detect", "application/octet-stream", bytes.NewBufferString("echo:foo"))
	if err != nil {
		t.Fatalf("POST /v1/detect: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body struct {
		Matches []string `json:"matches"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body.Matches) == 0 || body.Matches[0] != "echo" {
		t.Fatalf("matches = %v, want [echo]", body.Matches)
	}
}
