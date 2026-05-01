# Architecture

## The canonical-as-pivot pattern

Every format the translator handles defines two functions:

- `Decode(input []byte) (CanonicalEntity, error)` — parses the format's wire bytes into goat's canonical entity model.
- `Encode(entity CanonicalEntity) ([]byte, error)` — emits a CanonicalEntity in the format's wire bytes.

Translation between two formats is the composition of two adapters via the canonical pivot:

```text
input bytes → format-A.Decode() → CanonicalEntity → format-B.Encode() → output bytes
```

This is the standard hub-and-spoke pattern for format translation: N formats need N adapters (one per format) rather than N×(N-1) direct translators (one per pair). Adding a new format means writing one adapter, not retrofitting every existing format to translate to/from it.

## Why a service if the fabric already does this?

The goat fabric already implements canonical-as-pivot through per-format shims publishing to canonical topics (`entity/track/<id>`, `sensor/detection/<id>`, etc.). Subscribing to a canonical topic gets you data from every format-shim that publishes to it; the shim handles its native format inbound, the canonical layer handles the cross-format dispatch, and any consumer can subscribe.

The translator service exists for **off-fabric use cases** — situations where the goat fabric isn't running, isn't in scope, or isn't appropriate for the integration pattern:

- CLI batch translation of files
- Demo / training tools
- Operational debugging without mesh enrollment
- Partner exchange where the partner doesn't have a goat peer

For on-fabric integration, prefer the canonical-pivot through the shims directly. The service is a complement, not a replacement.

## Components

```text
goat-translator/
├── cmd/goat-translator/        # CLI entry point (cobra)
│
├── internal/
│   ├── canonical/              # CanonicalEntity placeholder + helpers
│   ├── adapter/                # Adapter interface + Registry
│   ├── pipeline/               # Translate, Detect, Validate
│   ├── config/                 # YAML config loader
│   └── server/                 # HTTP server (REST endpoints)
│
└── adapters/                   # Built-in adapters
    ├── echo/                   # Passthrough reference adapter (smoke test)
    └── (open-standards adapters land here when goat-open-standards provides bindings)
```

Vendor adapters do not live in this repo. They live in their respective private repos (`dlf-dds/goat-lattice-shim`, `dlf-dds/goat-picogrid-shim`) and are loaded into the translator only at build time when authorized via Go build tags. See [BUILD-TAGS.md](BUILD-TAGS.md).

## The Adapter contract

```go
package adapter

import "github.com/dlf-dds/goat-translator/internal/canonical"

// Adapter is the contract every format implements.
//
// Adapters are stateless. Adapter implementations must be safe for
// concurrent use by multiple goroutines.
type Adapter interface {
    // Name returns the canonical, lowercase name of the format
    // ("cot", "lattice", "sapient", "picogrid", etc.). Used as the
    // CLI/HTTP identifier; must match the registry key.
    Name() string

    // Description returns a short human-readable description for
    // the `list-formats` CLI surface.
    Description() string

    // Decode parses input bytes in this adapter's wire format into
    // a CanonicalEntity. Errors are wrapped with adapter context.
    Decode(input []byte) (canonical.Entity, error)

    // Encode serializes a CanonicalEntity into this adapter's wire
    // format. Errors are wrapped with adapter context.
    Encode(entity canonical.Entity) ([]byte, error)

    // Detect returns true if input bytes are likely in this adapter's
    // format. Used by the detect subcommand. Detection is best-effort;
    // a true return is not a guarantee that Decode will succeed.
    Detect(input []byte) bool
}
```

Adapters register themselves at package init time:

```go
package myformat

import "github.com/dlf-dds/goat-translator/internal/adapter"

func init() {
    adapter.Register(&Adapter{})
}
```

## Format detection

`Detect` is a best-effort heuristic. The pipeline's `Detect` function tries every registered adapter's `Detect` and returns the first match (or all matches, behind a flag). Adapters with stronger detection signals (e.g. magic bytes, schema-validated parse) come first; adapters with weaker signals (e.g. free-form JSON) come last.

For the CLI's `detect` subcommand, the heuristic order is configurable via `--strict` (single adapter must claim the input) vs `--first` (first adapter to claim wins) vs `--all` (return all claims, let caller pick).

## Validation

`Validate` is `Decode` followed by canonical-entity invariant checks. It returns success without producing output. Used by the CLI's `validate` subcommand for "is this payload well-formed?" without committing to a target format.

## Stateless service posture

The translator holds no state across requests. Configuration is loaded at startup; adapters are stateless; the canonical entity is constructed and discarded per translation. This matches goat's phoenix-native peer discipline ([phoenix-architecture §2A](https://github.com/dlf-dds/DesertBreadBird/blob/main/docs/design/phoenix-architecture.md)) — restart is always safe, no persistence on the translation path.

When the translator runs as a fabric peer (future work), it follows the [shim pattern §2.4 restart semantics](https://github.com/dlf-dds/DesertBreadBird/blob/main/docs/design/shim-pattern.md) and depends on `goat-shim-framework` for identity, audit, and namespace policing.

## Audit

Every translation emits an audit record (when running in fabric-peer mode) per [ADR 0508](https://github.com/dlf-dds/DesertBreadBird/blob/main/docs/adr/0508-audit-logger-interceptor.md). For CLI use, audit is per-invocation stderr unless directed elsewhere.

The audit record includes:

- Source format name + version (if known)
- Target format name + version (if known)
- Input size + hash
- Output size + hash
- Adapter library versions
- Translation timestamp + duration
- Errors (if any)

Audit is part of the contract: the translator does not translate silently.
