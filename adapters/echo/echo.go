// Package echo is a passthrough reference adapter for smoke-testing the
// translator pipeline. It accepts arbitrary input bytes and produces a
// canonical entity whose Attributes hold the input verbatim. Encoding
// retrieves the same bytes from Attributes.
//
// Echo is registered by default in every build (no build tag). It exists
// as a known-good adapter that future format adapters can be diff'd
// against — if the engine works with echo, the engine works.
package echo

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/dlf-dds/goat-translator/internal/adapter"
	"github.com/dlf-dds/goat-translator/internal/canonical"
)

const formatName = "echo"

type Adapter struct{}

func (Adapter) Name() string { return formatName }

func (Adapter) Description() string {
	return "Passthrough reference adapter for smoke testing. Encodes input verbatim into Entity.Attributes[\"raw\"]; decodes by reading the same key."
}

func (Adapter) Decode(input []byte) (canonical.Entity, error) {
	if len(input) == 0 {
		return canonical.Entity{}, fmt.Errorf("echo: empty input")
	}
	h := sha256.Sum256(input)
	return canonical.Entity{
		ID:   "echo-" + hex.EncodeToString(h[:8]),
		Kind: canonical.KindObservation,
		Attributes: map[string]any{
			"raw":      string(input),
			"raw_size": len(input),
		},
		Provenance: canonical.Provenance{
			SourceFormat: formatName,
		},
	}, nil
}

func (Adapter) Encode(e canonical.Entity) ([]byte, error) {
	if v, ok := e.Attributes["raw"].(string); ok {
		return []byte(v), nil
	}
	return nil, fmt.Errorf("echo: canonical entity has no Attributes[\"raw\"]; this entity did not originate from the echo adapter")
}

func (Adapter) Detect(input []byte) bool {
	// Echo is a low-priority detection — it only claims input that
	// starts with the literal "echo:" prefix to avoid stealing detection
	// from real format adapters.
	const prefix = "echo:"
	if len(input) < len(prefix) {
		return false
	}
	return string(input[:len(prefix)]) == prefix
}

func init() {
	adapter.Register(Adapter{})
}
