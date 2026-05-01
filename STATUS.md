# Status — goat-translator maturity

| Tier | Meaning |
|---|---|
| **proposed** | Repo created, scaffolding only, no working translation yet. |
| **scaffolding** | Engine + CLI + HTTP server framework in place; echo adapter only. No real format adapters. |
| **pilot** | At least one open-standards adapter (CoT or similar) translating end-to-end with tests + a measurement entry. |
| **production** | Used in at least one operational deployment with documented evidence (runbook + soak verdict + measurement). |

## Current

| Item | Tier | Notes |
|---|---|---|
| Repo + scaffolding | scaffolding | Go module, Makefile, README, ARCHITECTURE doc, build-tags pattern. |
| Engine: canonical entity placeholder + adapter interface + registry + pipeline | scaffolding | Framework in place; canonical entity model is a placeholder until goat data dictionary firms up. |
| CLI (cobra) | scaffolding | Subcommands: `translate`, `detect`, `list-formats`, `validate`, `serve`. |
| HTTP server | scaffolding | Endpoints: `/v1/translate`, `/v1/detect`, `/v1/formats`, `/healthz`, `/metrics`. |
| Echo adapter | scaffolding | Passthrough reference adapter for smoke testing the pipeline. |
| Open-standards adapters (CoT, SAPIENT, ISA, OMNI) | not started | Implemented after [`goat-open-standards`](https://github.com/dlf-dds/goat-open-standards) provides Go bindings. |
| Vendor SDK adapters (Lattice, Picogrid) | not started | Build-tag-gated; live in their respective private repos. |
| CI (build + test + lint + scan) | scaffolding | GitHub Actions workflow for default build; vendor-tag builds documented but not run in default CI (require private-repo access). |
| Operational deployment evidence | none | No deployments yet. |

## Sequencing

1. **goat-shim-framework** (goat trunk Block 49A) lands first — substrate every adapter depends on.
2. **First open-standards adapter** lands in `goat-open-standards`, then is wired into goat-translator. Likely CoT first (already has reference shim in goat trunk).
3. **Vendor adapters** land in their respective private repos, build-tag-gated into goat-translator only at authorized-operator build time.
4. **First measurement entry** (latency for a representative translation pair, file-batch throughput) when the first real adapter pair is end-to-end working.

## Change history

| Date | Change |
|---|---|
| 2026-05-01 | Initial — repo created with scaffolding, engine framework, CLI + HTTP skeleton, echo adapter, build-tags pattern. No format adapters yet. |
