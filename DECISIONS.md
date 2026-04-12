# Architecture Decision Records

This file documents the key decisions made during the design and implementation of `makakito-config-schema`. Each ADR follows the format: **Context → Decision → Rationale → Consequences**.

---

## ADR-001: JSON Schema as the canonical source of truth

**Status:** Accepted

**Context:** The config contract is consumed by multiple components in different languages (Go CLI and runner, React/TypeScript builder). We need one authoritative definition that all components stay in sync with.

**Decision:** JSON Schema files in `schema/` are the single source of truth. Go structs and any future TypeScript types are derived from — or validated against — the schema, never the reverse.

**Rationale:** JSON Schema is language-agnostic, self-documenting, and has mature validation libraries in every target language. Keeping schema first prevents the common drift where each consumer's model quietly diverges.

**Consequences:** Any field change must start in the schema. Go struct updates without a corresponding schema update are incomplete. CI catches divergence through roundtrip and schema-validation tests.

---

## ADR-002: Draft-07 schema dialect

**Status:** Accepted

**Context:** JSON Schema has several dialects (Draft-04, Draft-07, Draft 2020-12). The Go validation library ecosystem has uneven support for newer dialects.

**Decision:** Use Draft-07 (`"$schema": "http://json-schema.org/draft-07/schema#"`).

**Rationale:** `github.com/xeipuuv/gojsonschema` — the chosen validator (ADR-004) — has solid Draft-07 support and is widely used in the Go ecosystem. Draft 2020-12 support in Go is less mature and would require switching to a less battle-tested library.

**Consequences:** Some Draft 2020-12 features (e.g. `$defs`, `unevaluatedProperties`) are unavailable. Draft-07 `$ref` semantics (sibling keywords ignored) require care when combining `$ref` with `description`. The `example` keyword (singular) used in the schemas is non-standard in Draft-07 — `gojsonschema` silently ignores unknown keywords, so this is harmless but must not be "corrected" to `examples` (plural array).

---

## ADR-003: Go structs are hand-maintained, not auto-generated

**Status:** Accepted

**Context:** Tools exist to generate Go structs from JSON Schema (e.g. `go-jsonschema`, `omg-gen`). Using them would eliminate manual sync effort.

**Decision:** Go structs in `generated/go/*/types.go` are hand-maintained for v1.0.0.

**Rationale:** Generators for Draft-07 produce mediocre output: they cannot express the `*bool` vs `bool` distinction needed for `omitempty` correctness (the roundtrip trap), they generate verbose intermediate types, and they make it harder to add Go-specific documentation. The schema is small enough that manual maintenance is low-friction.

**Consequences:** Schema and struct must be updated in the same commit (see CLAUDE.md). The `generated/` directory name is intentional: it reserves the import path for future automation without breaking consumers when generators improve.

---

## ADR-004: `gojsonschema` for schema validation

**Status:** Accepted

**Context:** Several Go JSON Schema validation libraries exist: `gojsonschema`, `jsonschema` (santhosh-tekuri), `qri-io/jsonschema`, and others.

**Decision:** Use `github.com/xeipuuv/gojsonschema v1.2.0`.

**Rationale:** It has mature Draft-07 support, no CGo dependency, a stable API, broad adoption in the Go ecosystem, and produces structured error objects (`e.Field()`, `e.Description()`) that are easy to format for human output. It is available offline once vendored.

**Consequences:** The library is in maintenance mode (no new features planned). If Draft 2020-12 becomes necessary, migration to an actively developed library will be needed. For v1.0.0, Draft-07 is sufficient (ADR-002).

---

## ADR-005: Schema embedded via `go:embed`

**Status:** Accepted

**Context:** The validator binary and the Go packages need access to the JSON Schema at runtime for validation. Options: bundle in binary via embed, read from disk at runtime, or hardcode as a string constant.

**Decision:** Embed schema bytes at compile time using `//go:embed` in `schema/schemas.go`.

**Rationale:** Embedding makes the binary fully self-contained — no need to distribute or locate schema files alongside the binary. It also ensures the schema version in the binary always matches the Go package version.

**Implementation note:** Go's `//go:embed` directive does not allow `..` path traversal. The embed directives therefore live in `schema/schemas.go` (within the same directory as the JSON files), and `generated/go/config/schema.go` imports the `schema` package to expose `Schema() []byte`. See `docs/go-conventions.md`.

**Consequences:** Schema files must be updated and re-compiled together. Adding a new schema file requires a corresponding `//go:embed` variable in `schema/schemas.go`.

---

## ADR-006: TypeScript type generation deferred to the builder repo

**Status:** Accepted

**Context:** The React/TypeScript builder (`makakito-builder`) needs TypeScript types for the config shape. Options: generate them from the schema in this repo, or let the builder repo own its own generation.

**Decision:** TypeScript generation is out of scope for v1.0.0. The builder repo generates its own types from this repo's schema.

**Rationale:** Adding a TypeScript toolchain (Node.js, `json-schema-to-typescript`, npm scripts) to this repo for a single consumer violates YAGNI. The schema is already published and language-agnostic; the builder can run `json-schema-to-typescript` against the raw JSON as part of its own build.

**Consequences:** If a second TypeScript consumer emerges, centralising type generation here becomes worthwhile. No `package.json`, no `generated/typescript/`, no `scripts/generate.sh`, no npm steps in CI.

---

## ADR-007: `traffic-profile.schema.json` intentionally empty in v1.0.0

**Status:** Accepted

**Context:** The `makakito-traffic-profiles` repo is still stabilising its shape. The config schema references traffic profiles, but their structure is not yet finalised.

**Decision:** `schema/traffic-profile.schema.json` is a placeholder `{}` with a `$comment` explaining the TODO. `generated/go/trafficprofile/types.go` is an empty package stub.

**Rationale:** Committing a placeholder preserves the directory structure and import path for future work, signals intent clearly, and avoids blocking v1.0.0 on an undecided external shape.

**Consequences:** The traffic-profile schema will be defined in v1.1 once `makakito-traffic-profiles` stabilises. The placeholder must be replaced with a real schema before any consumer depends on it.

---

## ADR-008: Dependencies vendored for offline/VPN-restricted environments

**Status:** Accepted

**Context:** Development and CI may happen in VPN-restricted or air-gapped environments where outbound network access to `proxy.golang.org` is unavailable or unreliable.

**Decision:** Commit `vendor/` to the repository. Run `go mod tidy && go mod vendor` once during initial bootstrap, then commit the result.

**Rationale:** Go ≥1.14 automatically enables `-mod=vendor` when `vendor/` is present, making all `go build`, `go test`, and `go vet` commands work fully offline with no extra flags. CI does not need `go mod download`. `go mod verify` in CI confirms the vendored code matches `go.sum`, providing integrity without network access.

**Consequences:** Dependency updates require running `go get <pkg>@<version>`, `go mod tidy`, `go mod vendor`, and committing the updated `vendor/`. The `vendor/` directory is larger in git history, but this is a worthwhile trade-off for offline reliability.
