// Command goat-translator is the CLI + HTTP entry point for the goat
// format translator.
//
// Subcommands:
//
//	goat-translator translate --from <fmt> --to <fmt>
//	goat-translator detect [--mode first|strict|all]
//	goat-translator list-formats
//	goat-translator validate --format <fmt>
//	goat-translator serve [--listen 127.0.0.1:8080]
//
// Build tags select which adapters are compiled in. See docs/BUILD-TAGS.md.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/dlf-dds/goat-translator/internal/adapter"
	"github.com/dlf-dds/goat-translator/internal/pipeline"
	"github.com/dlf-dds/goat-translator/internal/server"

	// The echo passthrough adapter ships in every build (no build tag).
	_ "github.com/dlf-dds/goat-translator/adapters/echo"

	// Open-standards adapters are imported here when the
	// goat-open-standards repo provides Go bindings. Until then this
	// section stays empty.

	// Vendor SDK adapters are imported in build-tag-gated files alongside
	// this main file. See cmd/goat-translator/lattice_imports.go,
	// picogrid_imports.go, etc. (created when the corresponding adapter
	// repos are ready).
)

var (
	version    = "dev"
	buildTags  = "default"
	auditOut   = os.Stderr
)

func main() {
	root := &cobra.Command{
		Use:   "goat-translator",
		Short: "Format translator service for goat fabric (canonical-as-pivot)",
		Long: `goat-translator translates entity / object / sensor data between
data standards (CoT, SAPIENT, ISA, OMNI) and vendor SDKs (Lattice,
Picogrid via authorized builds).

Default build includes open-standards adapters only. Vendor adapters are
build-tag-gated; see docs/BUILD-TAGS.md.`,
	}
	root.AddCommand(translateCmd(), detectCmd(), listFormatsCmd(), validateCmd(), serveCmd(), versionCmd())
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func translateCmd() *cobra.Command {
	var from, to, inFile, outFile string
	cmd := &cobra.Command{
		Use:   "translate",
		Short: "Translate stdin (or --in) from --from to --to (or 'canonical')",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if from == "" || to == "" {
				return errors.New("--from and --to are required")
			}
			input, err := readInput(inFile)
			if err != nil {
				return err
			}
			res, err := pipeline.Translate(input, from, to)
			emitAudit(res.Audit)
			if err != nil {
				return err
			}
			return writeOutput(outFile, res.Output)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "source format (required)")
	cmd.Flags().StringVar(&to, "to", "", "target format (required); use 'canonical' to emit the canonical entity as JSON")
	cmd.Flags().StringVar(&inFile, "in", "", "read input from this file (default stdin)")
	cmd.Flags().StringVar(&outFile, "out", "", "write output to this file (default stdout)")
	return cmd
}

func detectCmd() *cobra.Command {
	var mode, inFile string
	cmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect the format of stdin (or --in)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			input, err := readInput(inFile)
			if err != nil {
				return err
			}
			var m pipeline.DetectMode
			switch mode {
			case "first", "":
				m = pipeline.DetectFirst
			case "strict":
				m = pipeline.DetectStrict
			case "all":
				m = pipeline.DetectAll
			default:
				return fmt.Errorf("--mode must be first|strict|all")
			}
			matches, err := pipeline.Detect(input, m)
			out := map[string]any{"matches": matches}
			if err != nil {
				out["error"] = err.Error()
			}
			return json.NewEncoder(os.Stdout).Encode(out)
		},
	}
	cmd.Flags().StringVar(&mode, "mode", "first", "detection mode: first|strict|all")
	cmd.Flags().StringVar(&inFile, "in", "", "read input from this file (default stdin)")
	return cmd
}

func listFormatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-formats",
		Short: "List every adapter compiled into this binary",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := map[string]any{
				"build_tags": buildTags,
				"formats":    []map[string]string{},
			}
			for _, a := range adapter.All() {
				out["formats"] = append(out["formats"].([]map[string]string), map[string]string{
					"name":        a.Name(),
					"description": a.Description(),
				})
			}
			return json.NewEncoder(os.Stdout).Encode(out)
		},
	}
}

func validateCmd() *cobra.Command {
	var format, inFile string
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Decode stdin (or --in) as --format and validate canonical invariants without producing output",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if format == "" {
				return errors.New("--format is required")
			}
			input, err := readInput(inFile)
			if err != nil {
				return err
			}
			if err := pipeline.Validate(input, format); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "ok")
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "", "format to validate against (required)")
	cmd.Flags().StringVar(&inFile, "in", "", "read input from this file (default stdin)")
	return cmd
}

func serveCmd() *cobra.Command {
	var listen string
	var maxBody int64
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run as an HTTP server (REST endpoints + Prometheus metrics)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
			handler := server.New(server.Config{
				Listen:       listen,
				MaxBodyBytes: maxBody,
				Logger:       logger,
			})
			srv := &http.Server{
				Addr:              listen,
				Handler:           handler,
				ReadHeaderTimeout: 5 * time.Second,
				ReadTimeout:       30 * time.Second,
				WriteTimeout:      30 * time.Second,
				IdleTimeout:       60 * time.Second,
			}
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			errCh := make(chan error, 1)
			go func() {
				logger.Info("translator listening", "addr", listen, "build_tags", buildTags, "version", version)
				if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					errCh <- err
				}
				close(errCh)
			}()
			select {
			case <-ctx.Done():
				logger.Info("shutdown signal received")
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				return srv.Shutdown(shutdownCtx)
			case err := <-errCh:
				return err
			}
		},
	}
	cmd.Flags().StringVar(&listen, "listen", "127.0.0.1:8080", "address to listen on")
	cmd.Flags().Int64Var(&maxBody, "max-body-bytes", 8<<20, "maximum HTTP request body size")
	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version + build-tags information",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Printf("goat-translator %s (build_tags=%s)\n", version, buildTags)
		},
	}
}

func readInput(path string) ([]byte, error) {
	if path == "" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

func writeOutput(path string, b []byte) error {
	if path == "" {
		_, err := os.Stdout.Write(b)
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func emitAudit(rec pipeline.AuditRecord) {
	enc := json.NewEncoder(auditOut)
	_ = enc.Encode(struct {
		Channel string                  `json:"channel"`
		Audit   pipeline.AuditRecord    `json:"audit"`
	}{Channel: "audit", Audit: rec})
}
