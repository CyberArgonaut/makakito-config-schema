// internal/jsonschema/parse.go
package jsonschema

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// supportedKeywords is the complete set of Draft-07 keywords this validator handles.
// Any keyword not in this set causes Parse to return an error immediately,
// regardless of where in the schema tree it appears.
var supportedKeywords = map[string]struct{}{
	"$schema":              {},
	"$id":                  {},
	"$ref":                 {},
	"$defs":                {},
	"definitions":          {}, // Draft-04/06 alias for $defs; accepted per project convention
	"title":                {},
	"description":          {},
	"default":              {},
	"example":              {}, // non-standard singular form; accepted per project convention
	"type":                 {},
	"properties":           {},
	"required":             {},
	"additionalProperties": {},
	"items":                {},
	"minItems":             {},
	"maxItems":             {},
	"uniqueItems":          {},
	"minLength":            {},
	"maxLength":            {},
	"pattern":              {},
	"minimum":              {},
	"maximum":              {},
	"exclusiveMinimum":     {},
	"exclusiveMaximum":     {},
	"enum":                 {},
	"const":                {},
	"anyOf":                {},
	"allOf":                {},
	"oneOf":                {},
}

// Parse compiles raw JSON Schema bytes into a validated, resolved Schema tree.
// Returns an error if the JSON is malformed, any keyword is unsupported,
// any $ref is not a fragment-only reference pointing into $defs, or a $ref
// target cannot be found.
func Parse(data []byte) (*Schema, error) {
	var raw map[string]any
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	if err := checkKeywords(raw, ""); err != nil {
		return nil, err
	}

	// Build the root defs map first so resolveRefs can look names up.
	// Both "$defs" (Draft-07) and "definitions" (Draft-04/06) are supported.
	defs := map[string]*Schema{}
	for _, defsKey := range []string{"$defs", "definitions"} {
		if rawDefs, ok := raw[defsKey].(map[string]any); ok {
			for name, v := range rawDefs {
				sub, ok := v.(map[string]any)
				if !ok {
					continue
				}
				defSchema, err := parseNode(sub, defs, schemaPart(defsKey, name))
				if err != nil {
					return nil, err
				}
				defs[name] = defSchema
			}
		}
	}

	s, err := parseNode(raw, defs, "")
	if err != nil {
		return nil, err
	}
	s.Defs = defs

	// Second pass: resolve all $ref strings to their Schema pointers.
	if err := resolveRefs(s, defs, nil); err != nil {
		return nil, err
	}

	return s, nil
}

// schemaPart builds a dot-separated schema path string (e.g. "$defs.Service").
func schemaPart(parent, key string) string {
	if parent == "" {
		return key
	}
	return parent + "." + key
}

// checkKeywords recursively verifies that every key in raw (and nested schemas)
// is in the supportedKeywords set. Returns an error naming the first unsupported keyword found.
func checkKeywords(raw map[string]any, path string) error {
	for k := range raw {
		if _, ok := supportedKeywords[k]; !ok {
			location := k
			if path != "" {
				location = path + "." + k
			}
			return fmt.Errorf("unsupported keyword %q at schema path %q", k, location)
		}
	}

	// Recurse into nested schema objects.
	if props, ok := raw["properties"].(map[string]any); ok {
		for name, v := range props {
			sub, ok := v.(map[string]any)
			if !ok {
				continue
			}
			if err := checkKeywords(sub, schemaPart(schemaPart(path, "properties"), name)); err != nil {
				return err
			}
		}
	}
	if rawItems, ok := raw["items"]; ok {
		if items, ok := rawItems.(map[string]any); ok {
			if err := checkKeywords(items, schemaPart(path, "items")); err != nil {
				return err
			}
		}
	}
	for _, kw := range []string{"anyOf", "allOf", "oneOf"} {
		if arr, ok := raw[kw].([]any); ok {
			for i, v := range arr {
				sub, ok := v.(map[string]any)
				if !ok {
					continue
				}
				if err := checkKeywords(sub, fmt.Sprintf("%s[%d]", schemaPart(path, kw), i)); err != nil {
					return err
				}
			}
		}
	}
	if ap, ok := raw["additionalProperties"].(map[string]any); ok {
		if err := checkKeywords(ap, schemaPart(path, "additionalProperties")); err != nil {
			return err
		}
	}
	for _, defsKey := range []string{"$defs", "definitions"} {
		if rawDefs, ok := raw[defsKey].(map[string]any); ok {
			for name, v := range rawDefs {
				sub, ok := v.(map[string]any)
				if !ok {
					continue
				}
				if err := checkKeywords(sub, schemaPart(schemaPart(path, defsKey), name)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// parseNode populates a Schema struct from a decoded map[string]any.
// defs is the root $defs map used during nested schema parsing; refs are resolved
// in a separate pass after all nodes are parsed.
func parseNode(raw map[string]any, defs map[string]*Schema, path string) (*Schema, error) {
	s := &Schema{}

	// Metadata
	if v, ok := raw["$schema"].(string); ok {
		s.SchemaURI = &v
	}
	if v, ok := raw["$id"].(string); ok {
		s.ID = &v
	}
	if v, ok := raw["title"].(string); ok {
		s.Title = &v
	}
	if v, ok := raw["description"].(string); ok {
		s.Description = &v
	}
	if v, ok := raw["default"]; ok {
		s.Default = v
	}
	if v, ok := raw["example"]; ok {
		s.Example = v
	}

	// $ref — store raw string; resolved in resolveRefs pass.
	if v, ok := raw["$ref"].(string); ok {
		s.rawRef = v
	}

	// type: string or []string
	switch v := raw["type"].(type) {
	case string:
		s.Types = []string{v}
	case []any:
		for i, t := range v {
			ts, ok := t.(string)
			if !ok {
				return nil, fmt.Errorf("keyword \"type\"[%d] must be a string at schema path %q", i, path)
			}
			s.Types = append(s.Types, ts)
		}
	case nil:
		// absent — no type constraint
	default:
		return nil, fmt.Errorf("keyword \"type\" must be a string or array at schema path %q", path)
	}

	// required
	if v, ok := raw["required"].([]any); ok {
		for i, item := range v {
			name, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("keyword \"required\"[%d] must be a string at schema path %q", i, path)
			}
			s.Required = append(s.Required, name)
		}
	}

	// uniqueItems
	if v, ok := raw["uniqueItems"].(bool); ok {
		s.UniqueItems = v
	}

	// minItems / maxItems
	if v, ok := raw["minItems"].(json.Number); ok {
		n, err := strconv.Atoi(v.String())
		if err != nil || n < 0 {
			return nil, fmt.Errorf("keyword \"minItems\" must be a non-negative integer at schema path %q", path)
		}
		s.MinItems = &n
	}
	if v, ok := raw["maxItems"].(json.Number); ok {
		n, err := strconv.Atoi(v.String())
		if err != nil || n < 0 {
			return nil, fmt.Errorf("keyword \"maxItems\" must be a non-negative integer at schema path %q", path)
		}
		s.MaxItems = &n
	}

	// minLength / maxLength
	if v, ok := raw["minLength"].(json.Number); ok {
		n, err := strconv.Atoi(v.String())
		if err != nil || n < 0 {
			return nil, fmt.Errorf("keyword \"minLength\" must be a non-negative integer at schema path %q", path)
		}
		s.MinLength = &n
	}
	if v, ok := raw["maxLength"].(json.Number); ok {
		n, err := strconv.Atoi(v.String())
		if err != nil || n < 0 {
			return nil, fmt.Errorf("keyword \"maxLength\" must be a non-negative integer at schema path %q", path)
		}
		s.MaxLength = &n
	}

	// pattern
	if v, ok := raw["pattern"].(string); ok {
		re, err := regexp.Compile(v)
		if err != nil {
			return nil, fmt.Errorf("keyword \"pattern\" is not a valid regular expression at schema path %q: %w", path, err)
		}
		s.Pattern = re
	}

	// numeric bounds
	for _, kw := range []string{"minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum"} {
		if v, ok := raw[kw].(json.Number); ok {
			f, err := strconv.ParseFloat(v.String(), 64)
			if err != nil {
				return nil, fmt.Errorf("keyword %q must be a number at schema path %q", kw, path)
			}
			switch kw {
			case "minimum":
				s.Minimum = &f
			case "maximum":
				s.Maximum = &f
			case "exclusiveMinimum":
				s.ExclusiveMinimum = &f
			case "exclusiveMaximum":
				s.ExclusiveMaximum = &f
			}
		}
	}

	// enum
	if v, ok := raw["enum"].([]any); ok {
		s.Enum = v
	}

	// const
	if v, ok := raw["const"]; ok {
		s.Const = &v
	}

	// properties
	if rawProps, ok := raw["properties"].(map[string]any); ok {
		s.Properties = make(map[string]*Schema, len(rawProps))
		for name, v := range rawProps {
			sub, ok := v.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("property %q must be a schema object at schema path %q", name, path)
			}
			child, err := parseNode(sub, defs, schemaPart(schemaPart(path, "properties"), name))
			if err != nil {
				return nil, err
			}
			s.Properties[name] = child
		}
	}

	// items — only object form supported; tuple form (array) is rejected
	if rawItems, ok := raw["items"]; ok {
		switch v := rawItems.(type) {
		case map[string]any:
			child, err := parseNode(v, defs, schemaPart(path, "items"))
			if err != nil {
				return nil, err
			}
			s.Items = child
		case []any:
			return nil, fmt.Errorf("keyword \"items\" as array (tuple form) is not supported at schema path %q", path)
		default:
			return nil, fmt.Errorf("keyword \"items\" must be a schema object at schema path %q", path)
		}
	}

	// additionalProperties — false, true, or sub-schema
	if rawAP, ok := raw["additionalProperties"]; ok {
		s.AdditionalProperties.Present = true
		switch v := rawAP.(type) {
		case bool:
			s.AdditionalProperties.Allowed = v
		case map[string]any:
			child, err := parseNode(v, defs, schemaPart(path, "additionalProperties"))
			if err != nil {
				return nil, err
			}
			s.AdditionalProperties.Schema = child
		default:
			return nil, fmt.Errorf("keyword \"additionalProperties\" must be a boolean or schema object at schema path %q", path)
		}
	}

	// composition keywords: anyOf, allOf, oneOf
	for _, kw := range []string{"anyOf", "allOf", "oneOf"} {
		if arr, ok := raw[kw].([]any); ok {
			branches := make([]*Schema, 0, len(arr))
			for i, v := range arr {
				sub, ok := v.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("keyword %q[%d] must be a schema object at schema path %q", kw, i, path)
				}
				child, err := parseNode(sub, defs, fmt.Sprintf("%s[%d]", schemaPart(path, kw), i))
				if err != nil {
					return nil, err
				}
				branches = append(branches, child)
			}
			switch kw {
			case "anyOf":
				s.AnyOf = branches
			case "allOf":
				s.AllOf = branches
			case "oneOf":
				s.OneOf = branches
			}
		}
	}

	return s, nil
}

// resolveRefs performs a second pass over the parsed Schema tree, replacing
// rawRef strings with pointers to the corresponding $defs entry.
// visiting tracks the schemas currently on the resolution stack to detect cycles.
func resolveRefs(s *Schema, defs map[string]*Schema, visiting map[*Schema]bool) error {
	if s == nil {
		return nil
	}
	if visiting == nil {
		visiting = map[*Schema]bool{}
	}
	if visiting[s] {
		return fmt.Errorf("circular $ref detected")
	}
	visiting[s] = true
	defer func() { visiting[s] = false }()

	if s.rawRef != "" {
		target, err := parseDefRef(s.rawRef, defs)
		if err != nil {
			return err
		}
		s.Ref = target
		s.rawRef = ""
	}

	for _, child := range s.Properties {
		if err := resolveRefs(child, defs, visiting); err != nil {
			return err
		}
	}
	if err := resolveRefs(s.Items, defs, visiting); err != nil {
		return err
	}
	if err := resolveRefs(s.AdditionalProperties.Schema, defs, visiting); err != nil {
		return err
	}
	for _, child := range s.AnyOf {
		if err := resolveRefs(child, defs, visiting); err != nil {
			return err
		}
	}
	for _, child := range s.AllOf {
		if err := resolveRefs(child, defs, visiting); err != nil {
			return err
		}
	}
	for _, child := range s.OneOf {
		if err := resolveRefs(child, defs, visiting); err != nil {
			return err
		}
	}
	// Defs entries themselves may contain $refs to sibling defs.
	for _, def := range s.Defs {
		if err := resolveRefs(def, defs, visiting); err != nil {
			return err
		}
	}

	return nil
}

// parseDefRef resolves a $ref string of the form "#/$defs/<Name>" or
// "#/definitions/<Name>" to the corresponding *Schema in defs.
// Any other form is rejected with an error.
func parseDefRef(ref string, defs map[string]*Schema) (*Schema, error) {
	var name string
	switch {
	case strings.HasPrefix(ref, "#/$defs/"):
		name = strings.TrimPrefix(ref, "#/$defs/")
	case strings.HasPrefix(ref, "#/definitions/"):
		name = strings.TrimPrefix(ref, "#/definitions/")
	default:
		return nil, fmt.Errorf("$ref %q is not supported: only fragment refs of the form \"#/$defs/<Name>\" or \"#/definitions/<Name>\" are allowed", ref)
	}
	target, ok := defs[name]
	if !ok {
		return nil, fmt.Errorf("$ref %q could not be resolved: no definition named %q", ref, name)
	}
	return target, nil
}
