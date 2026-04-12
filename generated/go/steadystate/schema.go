package steadystate

import schemapkg "github.com/CyberArgonaut/makakito-config-schema/schema"

// Schema returns the raw JSON Schema bytes for MakakitoSteadyState.
// The bytes are embedded at compile time via the schema package.
// The returned slice is the live embedded data; callers must not modify it.
func Schema() []byte {
	return schemapkg.SteadyState
}
