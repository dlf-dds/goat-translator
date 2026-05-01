package pipeline

import (
	"encoding/json"

	"github.com/dlf-dds/goat-translator/internal/canonical"
)

// canonicalJSON marshals a canonical entity to indented JSON. Used when
// the target format is the special name "canonical".
func canonicalJSON(e canonical.Entity) ([]byte, error) {
	return json.MarshalIndent(e, "", "  ")
}
