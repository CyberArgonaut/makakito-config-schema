package scenario

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/CyberArgonaut/makakito-config-schema/internal/jsonschema"
)

var (
	scenarioSchemaOnce sync.Once
	scenarioSchema     *jsonschema.Schema
	scenarioSchemaErr  error
)

func getCompiledSchema() (*jsonschema.Schema, error) {
	scenarioSchemaOnce.Do(func() {
		scenarioSchema, scenarioSchemaErr = jsonschema.Parse(Schema())
	})
	return scenarioSchema, scenarioSchemaErr
}

// Validate validates raw JSON against the embedded scenario schema.
// Returns structured violations on failure, nil on success.
// A non-nil error indicates an engine failure (malformed schema or document).
func Validate(data []byte) ([]jsonschema.Violation, error) {
	s, err := getCompiledSchema()
	if err != nil {
		return nil, fmt.Errorf("validation engine error: %w", err)
	}
	return jsonschema.Validate(s, data)
}

// Parse validates and unmarshals raw JSON into a MakakitoScenario.
func Parse(data []byte) (*MakakitoScenario, error) {
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

	var sc MakakitoScenario
	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}
	return &sc, nil
}
