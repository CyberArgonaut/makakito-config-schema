// Package schema provides the embedded raw JSON Schema bytes for all Makakito
// schema files. This package exists solely as an embed host — Go's //go:embed
// directive does not allow '..' in paths, so schema files must be embedded from
// a Go source file that lives in the same directory.
//
// Consumers should use the typed accessors in the generated Go packages
// (e.g. github.com/CyberArgonaut/makakito-config-schema/generated/go/config)
// rather than this package directly.
package schema

import _ "embed"

// Config is the raw Draft-07 JSON Schema for MakakitoConfig (config.schema.json).
//
//go:embed config.schema.json
var Config []byte

// Scenario is the raw Draft-07 JSON Schema for MakakitoScenario (scenario.schema.json).
//
//go:embed scenario.schema.json
var Scenario []byte

// SteadyState is the raw Draft-07 JSON Schema for MakakitoSteadyState (steady-state.schema.json).
//
//go:embed steady-state.schema.json
var SteadyState []byte

// TrafficProfile is the placeholder schema for traffic profiles (traffic-profile.schema.json).
// The full schema is defined in v1.1 — see DECISIONS.md ADR-007.
//
//go:embed traffic-profile.schema.json
var TrafficProfile []byte
