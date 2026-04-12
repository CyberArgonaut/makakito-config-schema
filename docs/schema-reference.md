# Schema Reference

Complete field-by-field reference for all JSON Schema files in `schema/`. The schemas are the canonical source of truth — Go structs and any future TypeScript types are derived from them.

All schemas use **Draft-07** (`"$schema": "http://json-schema.org/draft-07/schema#"`). The `example` keyword (singular) is used as a documentation hint; it is non-standard in Draft-07 but harmless — `gojsonschema` silently ignores unknown keywords. Do **not** rewrite it to `examples` (plural array).

---

## config.schema.json

**`$id`**: `https://makakito.dev/schemas/config/v1`  
**Root type**: `MakakitoConfig`  
**Required top-level fields**: `schemaVersion`, `playground`, `services`, `experiment`

### Top-level fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `schemaVersion` | `string` enum `["1.0.0"]` | yes | Must match `config.SchemaVersion` constant |
| `playground` | `Playground` object | yes | See below |
| `services` | `Service[]` minItems 1 | yes | See below |
| `experiment` | `Experiment` object | yes | See below |
| `report` | `Report` object | no | Omit to suppress report output |
| `traffic` | `Traffic` object | no | Inline traffic overrides; full profile support is v1.1+ |

### `Playground`

`additionalProperties: false`

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string minLength 1 | yes | Human-readable identifier |
| `type` | string enum | yes | `"microservices"`, `"queue"`, `"external-api"` |
| `description` | string | no | |

### `Service`

`additionalProperties: false`

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string minLength 1 | yes | Unique within `services[]` |
| `image` | string minLength 1 | yes | Docker image reference |
| `type` | string enum | yes | `"web"`, `"worker"`, `"database"`, `"cache"`, `"queue"`, `"gateway"`, `"downstream"` |
| `port` | integer 1–65535 | no | Primary exposed port |
| `env` | object | no | **Intentionally open**: `additionalProperties: {type: string}`. Arbitrary env vars. |
| `dependsOn` | string[] | no | Names of services that must be healthy first |
| `replicas` | integer minimum 1 | no | Container replica count; runner defaults to 1 |
| `resources` | `Resources` object | no | CPU/memory limits |

### `Resources`

`additionalProperties: false`

| Field | Type | Required | Notes |
|---|---|---|---|
| `cpu` | string minLength 1 | no | Docker format, e.g. `"0.5"` |
| `memory` | string minLength 1 | no | Docker format, e.g. `"256m"` |

### `Experiment`

`additionalProperties: false`

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string minLength 1 | yes | |
| `description` | string | no | |
| `scenario` | string minLength 1 | yes | Path to scenario YAML. **Omitting this field is the canonical way to produce an invalid config** (`examples/invalid-missing-fields.json`). |
| `targetService` | string minLength 1 | no | Must match a `services[].name` |
| `durationSeconds` | integer minimum 1 | no | Overrides scenario default |
| `parameters` | object | no | **Intentionally open**: `additionalProperties: true`. Scenario-specific overrides. |

### `Report`

`additionalProperties: false`

| Field | Type | Required | Notes |
|---|---|---|---|
| `outputFormat` | string enum | yes | `"json"`, `"html"`, `"both"` |
| `includeMetricsSnapshot` | boolean | no | **Go struct uses `*bool`** — see [go-conventions.md](go-conventions.md#the-bool-trap) |
| `outputPath` | string minLength 1 | no | Defaults to `./reports` in the runner |

### `Traffic`

`additionalProperties: false`

| Field | Type | Required | Notes |
|---|---|---|---|
| `requestsPerSecond` | number minimum 0.1 | no | |
| `connections` | integer minimum 1 | no | |
| `targetService` | string minLength 1 | no | Defaults to `experiment.targetService` |
| `durationSeconds` | integer minimum 1 | no | Defaults to `experiment.durationSeconds` |

---

## scenario.schema.json

**`$id`**: `https://makakito.dev/schemas/scenario/v1`  
**Root type**: `MakakitoScenario`  
**Required fields**: `name`, `faults`

### Top-level fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string minLength 1 | yes | |
| `description` | string | no | |
| `defaults` | `Defaults` object | no | Applied to faults that omit their own timing |
| `faults` | `Fault[]` minItems 1 | yes | Applied sequentially |

### `Defaults`

`additionalProperties: false`

| Field | Type | Required | Notes |
|---|---|---|---|
| `durationSeconds` | integer minimum 1 | no | |
| `cooldownSeconds` | integer minimum 0 | no | Note: minimum is 0 (zero means no cooldown). Go struct uses plain `int` — omitempty drops `0`, which is semantically correct (no cooldown = omitted). |

### `Fault`

`additionalProperties: false`

| Field | Type | Required | Notes |
|---|---|---|---|
| `type` | string minLength 1 | yes | Fault driver ID (e.g. `"cpu-stress"`, `"http-fault"`, `"network-latency"`, `"queue-consumer-lag"`) |
| `description` | string | no | |
| `durationSeconds` | integer minimum 1 | no | Overrides `defaults.durationSeconds` |
| `parameters` | object | no | **Intentionally open**: `additionalProperties: true`. Driver-specific config. |

---

## steady-state.schema.json

**`$id`**: `https://makakito.dev/schemas/steady-state/v1`  
**Root type**: `MakakitoSteadyState`  
**Required fields**: `name`, `thresholds`

### Top-level fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string minLength 1 | yes | |
| `description` | string | no | |
| `thresholds` | `Threshold[]` minItems 1 | yes | All must pass for the hypothesis to hold |

### `Threshold`

`additionalProperties: false`

| Field | Type | Required | Notes |
|---|---|---|---|
| `metric` | string minLength 1 | yes | Metric path as understood by the runner (e.g. `"http.p99_latency_ms"`) |
| `operator` | string enum | yes | `"lt"` (<), `"lte"` (≤), `"gt"` (>), `"gte"` (≥), `"eq"` (=) |
| `value` | number | yes | Threshold value |
| `description` | string | no | |

---

## traffic-profile.schema.json

**Status**: placeholder — v1.1+.  
The file contains only `{"$comment": "TODO: ..."}`. Do not add fields without also updating `generated/go/trafficprofile/types.go` and `DECISIONS.md`. See ADR-007.

---

## Strictness rules summary

Every closed object has `additionalProperties: false`. The following three are **intentionally open** — do not change them:

| Object | Schema location | Reason |
|---|---|---|
| `Service.env` | `config.schema.json` → `Service.env` | Arbitrary string env vars |
| `Experiment.parameters` | `config.schema.json` → `Experiment.parameters` | Scenario-specific overrides |
| `Fault.parameters` | `scenario.schema.json` → `Fault.parameters` | Driver-specific config |

---

## Producing invalid configs (for tests)

| Example file | What is broken |
|---|---|
| `examples/invalid-missing-fields.json` | `experiment.scenario` is omitted (required field) |
| `examples/invalid-bad-values.json` | `services[0].type = "not-a-type"` and `playground.type = "unknown"` (enum violations) |
