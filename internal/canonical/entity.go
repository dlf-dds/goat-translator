// Package canonical defines the placeholder canonical entity model used as
// the pivot between every adapter's wire format. See docs/CANONICAL.md for
// the placeholder posture — this package is replaced wholesale when goat's
// data dictionary entity schema firms up at goat trunk Block 23.
package canonical

import (
	"errors"
	"time"
)

// Kind enumerates the canonical entity kinds. Every adapter maps its native
// concepts to one of these. New kinds are added as the data-dictionary
// schema firms up; adapters never invent kinds locally.
type Kind string

const (
	KindUnknown     Kind = ""
	KindTrack       Kind = "track"
	KindAsset       Kind = "asset"
	KindObservation Kind = "observation"
	KindDetection   Kind = "detection"
	KindAlert       Kind = "alert"
	KindTask        Kind = "task"
)

// Position is a geographic point.
type Position struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	Altitude  float64 `json:"alt_m,omitempty"` // meters above WGS84 ellipsoid
}

// Provenance records where this entity came from. Every adapter populates
// this on Decode so the canonical entity is auditable downstream.
type Provenance struct {
	// SourceFormat is the lowercase format name ("cot", "lattice", etc.).
	// Matches the Adapter.Name() of the adapter that produced this entity.
	SourceFormat string `json:"source_format"`

	// SourceID is the format-native identifier from the wire payload, if
	// the format provided one (CoT uid, Lattice entity_id, etc.). Empty
	// when not present.
	SourceID string `json:"source_id,omitempty"`

	// SourceVersion is the format version asserted by the adapter, if
	// known. Empty when unspecified.
	SourceVersion string `json:"source_version,omitempty"`

	// DecodedAt is when the canonical entity was constructed. Set by the
	// pipeline, not the adapter.
	DecodedAt time.Time `json:"decoded_at"`
}

// Entity is the canonical entity placeholder. Real entity model lives in
// goat's data dictionary (Block 23); this struct is replaced when that
// firms up. Adapters write to / read from these fields; format-specific
// content that doesn't map cleanly goes in Attributes.
type Entity struct {
	// ID is a UUID-shaped canonical identifier. The pipeline mints this
	// if the adapter's Decode does not provide one. Adapters populating
	// ID directly must guarantee uniqueness within the source corpus.
	ID string `json:"id"`

	// Kind classifies the entity. KindUnknown is permitted but adapters
	// should populate when they can.
	Kind Kind `json:"kind"`

	// Time is the source timestamp from the wire payload. Different from
	// Provenance.DecodedAt (when we read the bytes). Empty if the format
	// did not supply a timestamp.
	Time time.Time `json:"time,omitempty"`

	// Position is set when the format supplied geographic information.
	// Nil when not applicable (alerts, abstract tasks, etc.).
	Position *Position `json:"position,omitempty"`

	// Provenance is mandatory — every entity carries the format that
	// produced it. Set by the pipeline if the adapter does not set it.
	Provenance Provenance `json:"provenance"`

	// Attributes is a free-form key-value bag for format-specific fields
	// that have no first-class canonical mapping yet. When a field
	// becomes common across formats, it gets promoted out of Attributes
	// into a first-class field — that promotion is a goat trunk Block
	// 23 (data dictionary) decision, not a translator decision.
	Attributes map[string]any `json:"attributes,omitempty"`
}

// Validate runs minimal canonical-invariant checks. Adapters should call
// this from their Decode before returning. The pipeline also calls this
// before invoking Encode on the target adapter.
func (e Entity) Validate() error {
	if e.ID == "" {
		return errors.New("canonical entity: ID is required")
	}
	if e.Provenance.SourceFormat == "" {
		return errors.New("canonical entity: provenance.source_format is required")
	}
	if e.Position != nil {
		if e.Position.Latitude < -90 || e.Position.Latitude > 90 {
			return errors.New("canonical entity: position.lat out of range")
		}
		if e.Position.Longitude < -180 || e.Position.Longitude > 180 {
			return errors.New("canonical entity: position.lon out of range")
		}
	}
	return nil
}
