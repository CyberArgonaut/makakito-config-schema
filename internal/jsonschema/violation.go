// internal/jsonschema/violation.go
package jsonschema

import "fmt"

// Violation describes a single schema constraint that a JSON document failed to satisfy.
type Violation struct {
	Field   string // dot-separated JSON path to the failing node, e.g. "experiment.scenario"
	Keyword string // the Draft-07 keyword whose constraint was violated, e.g. "required"
	Message string // human-readable explanation of the violation
}

// String returns a single-line representation suitable for display in CLI output or error messages.
func (v Violation) String() string {
	return fmt.Sprintf("%s: %s", v.Field, v.Message)
}
