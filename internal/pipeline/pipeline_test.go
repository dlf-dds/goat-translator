package pipeline

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/dlf-dds/goat-translator/internal/adapter"
	"github.com/dlf-dds/goat-translator/internal/canonical"
)

// passthroughAdapter is a minimal test adapter that round-trips bytes
// through a canonical entity whose Attributes hold the input verbatim.
type passthroughAdapter struct{ name string }

func (p *passthroughAdapter) Name() string        { return p.name }
func (p *passthroughAdapter) Description() string { return "passthrough " + p.name }

func (p *passthroughAdapter) Decode(input []byte) (canonical.Entity, error) {
	return canonical.Entity{
		ID:   p.name + "-fixture",
		Kind: canonical.KindObservation,
		Provenance: canonical.Provenance{
			SourceFormat: p.name,
		},
		Attributes: map[string]any{"raw": string(input)},
	}, nil
}

func (p *passthroughAdapter) Encode(e canonical.Entity) ([]byte, error) {
	if v, ok := e.Attributes["raw"].(string); ok {
		return []byte(v), nil
	}
	return []byte(e.ID), nil
}

func (p *passthroughAdapter) Detect(input []byte) bool {
	return bytes.HasPrefix(input, []byte(p.name+":"))
}

// failingDecoder always errors on Decode.
type failingDecoder struct{ name string }

func (f *failingDecoder) Name() string                              { return f.name }
func (f *failingDecoder) Description() string                       { return "fails " + f.name }
func (f *failingDecoder) Decode(_ []byte) (canonical.Entity, error) { return canonical.Entity{}, errors.New("nope") }
func (f *failingDecoder) Encode(_ canonical.Entity) ([]byte, error) { return nil, nil }
func (f *failingDecoder) Detect(_ []byte) bool                      { return false }

func setup(t *testing.T) {
	t.Helper()
	adapter.ResetForTest()
}

func TestTranslateRoundTrip(t *testing.T) {
	setup(t)
	adapter.Register(&passthroughAdapter{name: "alpha"})
	adapter.Register(&passthroughAdapter{name: "bravo"})

	res, err := Translate([]byte("hello"), "alpha", "bravo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(res.Output) != "hello" {
		t.Fatalf("output = %q, want %q", string(res.Output), "hello")
	}
	if res.Audit.SourceFormat != "alpha" || res.Audit.TargetFormat != "bravo" {
		t.Fatalf("audit formats wrong: %+v", res.Audit)
	}
	if res.Audit.InputSize != 5 || res.Audit.OutputSize != 5 {
		t.Fatalf("audit sizes wrong: %+v", res.Audit)
	}
	if res.Audit.InputHash == "" || res.Audit.OutputHash == "" {
		t.Fatalf("audit hashes missing: %+v", res.Audit)
	}
	if res.Audit.CanonicalID != "alpha-fixture" {
		t.Fatalf("audit canonical_id = %q, want %q", res.Audit.CanonicalID, "alpha-fixture")
	}
}

func TestTranslateToCanonical(t *testing.T) {
	setup(t)
	adapter.Register(&passthroughAdapter{name: "alpha"})

	res, err := Translate([]byte("hello"), "alpha", "canonical")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(res.Output), "alpha-fixture") {
		t.Fatalf("canonical output missing fixture id: %s", res.Output)
	}
	if !strings.Contains(string(res.Output), `"source_format": "alpha"`) {
		t.Fatalf("canonical output missing provenance: %s", res.Output)
	}
}

func TestTranslateEmptyInput(t *testing.T) {
	setup(t)
	adapter.Register(&passthroughAdapter{name: "alpha"})
	_, err := Translate(nil, "alpha", "canonical")
	if !errors.Is(err, ErrEmptyInput) {
		t.Fatalf("expected ErrEmptyInput, got %v", err)
	}
}

func TestTranslateUnknownSource(t *testing.T) {
	setup(t)
	_, err := Translate([]byte("hello"), "nope", "canonical")
	if !errors.Is(err, adapter.ErrUnknownFormat) {
		t.Fatalf("expected ErrUnknownFormat, got %v", err)
	}
}

func TestTranslateSameFormat(t *testing.T) {
	setup(t)
	adapter.Register(&passthroughAdapter{name: "alpha"})
	_, err := Translate([]byte("hello"), "alpha", "alpha")
	if !errors.Is(err, ErrSameFormat) {
		t.Fatalf("expected ErrSameFormat, got %v", err)
	}
}

func TestTranslateDecodeFailureCarriesErrorIntoAudit(t *testing.T) {
	setup(t)
	adapter.Register(&failingDecoder{name: "broken"})
	res, err := Translate([]byte("hello"), "broken", "canonical")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if res.Audit.Err == "" {
		t.Fatal("audit Err not set on failure")
	}
}

func TestDetect(t *testing.T) {
	setup(t)
	adapter.Register(&passthroughAdapter{name: "alpha"})
	adapter.Register(&passthroughAdapter{name: "bravo"})

	got, err := Detect([]byte("alpha:hello"), DetectFirst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != "alpha" {
		t.Fatalf("Detect = %v, want [alpha]", got)
	}
}

func TestDetectStrictMultiple(t *testing.T) {
	setup(t)
	// Two adapters that both claim every input.
	a := &alwaysDetect{name: "alpha"}
	b := &alwaysDetect{name: "bravo"}
	adapter.Register(a)
	adapter.Register(b)

	_, err := Detect([]byte("hello"), DetectStrict)
	if !errors.Is(err, ErrMultipleDetected) {
		t.Fatalf("expected ErrMultipleDetected, got %v", err)
	}
}

func TestDetectNoMatch(t *testing.T) {
	setup(t)
	adapter.Register(&passthroughAdapter{name: "alpha"})

	_, err := Detect([]byte("nothing claims this"), DetectFirst)
	if !errors.Is(err, ErrNoFormatDetected) {
		t.Fatalf("expected ErrNoFormatDetected, got %v", err)
	}
}

func TestValidate(t *testing.T) {
	setup(t)
	adapter.Register(&passthroughAdapter{name: "alpha"})

	if err := Validate([]byte("hello"), "alpha"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateUnknownFormat(t *testing.T) {
	setup(t)
	if err := Validate([]byte("hello"), "nope"); !errors.Is(err, adapter.ErrUnknownFormat) {
		t.Fatalf("expected ErrUnknownFormat, got %v", err)
	}
}

type alwaysDetect struct{ name string }

func (a *alwaysDetect) Name() string                                    { return a.name }
func (a *alwaysDetect) Description() string                             { return "always-detect " + a.name }
func (a *alwaysDetect) Decode(_ []byte) (canonical.Entity, error)       { return canonical.Entity{ID: a.name, Provenance: canonical.Provenance{SourceFormat: a.name}}, nil }
func (a *alwaysDetect) Encode(_ canonical.Entity) ([]byte, error)       { return nil, nil }
func (a *alwaysDetect) Detect(_ []byte) bool                            { return true }
