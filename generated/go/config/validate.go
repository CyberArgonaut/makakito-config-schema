package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/CyberArgonaut/makakito-config-schema/internal/jsonschema"
)

var (
	configSchemaOnce sync.Once
	configSchema     *jsonschema.Schema
	configSchemaErr  error
)

func getCompiledSchema() (*jsonschema.Schema, error) {
	configSchemaOnce.Do(func() {
		configSchema, configSchemaErr = jsonschema.Parse(Schema())
	})
	return configSchema, configSchemaErr
}

// Validate validates raw JSON against the embedded config schema.
// Returns a slice of human-readable violation strings on failure, nil on success.
// A non-nil error indicates a failure in the validation engine itself (e.g. malformed
// schema or document), not a schema violation.
func Validate(data []byte) ([]string, error) {
	s, err := getCompiledSchema()
	if err != nil {
		return nil, fmt.Errorf("validation engine error: %w", err)
	}
	violations, err := jsonschema.Validate(s, data)
	if err != nil {
		return nil, fmt.Errorf("validation engine error: %w", err)
	}
	if len(violations) == 0 {
		return nil, nil
	}
	result := make([]string, len(violations))
	for i, v := range violations {
		result[i] = "  • " + v.String()
	}
	return result, nil
}

// Parse validates and unmarshals raw JSON into a MakakitoConfig.
// Returns the parsed config on success, or an error that joins all schema
// violations into a single message.
func Parse(data []byte) (*MakakitoConfig, error) {
	violations, err := Validate(data)
	if err != nil {
		return nil, err
	}
	if len(violations) > 0 {
		errs := make([]error, len(violations))
		for i, v := range violations {
			errs[i] = errors.New(v)
		}
		return nil, errors.Join(errs...)
	}

	var cfg MakakitoConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}
	return &cfg, nil
}
