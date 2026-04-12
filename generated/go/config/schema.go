package config

import schemapkg "github.com/CyberArgonaut/makakito-config-schema/schema"

// Schema returns the raw JSON Schema bytes for MakakitoConfig.
// The bytes are embedded at compile time via the schema package, which hosts
// the //go:embed directives for all schema files. Go's embed path restrictions
// prohibit '..' traversal, so embedding must originate from within schema/.
//
// The returned slice is the live embedded data; callers must not modify it.
func Schema() []byte {
	return schemapkg.Config
}
