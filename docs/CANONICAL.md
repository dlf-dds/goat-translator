# Canonical entity model — placeholder

**Status: PLACEHOLDER as of 2026-05-01.** The canonical entity model in this repo is a minimal placeholder. The real canonical entity model is goat's data dictionary entity model, which is in active development at goat trunk Block 23.

When goat's data dictionary firms up its canonical entity schema, this package will be replaced with imports from the data-dictionary repo. The placeholder here exists so the engine can be built end-to-end before the data dictionary is final.

## Why a placeholder

The translator's value comes from the canonical pivot. The pivot needs a canonical type. The canonical type's *exact* shape doesn't matter for getting the engine running — what matters is that adapters round-trip through it.

The placeholder defines an `Entity` struct with the minimum fields every format we anticipate handling needs:

- Identity (UUID-shaped)
- Kind (track / asset / observation / detection)
- Time (sourced timestamp)
- Position (lat / lon / altitude where applicable)
- Provenance (source, source format)
- Attributes (free-form key-value bag for format-specific fields that don't map cleanly)

The placeholder is **not** a substitute for the data dictionary. When the data dictionary's entity schema lands, the placeholder is replaced wholesale and adapters are updated to map to the real schema.

## What this means for adapter authors

Write your adapter against the placeholder API for now. The placeholder is committed to the public package surface as `internal/canonical/entity.go`. When the real schema lands, the migration is mechanical — the API will be a strictly richer superset.

Specifically:

- If a real entity field has no analog in your format, leave it zero-valued; the canonical model handles "missing" cleanly.
- If your format has a field that has no analog in the canonical model, put it in the `Attributes` bag. When the data dictionary firms up, fields that turn out to be common become first-class.
- Don't invent canonical fields. If the canonical model doesn't have what you need, file a goat trunk issue against Block 23 (data dictionary); don't extend the local placeholder.

## What this means for goat trunk

When the data dictionary's entity schema is published as a Go module, this repo's `internal/canonical/` is replaced with imports from that module. The change should be a clean drop-in at the type level; adapters migrate one at a time as the schema gains fields.

Block 49H tracks the migration sequencing.

## See also

- [goat data dictionary design](https://github.com/dlf-dds/DesertBreadBird/blob/main/docs/design/data-dictionary.md)
- [ADR 0512 — rapid adaptation with AI](https://github.com/dlf-dds/DesertBreadBird/blob/main/docs/adr/0512-rapid-adaptation-with-ai.md) §"Posture" (mandatory schema declaration)
- Block 23 in [implementation-plan.md](https://github.com/dlf-dds/DesertBreadBird/blob/main/docs/project/implementation-plan.md)
