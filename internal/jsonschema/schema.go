// internal/jsonschema/schema.go
package jsonschema

import "regexp"

// Schema is the compiled, typed representation of a JSON Schema node.
// Every field corresponds directly to a supported Draft-07 keyword.
// Fields are nil/zero when the keyword was absent from the source schema.
type Schema struct {
	// Metadata — parsed and stored, not evaluated during validation.
	SchemaURI   *string
	ID          *string
	Title       *string
	Description *string
	Default     any
	Example     any // non-standard singular form used in this project; accepted without error

	// Type constraint
	Types []string // e.g. ["string"], ["number","integer"], ["object"]

	// Object keywords
	Properties           map[string]*Schema
	Required             []string
	AdditionalProperties AdditionalProperties

	// Array keywords
	Items       *Schema
	MinItems    *int
	MaxItems    *int
	UniqueItems bool

	// String keywords
	MinLength *int
	MaxLength *int
	Pattern   *regexp.Regexp // compiled at Parse() time; nil when keyword absent

	// Number / integer keywords — stored as float64 (sufficient for all schema ranges here)
	Minimum          *float64
	Maximum          *float64
	ExclusiveMinimum *float64
	ExclusiveMaximum *float64

	// Cross-type keywords
	Enum  []any
	Const *any

	// Composition keywords
	AnyOf []*Schema
	AllOf []*Schema
	OneOf []*Schema

	// References — Ref is resolved during Parse(); rawRef is cleared after resolution.
	Ref  *Schema            // resolved $ref target; nil when "$ref" was absent
	Defs map[string]*Schema // "$defs" entries; nil when absent

	// rawRef stores the unresolved "$ref" string during parsing.
	// It is always empty after Parse() returns successfully.
	rawRef string
}

// AdditionalProperties represents the three possible states of the
// "additionalProperties" keyword: absent (no constraint), boolean, or sub-schema.
type AdditionalProperties struct {
	// Present is true when "additionalProperties" appeared in the schema.
	// When false, additional properties are implicitly allowed regardless of Allowed.
	Present bool
	// Allowed is used when Present==true and Schema==nil.
	// true = additional properties are allowed; false = they are forbidden.
	Allowed bool
	// Schema is non-nil when "additionalProperties" is a sub-schema object.
	// When non-nil, Allowed is ignored and each additional property is validated against Schema.
	Schema *Schema
}
