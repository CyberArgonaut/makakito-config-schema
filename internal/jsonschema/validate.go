// internal/jsonschema/validate.go
package jsonschema

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Validate validates a JSON document against a compiled Schema.
// Returns (nil, nil) on success, ([]Violation, nil) on schema violations,
// and (nil, error) when the document cannot be decoded.
func Validate(schema *Schema, document []byte) ([]Violation, error) {
	dec := json.NewDecoder(strings.NewReader(string(document)))
	dec.UseNumber()
	var node any
	if err := dec.Decode(&node); err != nil {
		return nil, fmt.Errorf("document decode error: %w", err)
	}
	violations := validateNode(schema, node, "")
	if len(violations) == 0 {
		return nil, nil
	}
	return violations, nil
}

// childPath builds a dot-separated document path.
// Returns key alone when parent is empty (top-level field), otherwise parent.key.
func childPath(parent, key string) string {
	if parent == "" {
		return key
	}
	return parent + "." + key
}

// validateNode recursively validates node against schema and collects violations.
func validateNode(schema *Schema, node any, path string) []Violation {
	// 1. $ref — delegate entirely; ignore all other keywords on this schema node.
	if schema.Ref != nil {
		return validateNode(schema.Ref, node, path)
	}

	var violations []Violation

	// 2. type — wrong type → return early; further checks are meaningless on a mis-typed value.
	if len(schema.Types) > 0 && !typeMatches(schema.Types, node) {
		return []Violation{{
			Field:   path,
			Keyword: "type",
			Message: fmt.Sprintf("expected %s, got %s", strings.Join(schema.Types, " or "), typeName(node)),
		}}
	}

	// 3. enum
	if len(schema.Enum) > 0 {
		violations = append(violations, validateEnum(schema, node, path)...)
	}

	// 4. const
	if schema.Const != nil {
		violations = append(violations, validateConst(schema, node, path)...)
	}

	// 5. composition
	if len(schema.AnyOf) > 0 || len(schema.AllOf) > 0 || len(schema.OneOf) > 0 {
		violations = append(violations, validateComposition(schema, node, path)...)
	}

	// 6. type-specific validators
	switch v := node.(type) {
	case map[string]any:
		violations = append(violations, validateObject(schema, v, path)...)
	case []any:
		violations = append(violations, validateArray(schema, v, path)...)
	case string:
		violations = append(violations, validateString(schema, v, path)...)
	case json.Number:
		violations = append(violations, validateNumber(schema, v, path)...)
	}

	return violations
}

// typeName returns the JSON Schema type name for a decoded Go value.
func typeName(node any) string {
	switch v := node.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	case json.Number:
		// A number with no decimal point or exponent is an integer.
		if !strings.ContainsAny(v.String(), ".eE") {
			return "integer"
		}
		return "number"
	default:
		return "unknown"
	}
}

// typeMatches returns true if node's type is among the allowed types.
// "number" also matches "integer" values per JSON Schema spec.
func typeMatches(types []string, node any) bool {
	actual := typeName(node)
	for _, t := range types {
		if t == actual {
			return true
		}
		if t == "number" && actual == "integer" {
			return true
		}
	}
	return false
}

// validateEnum checks that node's JSON representation equals one of the allowed enum values.
func validateEnum(schema *Schema, node any, path string) []Violation {
	nodeJSON, err := json.Marshal(node)
	if err != nil {
		return nil
	}
	for _, allowed := range schema.Enum {
		allowedJSON, err := json.Marshal(allowed)
		if err != nil {
			continue
		}
		if string(allowedJSON) == string(nodeJSON) {
			return nil // match found
		}
	}
	parts := make([]string, 0, len(schema.Enum))
	for _, v := range schema.Enum {
		b, _ := json.Marshal(v)
		parts = append(parts, string(b))
	}
	return []Violation{{
		Field:   path,
		Keyword: "enum",
		Message: fmt.Sprintf("value must be one of: %s", strings.Join(parts, ", ")),
	}}
}

// validateConst checks that node's JSON representation exactly equals the const value.
func validateConst(schema *Schema, node any, path string) []Violation {
	nodeJSON, err := json.Marshal(node)
	if err != nil {
		return nil
	}
	constJSON, err := json.Marshal(*schema.Const)
	if err != nil {
		return nil
	}
	if string(nodeJSON) == string(constJSON) {
		return nil
	}
	return []Violation{{
		Field:   path,
		Keyword: "const",
		Message: fmt.Sprintf("value must be %s", string(constJSON)),
	}}
}

// validateComposition evaluates anyOf, allOf, and oneOf constraints.
func validateComposition(schema *Schema, node any, path string) []Violation {
	var violations []Violation

	// allOf: every branch must pass; include violations from failing branches.
	for _, sub := range schema.AllOf {
		violations = append(violations, validateNode(sub, node, path)...)
	}

	// anyOf: at least one branch must pass; surface violations from the closest match.
	if len(schema.AnyOf) > 0 {
		var bestMatch []Violation
		passed := false
		for _, sub := range schema.AnyOf {
			v := validateNode(sub, node, path)
			if len(v) == 0 {
				passed = true
				break
			}
			if bestMatch == nil || len(v) < len(bestMatch) {
				bestMatch = v
			}
		}
		if !passed {
			violations = append(violations, Violation{
				Field:   path,
				Keyword: "anyOf",
				Message: "value does not match any of the expected schemas",
			})
			violations = append(violations, bestMatch...)
		}
	}

	// oneOf: exactly one branch must pass.
	if len(schema.OneOf) > 0 {
		var bestMatch []Violation
		passCount := 0
		for _, sub := range schema.OneOf {
			v := validateNode(sub, node, path)
			if len(v) == 0 {
				passCount++
			} else if bestMatch == nil || len(v) < len(bestMatch) {
				bestMatch = v
			}
		}
		if passCount != 1 {
			violations = append(violations, Violation{
				Field:   path,
				Keyword: "oneOf",
				Message: fmt.Sprintf("value must match exactly one schema, but matched %d", passCount),
			})
			if passCount == 0 {
				violations = append(violations, bestMatch...)
			}
		}
	}

	return violations
}

// validateObject enforces required, properties, and additionalProperties constraints.
func validateObject(schema *Schema, node map[string]any, path string) []Violation {
	var violations []Violation

	// required: every named property must be present.
	for _, name := range schema.Required {
		if _, ok := node[name]; !ok {
			violations = append(violations, Violation{
				Field:   childPath(path, name),
				Keyword: "required",
				Message: name + " is required",
			})
		}
	}

	// properties and additionalProperties
	for key, val := range node {
		child := childPath(path, key)
		if sub, ok := schema.Properties[key]; ok {
			// Known property — validate recursively against its sub-schema.
			violations = append(violations, validateNode(sub, val, child)...)
		} else if schema.AdditionalProperties.Present {
			if schema.AdditionalProperties.Schema != nil {
				// Additional property must conform to the given sub-schema.
				violations = append(violations, validateNode(schema.AdditionalProperties.Schema, val, child)...)
			} else if !schema.AdditionalProperties.Allowed {
				// Additional properties are forbidden.
				violations = append(violations, Violation{
					Field:   child,
					Keyword: "additionalProperties",
					Message: fmt.Sprintf("additional property %q is not allowed", key),
				})
			}
			// else: Allowed==true → any additional property is fine.
		}
		// else: additionalProperties absent → all additional properties are implicitly allowed.
	}

	return violations
}

// validateArray enforces items, minItems, maxItems, and uniqueItems constraints.
func validateArray(schema *Schema, node []any, path string) []Violation {
	var violations []Violation

	if schema.MinItems != nil && len(node) < *schema.MinItems {
		violations = append(violations, Violation{
			Field:   path,
			Keyword: "minItems",
			Message: fmt.Sprintf("array must have at least %d items, got %d", *schema.MinItems, len(node)),
		})
	}
	if schema.MaxItems != nil && len(node) > *schema.MaxItems {
		violations = append(violations, Violation{
			Field:   path,
			Keyword: "maxItems",
			Message: fmt.Sprintf("array must have at most %d items, got %d", *schema.MaxItems, len(node)),
		})
	}

	if schema.Items != nil {
		for i, elem := range node {
			child := fmt.Sprintf("%s[%d]", path, i)
			violations = append(violations, validateNode(schema.Items, elem, child)...)
		}
	}

	if schema.UniqueItems && len(node) > 1 {
		seen := map[string]int{}
		for i, elem := range node {
			b, err := json.Marshal(elem)
			if err != nil {
				continue
			}
			key := string(b)
			if j, ok := seen[key]; ok {
				violations = append(violations, Violation{
					Field:   path,
					Keyword: "uniqueItems",
					Message: fmt.Sprintf("items at [%d] and [%d] are not unique", j, i),
				})
			} else {
				seen[key] = i
			}
		}
	}

	return violations
}

// validateString enforces minLength, maxLength, and pattern constraints.
// Length is measured in Unicode code points, not bytes.
func validateString(schema *Schema, node string, path string) []Violation {
	var violations []Violation

	runeCount := utf8.RuneCountInString(node)

	if schema.MinLength != nil && runeCount < *schema.MinLength {
		violations = append(violations, Violation{
			Field:   path,
			Keyword: "minLength",
			Message: fmt.Sprintf("string must be at least %d characters, got %d", *schema.MinLength, runeCount),
		})
	}
	if schema.MaxLength != nil && runeCount > *schema.MaxLength {
		violations = append(violations, Violation{
			Field:   path,
			Keyword: "maxLength",
			Message: fmt.Sprintf("string must be at most %d characters, got %d", *schema.MaxLength, runeCount),
		})
	}
	if schema.Pattern != nil && !schema.Pattern.MatchString(node) {
		violations = append(violations, Violation{
			Field:   path,
			Keyword: "pattern",
			Message: fmt.Sprintf("string does not match pattern %q", schema.Pattern.String()),
		})
	}

	return violations
}

// validateNumber enforces minimum, maximum, exclusiveMinimum, and exclusiveMaximum constraints.
// Also enforces "integer" type: a json.Number containing a decimal point or exponent fails.
func validateNumber(schema *Schema, node json.Number, path string) []Violation {
	var violations []Violation

	f, err := strconv.ParseFloat(node.String(), 64)
	if err != nil {
		// json.Number is already validated by the decoder; this should not happen.
		return violations
	}

	if schema.Minimum != nil && f < *schema.Minimum {
		violations = append(violations, Violation{
			Field:   path,
			Keyword: "minimum",
			Message: fmt.Sprintf("value %s is less than minimum %v", node, *schema.Minimum),
		})
	}
	if schema.Maximum != nil && f > *schema.Maximum {
		violations = append(violations, Violation{
			Field:   path,
			Keyword: "maximum",
			Message: fmt.Sprintf("value %s is greater than maximum %v", node, *schema.Maximum),
		})
	}
	if schema.ExclusiveMinimum != nil && f <= *schema.ExclusiveMinimum {
		violations = append(violations, Violation{
			Field:   path,
			Keyword: "exclusiveMinimum",
			Message: fmt.Sprintf("value %s is not greater than exclusive minimum %v", node, *schema.ExclusiveMinimum),
		})
	}
	if schema.ExclusiveMaximum != nil && f >= *schema.ExclusiveMaximum {
		violations = append(violations, Violation{
			Field:   path,
			Keyword: "exclusiveMaximum",
			Message: fmt.Sprintf("value %s is not less than exclusive maximum %v", node, *schema.ExclusiveMaximum),
		})
	}

	return violations
}
