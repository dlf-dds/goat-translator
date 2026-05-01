# goat-translator

Format translator service for the **goat** fabric. Translates entity / object / sensor data between data standards and vendor SDKs using a canonical-as-pivot architecture.

## What this is

A standalone CLI + HTTP service that translates between the data standards goat already speaks via shims:

- **Open standards** (built-in by default): CoT, SAPIENT, ISA, OMNI — when implemented in [`dlf-dds/goat-open-standards`](https://github.com/dlf-dds/goat-open-standards).
- **Commercial vendor SDKs** (build-tag-gated): Anduril Lattice, Picogrid ECN, Palantir OSDK — when the operator has authorization and includes the relevant private adapter repo at build time.

## What this is not

- **Not a replacement for the canonical-pivot pattern through the goat fabric.** If a peer is already on the goat fabric, the canonical entity model + per-format shims already deliver format-A-to-format-B translation by construction. This service is for **off-fabric use cases** — see §"When to use this service" below.
- **Not a canonical store.** The translator holds no entity state; it is a stateless pipeline. State lives on the fabric.
- **Not a proprietary-codec hub.** Format adapters live in their respective repos (open-standards repo for open formats, per-vendor private repos for vendor SDKs). The translator binary composes them; it does not vendor them.

## When to use this service

The canonical-pivot pattern through the goat fabric covers most format translation by construction — `entity/track/<id>` is published once, and any format-specific shim can subscribe and emit in its native format. The translator service exists for cases where that pattern doesn't fit:

| Use case | Why the translator is useful |
|---|---|
| **CLI batch translation of files** | Translate a directory of CoT XML files to Lattice protobuf files for archive transfer; no goat fabric required. |
| **Demo / training tools** | Show "this Lattice payload becomes this CoT event" without standing up the full mesh. Useful for partner workshops, ATAK debugging sessions, integration documentation. |
| **Operational debugging without mesh enrollment** | An operator needs to inspect a payload from format X without a goat peer running. `goat-translator translate --from X --to canonical < payload` answers the question. |
| **Partner exchange where the partner has no goat peer** | Translate at the boundary between systems neither side has integrated with the goat fabric yet. Bridge pattern, with the same wire-faithful posture as the per-vendor shims. |

If your case isn't above, the canonical-pivot through the fabric is probably what you want. Don't stand up the translator service for cases the fabric already handles natively.

## Quick start

> **Status: scaffolding only as of 2026-05-01.** The framework is in place; specific format adapters land when format details are provided. See [STATUS.md](STATUS.md).

```bash
# Build (default — open-standards adapters only)
make build

# Build with vendor adapters (requires authorized clones of the private vendor repos)
make build-with-lattice          # adds Anduril Lattice adapter
make build-with-picogrid         # adds Picogrid ECN adapter (requires goat-picogrid-shim access)
make build-authorized            # adds all available vendor adapters

# CLI usage
goat-translator list-formats
goat-translator translate --from cot --to canonical < input.xml > output.json
goat-translator detect < unknown.bin
goat-translator validate --format cot < event.xml

# HTTP server
goat-translator serve --listen 127.0.0.1:8080
```

## Architecture

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the canonical-as-pivot design and adapter contract.

In short: every format has a pair of functions — `Decode([]byte) → CanonicalEntity` and `Encode(CanonicalEntity) → []byte`. The translator pipeline composes them: `Decode(input) → CanonicalEntity → Encode(output)`. The canonical entity model is goat's data-dictionary entity model (placeholder until that firms up; see [docs/CANONICAL.md](docs/CANONICAL.md)).

## Build tags — public vs authorized builds

The default build includes **open-standards adapters only** (CoT, SAPIENT, ISA, OMNI). Vendor SDK adapters live in their respective private repos and are gated behind Go build tags so the public binary cannot accidentally include them.

See [docs/BUILD-TAGS.md](docs/BUILD-TAGS.md) for the full pattern.

## License

Apache 2.0 for translator engine + open-standards adapters built into this repo. See [`LICENSE`](LICENSE).

Vendor SDK adapters (Lattice, Picogrid, etc.) live in their respective private repos under their own license terms and are imported under build tags only by authorized operators.

## Discoverability

- Goat trunk integrations index: [`docs/integrations/INDEX.md`](https://github.com/dlf-dds/DesertBreadBird/blob/main/docs/integrations/INDEX.md)
- Data-standards review (which formats exist, how they relate): [`docs/integrations/data-standards-review.md`](https://github.com/dlf-dds/DesertBreadBird/blob/main/docs/integrations/data-standards-review.md)
- Implementation tracking: [`docs/project/implementation-plan.md`](https://github.com/dlf-dds/DesertBreadBird/blob/main/docs/project/implementation-plan.md) Block 49H

## Status

Scaffolding only as of 2026-05-01. No format adapters yet beyond the echo (passthrough) reference adapter. See [STATUS.md](STATUS.md).
