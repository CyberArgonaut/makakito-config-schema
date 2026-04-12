# makakito-config-schema

The shared contract repository for the **Makakito** chaos-engineering learning platform. It defines the canonical shape of `config.json` — the file produced by the builder, consumed by the runner, validated by the CLI, and used to hydrate playground Docker Compose environments.

## Role in the Makakito ecosystem

| Consumer | How it uses this repo |
|---|---|
| `makakito-cli` | Imports the generated Go package to validate `config.json` |
| `makakito-runner` | Imports the generated Go package to parse and act on `config.json` |
| `makakito-builder` (React/TS) | Generates its own TypeScript types from this repo's JSON Schema |
| `makakito-playgrounds` | References schema version in `COMPATIBILITY.md` |
| `makakito-chaos-scenarios` | Scenario YAML shape defined in `schema/scenario.schema.json` |
| `makakito-traffic-profiles` | Traffic profile reference shape defined in `schema/traffic-profile.schema.json` (v1.1+) |

## Design principles

- **JSON Schema is the single source of truth.** Go structs and (eventually) TypeScript types are derived from the schema, never the reverse.
- **Draft-07** is the schema dialect (chosen for Go ecosystem compatibility via `gojsonschema`).
- **Strict by default.** Every closed object has `additionalProperties: false`. The handful of intentionally-open maps (`service.env`, `experiment.parameters`, `fault.parameters`) are documented.
- **Go structs are hand-maintained**, not generated. See `DECISIONS.md` ADR-003.
- **Offline-first.** Eventual dependencies are vendored (`vendor/` committed). Builds and CI require no network after the initial bootstrap. Friendly to VPN-restricted development environments.
- **Semantic versioning.** Breaking changes bump the major version. Consumers pin against major versions via `COMPATIBILITY.md`.

## Repository layout

```
schema/                         JSON Schema source of truth
  config.schema.json
  scenario.schema.json
  steady-state.schema.json
  traffic-profile.schema.json   (stub for v1.1)
  schemas.go                    go:embed host for all schema files
generated/go/                   Hand-maintained Go packages
  config/                       types, validate, embedded schema
  scenario/
  steadystate/
  trafficprofile/               (empty stub for v1.1)
cmd/validator/                  Standalone CLI binary
examples/                       Valid + invalid example configs
tests/                          Schema validation + roundtrip tests
vendor/                         Vendored deps (offline-first)
docs/                           Agent and developer reference
  schema-reference.md
  go-conventions.md
```

## Validate a config from the command line

```bash
go build -o bin/validator ./cmd/validator
./bin/validator path/to/config.json
```

On success:
```
path/to/config.json is valid (schema v1.0.0)
```

On failure, schema violations are listed to stderr and the process exits 1:
```
path/to/config.json is invalid (schema v1.0.0):
  • experiment.scenario: experiment.scenario is required
```

## Consume in Go

```bash
go get github.com/CyberArgonaut/makakito-config-schema/generated/go/config@v1
```

**Validate only:**

```go
import "github.com/CyberArgonaut/makakito-config-schema/generated/go/config"

violations, err := config.Validate(data) // data is []byte
if err != nil {
    // engine failure (malformed schema, I/O error)
}
if len(violations) > 0 {
    // violations is []string of human-readable messages
}
```

**Validate and parse:**

```go
cfg, err := config.Parse(data)
if err != nil {
    // err joins all schema violations into a single error message
}
// cfg is *config.MakakitoConfig
```

**Access the embedded schema bytes** (e.g. to forward to another validator):

```go
raw := config.Schema() // []byte, embedded at compile time
```

## Schema versioning

- `config.SchemaVersion` is a Go constant (`"1.0.0"`) that must always match the version in `schema/config.schema.json`'s `$id` field.
- Breaking changes (removed or renamed required fields) bump the major version.
- Additive changes (new optional fields) bump the minor version.
- Consumers should pin to a major version: `@v1`, `@v2`, etc.
- See `COMPATIBILITY.md` for the consumer-repo compatibility table.

## Offline / VPN-friendly builds

All dependencies are vendored. After the initial bootstrap (`go mod tidy && go mod vendor`), no network access is required:

```bash
go build ./...   # fully offline
go test ./...    # fully offline
go vet ./...     # fully offline
```

## Roadmap

**v1.0.0 (current)** — initial scaffold: `config`, `scenario`, and `steady-state` schemas; Go packages; standalone validator; examples; tests; CI.

**v1.1** — `schema/traffic-profile.schema.json` filled in once `makakito-traffic-profiles` stabilises its shape.

## Further reading

- `COMPATIBILITY.md` — schema version → consumer repo compatibility table
- `docs/schema-reference.md` — complete field-by-field schema reference
- `docs/go-conventions.md` — Go implementation patterns and pitfalls
