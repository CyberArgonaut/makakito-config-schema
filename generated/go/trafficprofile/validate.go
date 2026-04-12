package trafficprofile

import "github.com/CyberArgonaut/makakito-config-schema/internal/jsonschema"

// Validate is a no-op stub. The traffic-profile schema is a placeholder ({})
// pending stabilisation of makakito-traffic-profiles (see DECISIONS.md ADR-007, v1.1).
// All documents are considered valid until the real schema is defined.
func Validate(_ []byte) ([]jsonschema.Violation, error) {
	return nil, nil
}
