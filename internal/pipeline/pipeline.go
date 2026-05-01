// Package pipeline implements the translate / detect / validate operations
// over the adapter registry. It is the engine that composes adapters into
// the canonical-as-pivot translation pattern.
//
// Pipeline operations are stateless and concurrency-safe.
package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/dlf-dds/goat-translator/internal/adapter"
	"github.com/dlf-dds/goat-translator/internal/canonical"
)

// Result is the outcome of a Translate call. It carries the output bytes
// plus an audit record. When the translator runs in fabric-peer mode the
// audit record is published per ADR 0508; for CLI use it is emitted to
// stderr.
type Result struct {
	Output []byte
	Audit  AuditRecord
}

// AuditRecord captures one translation event for the audit pipeline.
type AuditRecord struct {
	SourceFormat   string        `json:"source_format"`
	TargetFormat   string        `json:"target_format"`
	InputSize      int           `json:"input_size_bytes"`
	OutputSize     int           `json:"output_size_bytes"`
	InputHash      string        `json:"input_sha256"`
	OutputHash     string        `json:"output_sha256"`
	StartedAt      time.Time     `json:"started_at"`
	Duration       time.Duration `json:"duration"`
	CanonicalKind  canonical.Kind `json:"canonical_kind,omitempty"`
	CanonicalID    string        `json:"canonical_id,omitempty"`
	Err            string        `json:"error,omitempty"`
}

// Errors returned by pipeline operations.
var (
	ErrEmptyInput        = errors.New("pipeline: empty input")
	ErrSameFormat        = errors.New("pipeline: source and target formats are identical (use --to canonical for round-trip testing)")
	ErrNoFormatDetected  = errors.New("pipeline: no registered adapter detected this input")
	ErrMultipleDetected  = errors.New("pipeline: multiple adapters detected this input")
)

// Translate composes source.Decode → canonical → target.Encode. The
// canonical entity returned by Decode is validated before being passed to
// Encode. Provenance fields are populated by the pipeline if the source
// adapter did not set them.
func Translate(input []byte, sourceName, targetName string) (Result, error) {
	started := time.Now()
	rec := AuditRecord{
		SourceFormat: sourceName,
		TargetFormat: targetName,
		InputSize:    len(input),
		InputHash:    sha256Hex(input),
		StartedAt:    started,
	}

	if len(input) == 0 {
		rec.Err = ErrEmptyInput.Error()
		rec.Duration = time.Since(started)
		return Result{Audit: rec}, ErrEmptyInput
	}

	source, err := adapter.Get(sourceName)
	if err != nil {
		rec.Err = err.Error()
		rec.Duration = time.Since(started)
		return Result{Audit: rec}, fmt.Errorf("source: %w", err)
	}

	entity, err := source.Decode(input)
	if err != nil {
		rec.Err = err.Error()
		rec.Duration = time.Since(started)
		return Result{Audit: rec}, fmt.Errorf("decode: %w", err)
	}

	// Pipeline backfills provenance the adapter did not set.
	if entity.Provenance.SourceFormat == "" {
		entity.Provenance.SourceFormat = sourceName
	}
	if entity.Provenance.DecodedAt.IsZero() {
		entity.Provenance.DecodedAt = started
	}

	if err := entity.Validate(); err != nil {
		rec.Err = err.Error()
		rec.Duration = time.Since(started)
		return Result{Audit: rec}, fmt.Errorf("canonical validation: %w", err)
	}

	rec.CanonicalKind = entity.Kind
	rec.CanonicalID = entity.ID

	// Special target name "canonical" emits the canonical entity directly
	// as JSON. Useful for debugging and for adapters that want to inspect
	// the pivot without re-encoding.
	if targetName == "canonical" {
		out, err := canonicalJSON(entity)
		if err != nil {
			rec.Err = err.Error()
			rec.Duration = time.Since(started)
			return Result{Audit: rec}, fmt.Errorf("canonical encode: %w", err)
		}
		rec.OutputSize = len(out)
		rec.OutputHash = sha256Hex(out)
		rec.Duration = time.Since(started)
		return Result{Output: out, Audit: rec}, nil
	}

	if sourceName == targetName {
		rec.Err = ErrSameFormat.Error()
		rec.Duration = time.Since(started)
		return Result{Audit: rec}, ErrSameFormat
	}

	target, err := adapter.Get(targetName)
	if err != nil {
		rec.Err = err.Error()
		rec.Duration = time.Since(started)
		return Result{Audit: rec}, fmt.Errorf("target: %w", err)
	}

	out, err := target.Encode(entity)
	if err != nil {
		rec.Err = err.Error()
		rec.Duration = time.Since(started)
		return Result{Audit: rec}, fmt.Errorf("encode: %w", err)
	}

	rec.OutputSize = len(out)
	rec.OutputHash = sha256Hex(out)
	rec.Duration = time.Since(started)
	return Result{Output: out, Audit: rec}, nil
}

// DetectMode controls how the Detect operation handles ambiguity.
type DetectMode int

const (
	// DetectFirst returns the first adapter (alphabetically) that claims
	// the input. Useful for quick interactive use.
	DetectFirst DetectMode = iota

	// DetectStrict returns ErrMultipleDetected if more than one adapter
	// claims the input.
	DetectStrict

	// DetectAll returns every adapter that claims the input.
	DetectAll
)

// Detect runs every registered adapter's Detect over the input and returns
// matching adapter names per the selected mode.
func Detect(input []byte, mode DetectMode) ([]string, error) {
	if len(input) == 0 {
		return nil, ErrEmptyInput
	}
	matches := []string{}
	for _, a := range adapter.All() {
		if a.Detect(input) {
			matches = append(matches, a.Name())
		}
	}
	switch mode {
	case DetectStrict:
		if len(matches) == 0 {
			return nil, ErrNoFormatDetected
		}
		if len(matches) > 1 {
			return matches, ErrMultipleDetected
		}
		return matches, nil
	case DetectAll:
		if len(matches) == 0 {
			return nil, ErrNoFormatDetected
		}
		return matches, nil
	case DetectFirst:
		fallthrough
	default:
		if len(matches) == 0 {
			return nil, ErrNoFormatDetected
		}
		return matches[:1], nil
	}
}

// Validate runs the source adapter's Decode and checks canonical
// invariants without producing output. Used by the validate subcommand.
func Validate(input []byte, sourceName string) error {
	if len(input) == 0 {
		return ErrEmptyInput
	}
	source, err := adapter.Get(sourceName)
	if err != nil {
		return fmt.Errorf("source: %w", err)
	}
	entity, err := source.Decode(input)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	if entity.Provenance.SourceFormat == "" {
		entity.Provenance.SourceFormat = sourceName
	}
	if err := entity.Validate(); err != nil {
		return fmt.Errorf("canonical validation: %w", err)
	}
	return nil
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
