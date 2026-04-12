// internal/jsonschema/validate_test.go
package jsonschema_test

import (
	"strings"
	"testing"

	"github.com/CyberArgonaut/makakito-config-schema/internal/jsonschema"
)

// helper: parse schema or fail
func mustParse(t *testing.T, raw string) *jsonschema.Schema {
	t.Helper()
	s, err := jsonschema.Parse([]byte(raw))
	if err != nil {
		t.Fatalf("mustParse: unexpected error: %v", err)
	}
	return s
}

// helper: assert no violations
func assertValid(t *testing.T, schema *jsonschema.Schema, doc string) {
	t.Helper()
	violations, err := jsonschema.Validate(schema, []byte(doc))
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("expected no violations, got:\n%v", violations)
	}
}

// helper: assert at least one violation with given keyword
func assertViolation(t *testing.T, schema *jsonschema.Schema, doc string, keyword string) {
	t.Helper()
	violations, err := jsonschema.Validate(schema, []byte(doc))
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if len(violations) == 0 {
		t.Fatalf("expected violation with keyword %q, got no violations", keyword)
	}
	found := false
	for _, v := range violations {
		if v.Keyword == keyword {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected violation with keyword %q, got: %v", keyword, violations)
	}
}

// --- Task 6: type checking and $ref delegation ---

func TestValidateType(t *testing.T) {
	t.Run("string_match", func(t *testing.T) {
		s := mustParse(t, `{"type": "string"}`)
		assertValid(t, s, `"hello"`)
	})

	t.Run("integer_match", func(t *testing.T) {
		s := mustParse(t, `{"type": "integer"}`)
		assertValid(t, s, `42`)
	})

	t.Run("number_accepts_integer", func(t *testing.T) {
		// JSON Schema: "number" type accepts integer values
		s := mustParse(t, `{"type": "number"}`)
		assertValid(t, s, `42`)
	})

	t.Run("integer_rejects_float", func(t *testing.T) {
		s := mustParse(t, `{"type": "integer"}`)
		assertViolation(t, s, `3.14`, "type")
	})

	t.Run("string_rejects_integer", func(t *testing.T) {
		s := mustParse(t, `{"type": "string"}`)
		assertViolation(t, s, `42`, "type")
	})

	t.Run("object_rejects_array", func(t *testing.T) {
		s := mustParse(t, `{"type": "object"}`)
		assertViolation(t, s, `[1,2,3]`, "type")
	})

	t.Run("null_type", func(t *testing.T) {
		s := mustParse(t, `{"type": "null"}`)
		assertValid(t, s, `null`)
		assertViolation(t, s, `"hello"`, "type")
	})

	t.Run("multiple_types", func(t *testing.T) {
		s := mustParse(t, `{"type": ["string", "null"]}`)
		assertValid(t, s, `"hello"`)
		assertValid(t, s, `null`)
		assertViolation(t, s, `42`, "type")
	})

	t.Run("no_type_constraint_accepts_anything", func(t *testing.T) {
		s := mustParse(t, `{}`)
		assertValid(t, s, `"hello"`)
		assertValid(t, s, `42`)
		assertValid(t, s, `null`)
	})
}

func TestValidateMalformedDocument(t *testing.T) {
	s := mustParse(t, `{"type": "object"}`)
	_, err := jsonschema.Validate(s, []byte(`{invalid json`))
	if err == nil {
		t.Fatal("expected error for malformed document JSON, got nil")
	}
}

func TestValidateRef_Delegation(t *testing.T) {
	schema := `{
		"$defs": {
			"Name": {"type": "string", "minLength": 1}
		},
		"type": "object",
		"properties": {
			"name": {"$ref": "#/$defs/Name"}
		},
		"required": ["name"],
		"additionalProperties": false
	}`
	s := mustParse(t, schema)
	assertValid(t, s, `{"name": "alice"}`)
}

// --- Task 7: object keywords ---

func TestValidateObject(t *testing.T) {
	t.Run("required_present", func(t *testing.T) {
		s := mustParse(t, `{"type":"object","required":["name"]}`)
		assertValid(t, s, `{"name":"alice"}`)
	})

	t.Run("required_missing", func(t *testing.T) {
		s := mustParse(t, `{"type":"object","required":["name","type"]}`)
		violations, _ := jsonschema.Validate(s, []byte(`{"name":"alice"}`))
		found := false
		for _, v := range violations {
			if v.Keyword == "required" && v.Field == "type" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected required violation for field 'type', got: %v", violations)
		}
	})

	t.Run("additional_properties_false_valid", func(t *testing.T) {
		s := mustParse(t, `{"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":false}`)
		assertValid(t, s, `{"name":"alice"}`)
	})

	t.Run("additional_properties_false_extra_key", func(t *testing.T) {
		s := mustParse(t, `{"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":false}`)
		violations, _ := jsonschema.Validate(s, []byte(`{"name":"alice","extra":"value"}`))
		found := false
		for _, v := range violations {
			if v.Keyword == "additionalProperties" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected additionalProperties violation for extra key, got: %v", violations)
		}
	})

	t.Run("additional_properties_schema_valid", func(t *testing.T) {
		s := mustParse(t, `{"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":{"type":"string"}}`)
		assertValid(t, s, `{"name":"alice","extra":"value"}`)
	})

	t.Run("additional_properties_schema_invalid", func(t *testing.T) {
		s := mustParse(t, `{"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":{"type":"string"}}`)
		assertViolation(t, s, `{"name":"alice","extra":42}`, "type")
	})

	t.Run("nested_property_violation_path", func(t *testing.T) {
		s := mustParse(t, `{
			"type":"object",
			"properties":{"port":{"type":"integer","minimum":1,"maximum":65535}},
			"required":["port"]
		}`)
		violations, _ := jsonschema.Validate(s, []byte(`{"port": 0}`))
		found := false
		for _, v := range violations {
			if v.Field == "port" && v.Keyword == "minimum" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected minimum violation at field 'port', got: %v", violations)
		}
	})
}

// --- Task 8: array keywords ---

func TestValidateArray(t *testing.T) {
	t.Run("minItems_pass", func(t *testing.T) {
		s := mustParse(t, `{"type":"array","minItems":1}`)
		assertValid(t, s, `[1]`)
	})

	t.Run("minItems_fail", func(t *testing.T) {
		s := mustParse(t, `{"type":"array","minItems":1}`)
		violations, _ := jsonschema.Validate(s, []byte(`[]`))
		found := false
		for _, v := range violations {
			if v.Keyword == "minItems" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected minItems violation, got: %v", violations)
		}
	})

	t.Run("maxItems_pass", func(t *testing.T) {
		s := mustParse(t, `{"type":"array","maxItems":2}`)
		assertValid(t, s, `[1,2]`)
	})

	t.Run("maxItems_fail", func(t *testing.T) {
		s := mustParse(t, `{"type":"array","maxItems":2}`)
		assertViolation(t, s, `[1,2,3]`, "maxItems")
	})

	t.Run("items_valid", func(t *testing.T) {
		s := mustParse(t, `{"type":"array","items":{"type":"string"}}`)
		assertValid(t, s, `["a","b","c"]`)
	})

	t.Run("items_invalid_element", func(t *testing.T) {
		s := mustParse(t, `{"type":"array","items":{"type":"string"}}`)
		violations, _ := jsonschema.Validate(s, []byte(`["a", 42, "c"]`))
		found := false
		for _, v := range violations {
			if v.Keyword == "type" && v.Field == "[1]" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected type violation at [1], got: %v", violations)
		}
	})

	t.Run("uniqueItems_pass", func(t *testing.T) {
		s := mustParse(t, `{"type":"array","uniqueItems":true}`)
		assertValid(t, s, `["a","b","c"]`)
	})

	t.Run("uniqueItems_fail", func(t *testing.T) {
		s := mustParse(t, `{"type":"array","uniqueItems":true}`)
		violations, _ := jsonschema.Validate(s, []byte(`["a","b","a"]`))
		found := false
		for _, v := range violations {
			if v.Keyword == "uniqueItems" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected uniqueItems violation, got: %v", violations)
		}
	})
}

// --- Task 9: string and number keywords ---

func TestValidateString(t *testing.T) {
	t.Run("minLength_pass", func(t *testing.T) {
		s := mustParse(t, `{"type":"string","minLength":3}`)
		assertValid(t, s, `"abc"`)
	})

	t.Run("minLength_fail", func(t *testing.T) {
		s := mustParse(t, `{"type":"string","minLength":3}`)
		assertViolation(t, s, `"ab"`, "minLength")
	})

	t.Run("maxLength_pass", func(t *testing.T) {
		s := mustParse(t, `{"type":"string","maxLength":5}`)
		assertValid(t, s, `"hello"`)
	})

	t.Run("maxLength_fail", func(t *testing.T) {
		s := mustParse(t, `{"type":"string","maxLength":5}`)
		assertViolation(t, s, `"toolong"`, "maxLength")
	})

	t.Run("minLength_unicode", func(t *testing.T) {
		// "日本語" is 3 Unicode code points — must pass minLength:3
		s := mustParse(t, `{"type":"string","minLength":3}`)
		assertValid(t, s, `"日本語"`)
	})

	t.Run("pattern_match", func(t *testing.T) {
		s := mustParse(t, `{"type":"string","pattern":"^[a-z]+$"}`)
		assertValid(t, s, `"hello"`)
	})

	t.Run("pattern_no_match", func(t *testing.T) {
		s := mustParse(t, `{"type":"string","pattern":"^[a-z]+$"}`)
		assertViolation(t, s, `"Hello123"`, "pattern")
	})
}

func TestValidateNumber(t *testing.T) {
	t.Run("minimum_pass_exact", func(t *testing.T) {
		// Boundary: value == minimum must pass
		s := mustParse(t, `{"type":"integer","minimum":1}`)
		assertValid(t, s, `1`)
	})

	t.Run("minimum_fail", func(t *testing.T) {
		s := mustParse(t, `{"type":"integer","minimum":1}`)
		assertViolation(t, s, `0`, "minimum")
	})

	t.Run("maximum_pass_exact", func(t *testing.T) {
		// Boundary: value == maximum must pass
		s := mustParse(t, `{"type":"integer","maximum":65535}`)
		assertValid(t, s, `65535`)
	})

	t.Run("maximum_fail", func(t *testing.T) {
		s := mustParse(t, `{"type":"integer","maximum":65535}`)
		assertViolation(t, s, `65536`, "maximum")
	})

	t.Run("exclusiveMinimum_pass", func(t *testing.T) {
		s := mustParse(t, `{"type":"number","exclusiveMinimum":0}`)
		assertValid(t, s, `0.1`)
	})

	t.Run("exclusiveMinimum_fail_equal", func(t *testing.T) {
		// Value equal to exclusiveMinimum must fail
		s := mustParse(t, `{"type":"number","exclusiveMinimum":0}`)
		assertViolation(t, s, `0`, "exclusiveMinimum")
	})

	t.Run("exclusiveMaximum_pass", func(t *testing.T) {
		s := mustParse(t, `{"type":"number","exclusiveMaximum":1}`)
		assertValid(t, s, `0.9`)
	})

	t.Run("exclusiveMaximum_fail_equal", func(t *testing.T) {
		s := mustParse(t, `{"type":"number","exclusiveMaximum":1}`)
		assertViolation(t, s, `1`, "exclusiveMaximum")
	})
}

// --- Task 10: enum, const, and composition ---

func TestValidateEnum(t *testing.T) {
	t.Run("valid_value", func(t *testing.T) {
		s := mustParse(t, `{"type":"string","enum":["json","html","both"]}`)
		assertValid(t, s, `"json"`)
	})

	t.Run("invalid_value", func(t *testing.T) {
		s := mustParse(t, `{"type":"string","enum":["json","html","both"]}`)
		violations, _ := jsonschema.Validate(s, []byte(`"xml"`))
		found := false
		for _, v := range violations {
			if v.Keyword == "enum" {
				found = true
				// Message should list allowed values
				if !strings.Contains(v.Message, "json") {
					t.Errorf("enum violation message should list allowed values, got: %s", v.Message)
				}
			}
		}
		if !found {
			t.Errorf("expected enum violation, got: %v", violations)
		}
	})

	t.Run("integer_enum", func(t *testing.T) {
		s := mustParse(t, `{"type":"integer","enum":[1,2,3]}`)
		assertValid(t, s, `2`)
		assertViolation(t, s, `4`, "enum")
	})
}

func TestValidateConst(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		s := mustParse(t, `{"const":"fixed"}`)
		assertValid(t, s, `"fixed"`)
	})

	t.Run("invalid", func(t *testing.T) {
		s := mustParse(t, `{"const":"fixed"}`)
		assertViolation(t, s, `"other"`, "const")
	})

	t.Run("const_integer", func(t *testing.T) {
		s := mustParse(t, `{"const":42}`)
		assertValid(t, s, `42`)
		assertViolation(t, s, `43`, "const")
	})
}

func TestValidateComposition(t *testing.T) {
	t.Run("anyOf_one_passes", func(t *testing.T) {
		s := mustParse(t, `{"anyOf":[{"type":"string"},{"type":"integer"}]}`)
		assertValid(t, s, `"hello"`)
		assertValid(t, s, `42`)
	})

	t.Run("anyOf_all_fail", func(t *testing.T) {
		s := mustParse(t, `{"anyOf":[{"type":"string"},{"type":"integer"}]}`)
		violations, _ := jsonschema.Validate(s, []byte(`[1,2,3]`))
		found := false
		for _, v := range violations {
			if v.Keyword == "anyOf" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected anyOf violation, got: %v", violations)
		}
	})

	t.Run("allOf_all_pass", func(t *testing.T) {
		s := mustParse(t, `{"allOf":[{"type":"string"},{"minLength":2}]}`)
		assertValid(t, s, `"hello"`)
	})

	t.Run("allOf_one_fails", func(t *testing.T) {
		s := mustParse(t, `{"allOf":[{"type":"string"},{"minLength":10}]}`)
		assertViolation(t, s, `"hi"`, "minLength")
	})

	t.Run("oneOf_exactly_one_passes", func(t *testing.T) {
		s := mustParse(t, `{"oneOf":[{"type":"string"},{"type":"integer"}]}`)
		assertValid(t, s, `"hello"`)
	})

	t.Run("oneOf_zero_pass", func(t *testing.T) {
		s := mustParse(t, `{"oneOf":[{"type":"string"},{"type":"integer"}]}`)
		assertViolation(t, s, `[1,2,3]`, "oneOf")
	})

	t.Run("oneOf_two_pass", func(t *testing.T) {
		// Both string and minLength:0 match "hello"; oneOf must fail.
		s := mustParse(t, `{"oneOf":[{"type":"string"},{"minLength":0}]}`)
		assertViolation(t, s, `"hello"`, "oneOf")
	})
}

// --- Task 11: nested path accuracy and $ref delegation full cycle ---

func TestValidateNestedPath(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"services": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"port": {"type": "integer", "minimum": 1}
					},
					"required": ["port"]
				}
			}
		},
		"required": ["services"]
	}`
	s := mustParse(t, schema)

	t.Run("valid", func(t *testing.T) {
		assertValid(t, s, `{"services":[{"port":8080}]}`)
	})

	t.Run("nested_minimum_violation_path", func(t *testing.T) {
		violations, _ := jsonschema.Validate(s, []byte(`{"services":[{"port":0}]}`))
		found := false
		for _, v := range violations {
			if v.Field == "services[0].port" && v.Keyword == "minimum" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected violation at path 'services[0].port' with keyword 'minimum', got: %v", violations)
		}
	})

	t.Run("nested_required_violation_path", func(t *testing.T) {
		violations, _ := jsonschema.Validate(s, []byte(`{"services":[{}]}`))
		found := false
		for _, v := range violations {
			if v.Field == "services[0].port" && v.Keyword == "required" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected violation at path 'services[0].port' with keyword 'required', got: %v", violations)
		}
	})
}

func TestValidateRefDelegation_FullCycle(t *testing.T) {
	// $ref delegates entirely: validate via the resolved target schema.
	schema := `{
		"$defs": {
			"Port": {"type": "integer", "minimum": 1, "maximum": 65535}
		},
		"type": "object",
		"properties": {
			"port": {"$ref": "#/$defs/Port"}
		},
		"required": ["port"],
		"additionalProperties": false
	}`
	s := mustParse(t, schema)

	t.Run("valid_port", func(t *testing.T) {
		assertValid(t, s, `{"port": 8080}`)
	})

	t.Run("port_below_minimum", func(t *testing.T) {
		violations, _ := jsonschema.Validate(s, []byte(`{"port": 0}`))
		found := false
		for _, v := range violations {
			if v.Field == "port" && v.Keyword == "minimum" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected minimum violation for port, got: %v", violations)
		}
	})

	t.Run("port_above_maximum", func(t *testing.T) {
		violations, _ := jsonschema.Validate(s, []byte(`{"port": 99999}`))
		found := false
		for _, v := range violations {
			if v.Field == "port" && v.Keyword == "maximum" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected maximum violation for port, got: %v", violations)
		}
	})
}
