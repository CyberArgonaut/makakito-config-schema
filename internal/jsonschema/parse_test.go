// internal/jsonschema/parse_test.go
package jsonschema_test

import (
	"strings"
	"testing"

	"github.com/CyberArgonaut/makakito-config-schema/internal/jsonschema"
)

func TestParseUnsupportedKeyword_Root(t *testing.T) {
	_, err := jsonschema.Parse([]byte(`{"if": {"type": "string"}}`))
	if err == nil {
		t.Fatal("expected error for unsupported keyword 'if', got nil")
	}
	if !strings.Contains(err.Error(), `"if"`) {
		t.Errorf("error should mention keyword 'if', got: %v", err)
	}
}

func TestParseUnsupportedKeyword_InProperties(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string", "then": {"type": "string"}}
		}
	}`
	_, err := jsonschema.Parse([]byte(schema))
	if err == nil {
		t.Fatal("expected error for unsupported keyword 'then' inside properties.name")
	}
	if !strings.Contains(err.Error(), `"then"`) {
		t.Errorf("error should mention 'then', got: %v", err)
	}
}

func TestParseUnsupportedKeyword_InDefs(t *testing.T) {
	schema := `{
		"$defs": {
			"Foo": {"type": "string", "not": {"type": "integer"}}
		}
	}`
	_, err := jsonschema.Parse([]byte(schema))
	if err == nil {
		t.Fatal("expected error for unsupported keyword 'not' inside $defs.Foo")
	}
	if !strings.Contains(err.Error(), `"not"`) {
		t.Errorf("error should mention 'not', got: %v", err)
	}
}

func TestParseExampleKeyword_Accepted(t *testing.T) {
	// "example" (singular) is non-standard but used throughout this project.
	// It must be accepted without error.
	schema := `{"type": "string", "example": "hello"}`
	_, err := jsonschema.Parse([]byte(schema))
	if err != nil {
		t.Fatalf("unexpected error for 'example' keyword: %v", err)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := jsonschema.Parse([]byte(`{invalid json`))
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestParseMinimalValidSchema(t *testing.T) {
	schema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "Test",
		"type": "object",
		"properties": {"name": {"type": "string"}},
		"required": ["name"],
		"additionalProperties": false
	}`
	s, err := jsonschema.Parse([]byte(schema))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil Schema")
	}
}

func TestParseScalarKeywords(t *testing.T) {
	t.Run("type_string", func(t *testing.T) {
		s, err := jsonschema.Parse([]byte(`{"type": "string"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(s.Types) != 1 || s.Types[0] != "string" {
			t.Errorf("expected Types=[string], got %v", s.Types)
		}
	})

	t.Run("type_array", func(t *testing.T) {
		s, err := jsonschema.Parse([]byte(`{"type": ["string", "null"]}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(s.Types) != 2 || s.Types[0] != "string" || s.Types[1] != "null" {
			t.Errorf("expected Types=[string null], got %v", s.Types)
		}
	})

	t.Run("minimum_and_maximum", func(t *testing.T) {
		s, err := jsonschema.Parse([]byte(`{"type": "integer", "minimum": 1, "maximum": 65535}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Minimum == nil || *s.Minimum != 1.0 {
			t.Errorf("expected Minimum=1.0, got %v", s.Minimum)
		}
		if s.Maximum == nil || *s.Maximum != 65535.0 {
			t.Errorf("expected Maximum=65535.0, got %v", s.Maximum)
		}
	})

	t.Run("exclusive_bounds", func(t *testing.T) {
		s, err := jsonschema.Parse([]byte(`{"type": "number", "exclusiveMinimum": 0.0, "exclusiveMaximum": 1.0}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.ExclusiveMinimum == nil || *s.ExclusiveMinimum != 0.0 {
			t.Errorf("expected ExclusiveMinimum=0.0, got %v", s.ExclusiveMinimum)
		}
		if s.ExclusiveMaximum == nil || *s.ExclusiveMaximum != 1.0 {
			t.Errorf("expected ExclusiveMaximum=1.0, got %v", s.ExclusiveMaximum)
		}
	})

	t.Run("minItems_and_maxItems", func(t *testing.T) {
		s, err := jsonschema.Parse([]byte(`{"type": "array", "minItems": 1, "maxItems": 10}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.MinItems == nil || *s.MinItems != 1 {
			t.Errorf("expected MinItems=1, got %v", s.MinItems)
		}
		if s.MaxItems == nil || *s.MaxItems != 10 {
			t.Errorf("expected MaxItems=10, got %v", s.MaxItems)
		}
	})

	t.Run("minLength_and_maxLength", func(t *testing.T) {
		s, err := jsonschema.Parse([]byte(`{"type": "string", "minLength": 2, "maxLength": 64}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.MinLength == nil || *s.MinLength != 2 {
			t.Errorf("expected MinLength=2, got %v", s.MinLength)
		}
		if s.MaxLength == nil || *s.MaxLength != 64 {
			t.Errorf("expected MaxLength=64, got %v", s.MaxLength)
		}
	})

	t.Run("pattern_valid", func(t *testing.T) {
		s, err := jsonschema.Parse([]byte(`{"type": "string", "pattern": "^[a-z]+$"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Pattern == nil {
			t.Fatal("expected non-nil Pattern")
		}
		if s.Pattern.String() != "^[a-z]+$" {
			t.Errorf("expected pattern '^[a-z]+$', got %q", s.Pattern.String())
		}
	})

	t.Run("pattern_invalid_regex", func(t *testing.T) {
		_, err := jsonschema.Parse([]byte(`{"type": "string", "pattern": "[invalid"}`))
		if err == nil {
			t.Fatal("expected error for invalid regex pattern")
		}
	})

	t.Run("required", func(t *testing.T) {
		s, err := jsonschema.Parse([]byte(`{"type": "object", "required": ["name", "type"]}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(s.Required) != 2 || s.Required[0] != "name" || s.Required[1] != "type" {
			t.Errorf("expected Required=[name type], got %v", s.Required)
		}
	})

	t.Run("uniqueItems", func(t *testing.T) {
		s, err := jsonschema.Parse([]byte(`{"type": "array", "uniqueItems": true}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !s.UniqueItems {
			t.Error("expected UniqueItems=true")
		}
	})

	t.Run("enum", func(t *testing.T) {
		s, err := jsonschema.Parse([]byte(`{"type": "string", "enum": ["json", "html", "both"]}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(s.Enum) != 3 {
			t.Errorf("expected 3 enum values, got %d", len(s.Enum))
		}
	})

	t.Run("const", func(t *testing.T) {
		s, err := jsonschema.Parse([]byte(`{"const": "fixed"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Const == nil {
			t.Fatal("expected non-nil Const")
		}
	})

	t.Run("metadata_fields", func(t *testing.T) {
		s, err := jsonschema.Parse([]byte(`{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"$id": "https://example.com/test",
			"title": "My Schema",
			"description": "A test schema",
			"default": "hello",
			"example": "world"
		}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Title == nil || *s.Title != "My Schema" {
			t.Errorf("expected Title='My Schema', got %v", s.Title)
		}
		if s.Description == nil || *s.Description != "A test schema" {
			t.Errorf("expected Description set, got %v", s.Description)
		}
	})

	t.Run("absent_fields_are_nil", func(t *testing.T) {
		s, err := jsonschema.Parse([]byte(`{"type": "string"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Minimum != nil {
			t.Errorf("expected Minimum=nil for absent keyword, got %v", s.Minimum)
		}
		if s.Pattern != nil {
			t.Errorf("expected Pattern=nil for absent keyword")
		}
		if len(s.Required) != 0 {
			t.Errorf("expected empty Required for absent keyword, got %v", s.Required)
		}
	})
}

func TestParseNestedSchemas(t *testing.T) {
	t.Run("properties", func(t *testing.T) {
		schema := `{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"port": {"type": "integer", "minimum": 1, "maximum": 65535}
			}
		}`
		s, err := jsonschema.Parse([]byte(schema))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(s.Properties) != 2 {
			t.Fatalf("expected 2 properties, got %d", len(s.Properties))
		}
		port := s.Properties["port"]
		if port == nil {
			t.Fatal("expected port property to be non-nil")
		}
		if port.Minimum == nil || *port.Minimum != 1.0 {
			t.Errorf("expected port.Minimum=1.0, got %v", port.Minimum)
		}
	})

	t.Run("items", func(t *testing.T) {
		schema := `{"type": "array", "items": {"type": "string"}}`
		s, err := jsonschema.Parse([]byte(schema))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Items == nil {
			t.Fatal("expected non-nil Items")
		}
		if len(s.Items.Types) != 1 || s.Items.Types[0] != "string" {
			t.Errorf("expected Items.Types=[string], got %v", s.Items.Types)
		}
	})

	t.Run("items_array_form_rejected", func(t *testing.T) {
		// Tuple form of items is unsupported and must fail.
		schema := `{"type": "array", "items": [{"type": "string"}, {"type": "integer"}]}`
		_, err := jsonschema.Parse([]byte(schema))
		if err == nil {
			t.Fatal("expected error for tuple form of items, got nil")
		}
	})

	t.Run("additionalProperties_false", func(t *testing.T) {
		schema := `{"type": "object", "additionalProperties": false}`
		s, err := jsonschema.Parse([]byte(schema))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		ap := s.AdditionalProperties
		if !ap.Present {
			t.Error("expected AdditionalProperties.Present=true")
		}
		if ap.Allowed {
			t.Error("expected AdditionalProperties.Allowed=false")
		}
		if ap.Schema != nil {
			t.Error("expected AdditionalProperties.Schema=nil")
		}
	})

	t.Run("additionalProperties_schema", func(t *testing.T) {
		schema := `{"type": "object", "additionalProperties": {"type": "string"}}`
		s, err := jsonschema.Parse([]byte(schema))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		ap := s.AdditionalProperties
		if !ap.Present {
			t.Error("expected AdditionalProperties.Present=true")
		}
		if ap.Schema == nil {
			t.Fatal("expected AdditionalProperties.Schema to be non-nil")
		}
		if len(ap.Schema.Types) != 1 || ap.Schema.Types[0] != "string" {
			t.Errorf("expected additionalProperties schema type=string, got %v", ap.Schema.Types)
		}
	})

	t.Run("additionalProperties_absent", func(t *testing.T) {
		schema := `{"type": "object"}`
		s, err := jsonschema.Parse([]byte(schema))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.AdditionalProperties.Present {
			t.Error("expected AdditionalProperties.Present=false when keyword is absent")
		}
	})

	t.Run("anyOf", func(t *testing.T) {
		schema := `{"anyOf": [{"type": "string"}, {"type": "integer"}]}`
		s, err := jsonschema.Parse([]byte(schema))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(s.AnyOf) != 2 {
			t.Fatalf("expected 2 anyOf branches, got %d", len(s.AnyOf))
		}
		if len(s.AnyOf[0].Types) != 1 || s.AnyOf[0].Types[0] != "string" {
			t.Errorf("expected first anyOf branch type=string, got %v", s.AnyOf[0].Types)
		}
	})

	t.Run("allOf", func(t *testing.T) {
		schema := `{"allOf": [{"minLength": 1}, {"maxLength": 64}]}`
		s, err := jsonschema.Parse([]byte(schema))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(s.AllOf) != 2 {
			t.Fatalf("expected 2 allOf branches, got %d", len(s.AllOf))
		}
	})

	t.Run("oneOf", func(t *testing.T) {
		schema := `{"oneOf": [{"type": "string"}, {"type": "null"}]}`
		s, err := jsonschema.Parse([]byte(schema))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(s.OneOf) != 2 {
			t.Fatalf("expected 2 oneOf branches, got %d", len(s.OneOf))
		}
	})
}

func TestParseDefsAndRefs(t *testing.T) {
	t.Run("defs_populated", func(t *testing.T) {
		schema := `{
			"$defs": {
				"Port": {"type": "integer", "minimum": 1, "maximum": 65535}
			},
			"type": "object",
			"properties": {
				"port": {"$ref": "#/$defs/Port"}
			}
		}`
		s, err := jsonschema.Parse([]byte(schema))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Defs == nil {
			t.Fatal("expected non-nil Defs")
		}
		port, ok := s.Defs["Port"]
		if !ok || port == nil {
			t.Fatal("expected Defs[Port] to be populated")
		}
		if port.Minimum == nil || *port.Minimum != 1.0 {
			t.Errorf("expected Port.Minimum=1.0, got %v", port.Minimum)
		}
	})

	t.Run("ref_resolved", func(t *testing.T) {
		schema := `{
			"$defs": {
				"Name": {"type": "string", "minLength": 1}
			},
			"type": "object",
			"properties": {
				"name": {"$ref": "#/$defs/Name"}
			}
		}`
		s, err := jsonschema.Parse([]byte(schema))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		nameProp := s.Properties["name"]
		if nameProp == nil {
			t.Fatal("expected name property to exist")
		}
		if nameProp.Ref == nil {
			t.Fatal("expected name.$ref to be resolved to a non-nil Schema")
		}
		if len(nameProp.Ref.Types) != 1 || nameProp.Ref.Types[0] != "string" {
			t.Errorf("expected resolved ref type=string, got %v", nameProp.Ref.Types)
		}
	})

	t.Run("non_fragment_ref_rejected", func(t *testing.T) {
		schema := `{"$ref": "https://example.com/other-schema"}`
		_, err := jsonschema.Parse([]byte(schema))
		if err == nil {
			t.Fatal("expected error for non-fragment $ref, got nil")
		}
	})

	t.Run("unresolvable_ref_rejected", func(t *testing.T) {
		schema := `{"$ref": "#/$defs/DoesNotExist"}`
		_, err := jsonschema.Parse([]byte(schema))
		if err == nil {
			t.Fatal("expected error for unresolvable $ref, got nil")
		}
	})
}
