package echo

import (
	"strings"
	"testing"

	"github.com/dlf-dds/goat-translator/internal/canonical"
)

func TestDecodeEncodeRoundTrip(t *testing.T) {
	a := Adapter{}
	in := []byte("hello world")
	entity, err := a.Decode(in)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if entity.Provenance.SourceFormat != formatName {
		t.Fatalf("source_format = %q, want %q", entity.Provenance.SourceFormat, formatName)
	}
	if !strings.HasPrefix(entity.ID, "echo-") {
		t.Fatalf("ID = %q, want echo- prefix", entity.ID)
	}
	out, err := a.Encode(entity)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if string(out) != "hello world" {
		t.Fatalf("round-trip = %q, want %q", string(out), "hello world")
	}
}

func TestDecodeEmpty(t *testing.T) {
	a := Adapter{}
	if _, err := a.Decode(nil); err == nil {
		t.Fatal("expected error on empty input")
	}
}

func TestEncodeWithoutAttributes(t *testing.T) {
	a := Adapter{}
	if _, err := a.Encode(canonical.Entity{ID: "x"}); err == nil {
		t.Fatal("expected error encoding entity without raw attribute")
	}
}

func TestDetect(t *testing.T) {
	a := Adapter{}
	if !a.Detect([]byte("echo:foo")) {
		t.Fatal("expected detect on echo: prefix")
	}
	if a.Detect([]byte("not for echo")) {
		t.Fatal("expected no detect without prefix")
	}
	if a.Detect([]byte("e")) {
		t.Fatal("expected no detect on too-short input")
	}
}
