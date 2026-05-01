# Build tags — public vs vendor-authorized builds

The translator's adapters split into two classes:

- **Open-standards adapters** (CoT, SAPIENT, ISA, OMNI) are built into this repo by default. The default `make build` produces a binary that handles all open standards.
- **Commercial vendor SDK adapters** (Anduril Lattice, Picogrid ECN, future Palantir, etc.) live in their respective private vendor-shim repos. They are loaded into the translator binary **only at build time when the operator builds with the appropriate Go build tag** and has the vendor's private repo cloned and accessible.

This pattern keeps three properties:

1. **The default public binary cannot accidentally include vendor-proprietary adapter code.** A stranger building from this public repo gets the open-standards binary; they cannot produce a Lattice or Picogrid binary because the vendor adapter packages aren't in this repo.
2. **The per-class repo containment from [ADR 0217](https://github.com/dlf-dds/DesertBreadBird/blob/main/docs/adr/0217-shim-repo-containment.md) is preserved.** Vendor adapter code stays in vendor private repos; the translator composes them.
3. **Authorized operators can produce a single binary that handles every format they have rights to.** No per-vendor binary; one binary per build configuration.

## How it works

Each vendor's adapter lives in its private repo at a known path:

- `github.com/dlf-dds/goat-lattice-shim/translator-adapter` (when written)
- `github.com/dlf-dds/goat-picogrid-shim/translator-adapter` (when written)

In the translator's `cmd/goat-translator/main.go`, vendor adapters are imported behind build tags:

```go
//go:build lattice

package main

import _ "github.com/dlf-dds/goat-lattice-shim/translator-adapter"
```

```go
//go:build picogrid

package main

import _ "github.com/dlf-dds/goat-picogrid-shim/translator-adapter"
```

The blank imports trigger the adapter's `init()` registration. Without the build tag, the file is excluded from the build entirely and the adapter is not registered.

## Building

```bash
# Default — open-standards adapters only
make build
# → bin/goat-translator (handles cot, sapient, isa, omni)

# With Anduril Lattice
make build-with-lattice
# → bin/goat-translator (handles cot, sapient, isa, omni, lattice)
# Requires the operator to have authorized access to dlf-dds/goat-lattice-shim
# and to have it cloned at $GOPATH/src/github.com/dlf-dds/goat-lattice-shim
# (or available via Go modules with appropriate auth).

# With Picogrid ECN
make build-with-picogrid
# Requires authorized access to dlf-dds/goat-picogrid-shim (private,
# government-internal access only).

# With all available vendor adapters
make build-authorized
# → bin/goat-translator (every format the operator has rights to)
```

## What goes wrong if a vendor adapter is missing

If you run `make build-with-lattice` but don't have access to `dlf-dds/goat-lattice-shim`, the build fails at `go mod download` time with a "module not found / access denied" error. The build will not silently degrade to an open-standards-only binary. This is intentional — silent omission would let an authorized operator think they had the Lattice adapter when they don't.

## Vendor-adapter authoring contract

When authoring a vendor adapter in a private vendor-shim repo, the conventions are:

- Adapter package path: `github.com/dlf-dds/goat-<vendor>-shim/translator-adapter`
- Adapter's `init()` calls `adapter.Register(&Adapter{})` from `github.com/dlf-dds/goat-translator/internal/adapter`
- Adapter implements the `Adapter` interface defined in [`internal/adapter/adapter.go`](../internal/adapter/adapter.go)
- Adapter's `Name()` returns the lowercase vendor name (`"lattice"`, `"picogrid"`, etc.)
- Adapter has its own unit tests inside the vendor-shim repo
- Adapter follows the [goat shim-pattern](https://github.com/dlf-dds/DesertBreadBird/blob/main/docs/design/shim-pattern.md) anti-patterns (no business logic, no per-deployment config baked in, etc.)

The translator imports the adapter; the adapter imports `goat-translator/internal/adapter` for its registration. There is no other coupling — vendor adapters do not depend on translator-engine internals beyond the `Adapter` interface.
