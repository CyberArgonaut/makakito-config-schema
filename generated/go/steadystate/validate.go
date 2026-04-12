package steadystate

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/CyberArgonaut/makakito-config-schema/internal/jsonschema"
)

var (
	steadyStateSchemaOnce sync.Once
	steadyStateSchema     *jsonschema.Schema
	steadyStateSchemaErr  error
)

func getCompiledSchema() (*jsonschema.Schema, error) {
	steadyStateSchemaOnce.Do(func() {
		steadyStateSchema, steadyStateSchemaErr = jsonschema.Parse(Schema())
	})
	return steadyStateSchema, steadyStateSchemaErr
}

// Validate validates raw JSON against the embedded steady-state schema.
// Returns structured violations on failure, nil on success.
// A non-nil error indicates an engine failure (malformed schema or document).
func Validate(data []byte) ([]jsonschema.Violation, error) {
	s, err := getCompiledSchema()
	if err != nil {
		return nil, fmt.Errorf("validation engine error: %w", err)
	}
	return jsonschema.Validate(s, data)
}

// Parse validates and unmarshals raw JSON into a MakakitoSteadyState.
func Parse(data []byte) (*MakakitoSteadyState, error) {
	violations, err := Validate(data)
	if err != nil {
		return nil, err
	}
	if len(violations) > 0 {
		errs := make([]error, len(violations))
		for i, v := range violations {
			errs[i] = errors.New(v.String())
		}
		return nil, errors.Join(errs...)
	}

	var ss MakakitoSteadyState
	if err := json.Unmarshal(data, &ss); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}
	return &ss, nil
}
