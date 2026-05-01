// Package server implements the HTTP REST surface of the translator.
//
// Endpoints:
//
//	POST /v1/translate?from=<source>&to=<target>   request body is input bytes
//	POST /v1/detect                                request body is input bytes
//	GET  /v1/formats                               JSON list of registered formats
//	GET  /healthz                                  liveness check
//	GET  /metrics                                  Prometheus metrics
//
// The server is stateless. It depends on the global adapter registry; any
// adapters registered before Start() are served. There is no
// dynamic-loading of adapters at runtime — that would defeat the
// build-tags posture from docs/BUILD-TAGS.md.
package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/dlf-dds/goat-translator/internal/adapter"
	"github.com/dlf-dds/goat-translator/internal/pipeline"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config controls server behavior.
type Config struct {
	// Listen is the address the HTTP server binds to (e.g. "127.0.0.1:8080").
	Listen string

	// MaxBodyBytes caps the size of any single request body. Default is 8 MiB
	// when zero.
	MaxBodyBytes int64

	// Logger is the slog instance for request logs. Required.
	Logger *slog.Logger
}

const defaultMaxBody = 8 << 20 // 8 MiB

// New returns an http.Handler ready to be served. The handler is the full
// REST mux; callers attach it to a *http.Server with their own timeouts.
func New(cfg Config) http.Handler {
	if cfg.MaxBodyBytes == 0 {
		cfg.MaxBodyBytes = defaultMaxBody
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/translate", translateHandler(cfg))
	mux.HandleFunc("/v1/detect", detectHandler(cfg))
	mux.HandleFunc("/v1/formats", formatsHandler(cfg))
	mux.HandleFunc("/healthz", healthHandler())
	mux.Handle("/metrics", promhttp.Handler())
	return mux
}

func translateHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		if from == "" || to == "" {
			httpError(w, http.StatusBadRequest, "from= and to= query parameters required")
			return
		}
		input, err := readBody(r, cfg.MaxBodyBytes)
		if err != nil {
			httpError(w, http.StatusBadRequest, err.Error())
			return
		}
		res, err := pipeline.Translate(input, from, to)
		if err != nil {
			cfg.Logger.Warn("translate failed", "from", from, "to", to, "err", err)
			httpError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		cfg.Logger.Info("translate ok",
			"from", from,
			"to", to,
			"input_size", res.Audit.InputSize,
			"output_size", res.Audit.OutputSize,
			"duration_ms", res.Audit.Duration.Milliseconds(),
		)
		w.Header().Set("X-Goat-Translator-Source-Format", from)
		w.Header().Set("X-Goat-Translator-Target-Format", to)
		w.Header().Set("X-Goat-Translator-Canonical-Id", res.Audit.CanonicalID)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(res.Output)
	}
}

func detectHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		mode := pipeline.DetectFirst
		switch r.URL.Query().Get("mode") {
		case "strict":
			mode = pipeline.DetectStrict
		case "all":
			mode = pipeline.DetectAll
		case "", "first":
			mode = pipeline.DetectFirst
		default:
			httpError(w, http.StatusBadRequest, "mode= must be first|strict|all")
			return
		}
		input, err := readBody(r, cfg.MaxBodyBytes)
		if err != nil {
			httpError(w, http.StatusBadRequest, err.Error())
			return
		}
		matches, err := pipeline.Detect(input, mode)
		if err != nil {
			cfg.Logger.Info("detect: no match", "err", err)
			writeJSON(w, http.StatusOK, map[string]any{"matches": matches, "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"matches": matches})
	}
}

func formatsHandler(_ Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpError(w, http.StatusMethodNotAllowed, "GET required")
			return
		}
		all := adapter.All()
		out := make([]map[string]string, 0, len(all))
		for _, a := range all {
			out = append(out, map[string]string{
				"name":        a.Name(),
				"description": a.Description(),
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"formats": out})
	}
}

func healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpError(w, http.StatusMethodNotAllowed, "GET required")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}
}

func readBody(r *http.Request, maxBytes int64) ([]byte, error) {
	if r.Body == nil {
		return nil, fmt.Errorf("empty request body")
	}
	defer r.Body.Close()
	limited := http.MaxBytesReader(nil, r.Body, maxBytes)
	b, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if len(b) == 0 {
		return nil, fmt.Errorf("empty request body")
	}
	return b, nil
}

func httpError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
