package trafficprofile

import schemapkg "github.com/CyberArgonaut/makakito-config-schema/schema"

// Schema returns the raw JSON Schema bytes for the traffic-profile schema.
// The schema is a placeholder ({}) in v1.0.0 pending stabilisation of
// makakito-traffic-profiles. See DECISIONS.md ADR-007.
// The returned slice is the live embedded data; callers must not modify it.
func Schema() []byte {
	return schemapkg.TrafficProfile
}
