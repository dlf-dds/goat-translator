package canonical

import (
	"strings"
	"testing"
	"time"
)

func TestEntityValidate(t *testing.T) {
	tests := []struct {
		name    string
		entity  Entity
		wantErr string
	}{
		{
			name: "valid minimal entity",
			entity: Entity{
				ID: "abc",
				Provenance: Provenance{
					SourceFormat: "echo",
					DecodedAt:    time.Now(),
				},
			},
		},
		{
			name: "valid with position",
			entity: Entity{
				ID:   "abc",
				Kind: KindTrack,
				Position: &Position{
					Latitude:  37.0,
					Longitude: -122.0,
					Altitude:  100,
				},
				Provenance: Provenance{SourceFormat: "echo"},
			},
		},
		{
			name:    "missing ID",
			entity:  Entity{Provenance: Provenance{SourceFormat: "echo"}},
			wantErr: "ID is required",
		},
		{
			name:    "missing source format",
			entity:  Entity{ID: "abc"},
			wantErr: "provenance.source_format is required",
		},
		{
			name: "lat out of range",
			entity: Entity{
				ID:         "abc",
				Position:   &Position{Latitude: 91, Longitude: 0},
				Provenance: Provenance{SourceFormat: "echo"},
			},
			wantErr: "position.lat out of range",
		},
		{
			name: "lon out of range",
			entity: Entity{
				ID:         "abc",
				Position:   &Position{Latitude: 0, Longitude: -181},
				Provenance: Provenance{SourceFormat: "echo"},
			},
			wantErr: "position.lon out of range",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entity.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}
