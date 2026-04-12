# Go Implementation Conventions

This document covers patterns and pitfalls that all agents writing Go code in this repo must follow. Read this before writing or reviewing any file under `generated/go/` or `cmd/`.

---

## Package layout

```
generated/go/
  config/         # types.go  schema.go  validate.go
  scenario/       # types.go
  steadystate/    # types.go
  trafficprofile/ # types.go (empty stub — v1.1)
cmd/
  validator/      # main.go
```

Despite living under `generated/`, these packages are **hand-maintained** (ADR-003). The directory name reserves space for future automation without breaking import paths. Never auto-generate or overwrite them without a plan change.

---

## The `*bool` trap

**Rule**: any optional boolean field that uses `omitempty` in its JSON tag **must** be typed `*bool`, not `bool`.

**Why**: Go's `encoding/json` treats `false` as a zero value for `bool`. With `omitempty`, `json.Marshal` silently drops the field when it is `false`. On remarshal, the field disappears from the JSON, breaking roundtrip equality.

**Affected field**: `Report.IncludeMetricsSnapshot *bool` in `generated/go/config/types.go`.

**Not affected**: optional `int`, `float64`, and `string` fields are safe because their Go zero values (`0`, `0.0`, `""`) are all outside the schema's allowed ranges (ints `minimum >= 1`, floats `minimum >= 0.1`, strings `minLength >= 1`). A dropped zero value means the field was never set.

**Exception to watch for**: `Scenario.Defaults.CooldownSeconds` has `minimum: 0`, meaning `0` is a valid value. The scenario package is not covered by the roundtrip test, so `int` + `omitempty` is acceptable there (0 = no cooldown = same as omitted). If a future test exercises scenario roundtrip, this field must become `*int`.

**Future discipline**: any new optional `bool` field anywhere in any Go package in this repo must be typed `*bool`. Add a note to your PR description calling this out explicitly.

---

## Open maps

These three fields correspond to intentionally-open schema objects. Use the types below exactly:

| Go field | Go type | Schema location |
|---|---|---|
| `Service.Env` | `map[string]string` | `Service.env` |
| `Experiment.Parameters` | `map[string]any` | `Experiment.parameters` |
| `Fault.Parameters` | `map[string]any` | `Fault.parameters` |

Use `map[string]any`, **not** `map[string]interface{}`. The `any` alias is idiomatic in Go 1.18+ and required by the Go 1.26 target.

---

## Modern Go idioms (Go 1.26 target)

| Old | Correct |
|---|---|
| `interface{}` | `any` |
| `ioutil.ReadFile` | `os.ReadFile` |
| `ioutil.WriteFile` | `os.WriteFile` |
| Manual error concatenation | `errors.Join(err1, err2, ...)` |
| `sort.Slice` where `slices` works | `slices.Sort`, `slices.Contains`, etc. |

---

## `go:embed` — why the schema package exists

Go's `//go:embed` directive does **not** allow `..` path elements. A source file in `generated/go/config/` cannot embed `../../../schema/config.schema.json` — this is a compile-time error.

To work around this, the `schema/` directory contains a Go source file (`schema/schemas.go`) that acts as the embed host. Because the Go file lives **in** `schema/`, it can embed files in the same directory with plain names:

```go
// schema/schemas.go
package schema

import _ "embed"

//go:embed config.schema.json
var Config []byte
```

`generated/go/config/schema.go` then imports this package:

```go
package config

import schemapkg "github.com/CyberArgonaut/makakito-config-schema/schema"

func Schema() []byte {
    return schemapkg.Config
}
```

**Rule**: if you add a new schema file to `schema/`, add a corresponding `//go:embed` variable to `schema/schemas.go`. Never attempt to embed schema files from a file outside the `schema/` directory.

---

## `gojsonschema` API patterns

### Basic validation

```go
schemaLoader   := gojsonschema.NewBytesLoader(Schema())   // embedded schema bytes
documentLoader := gojsonschema.NewBytesLoader(data)       // raw JSON to validate

result, err := gojsonschema.Validate(schemaLoader, documentLoader)
if err != nil {
    // engine failure (malformed schema, I/O error) — not a validation error
    return nil, fmt.Errorf("validation engine error: %w", err)
}

if result.Valid() {
    return nil, nil  // no violations
}

for _, e := range result.Errors() {
    // e.Field()       → JSON path, e.g. "playground.type"
    // e.Description() → human-readable message
    fmt.Sprintf("  • %s: %s", e.Field(), e.Description())
}
```

### Error aggregation

Use `errors.Join` when converting a slice of violation strings into a single `error`:

```go
errs := make([]error, len(violations))
for i, v := range violations {
    errs[i] = errors.New(v)
}
return errors.Join(errs...)
```

`errors.Join` was added in Go 1.20 and is part of the standard library — no import needed beyond `"errors"`.

---

## JSON struct tags

- **Required fields**: `json:"fieldName"` (no `omitempty`)
- **Optional fields**: `json:"fieldName,omitempty"`
- **Optional bool fields**: `json:"fieldName,omitempty"` with Go type `*bool`

Go export names use PascalCase; JSON tags use camelCase to match the schema. Example:

```go
type Experiment struct {
    Name            string         `json:"name"`              // required
    Description     string         `json:"description,omitempty"` // optional
    Scenario        string         `json:"scenario"`          // required
    TargetService   string         `json:"targetService,omitempty"`
    DurationSeconds int            `json:"durationSeconds,omitempty"`
    Parameters      map[string]any `json:"parameters,omitempty"`
}
```

---

## Validator CLI conventions

`cmd/validator/main.go` is a thin wrapper — no business logic. It:
1. Reads the file with `os.ReadFile`
2. Calls `config.Validate(data)` directly (no intermediate wrapper package)
3. Prints `config.SchemaVersion` in both the success and failure lines
4. Exits 0 on success, 1 on any failure

Error output goes to `os.Stderr`; success output goes to `os.Stdout`. CI scripts rely on the exit code.

---

## What not to do

- Do not call `go mod tidy` or `go mod vendor` during normal development tasks — those are reserved for the bootstrap task (#10) which runs once, after all source files exist.
- Do not add a wrapper package between `cmd/validator` and the `config` package. The CLI calls `config.Validate` directly.
- Do not use `gofmt` manually — CI enforces `test -z "$(gofmt -l .)"`. Run `gofmt -w .` before committing.
- Do not add indirect dependencies beyond `gojsonschema` (which brings in `gojsonpointer` and `gojsonreference` as its own transitive deps — those are already vendored). Any new direct dependency requires `go get <pkg>@<version>`, `go mod tidy`, `go mod vendor`, and committing the updated `vendor/`.
