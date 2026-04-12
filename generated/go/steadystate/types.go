package steadystate

// MakakitoSteadyState defines a steady-state hypothesis: baseline conditions that
// must hold before a chaos experiment begins and should be restored afterward.
type MakakitoSteadyState struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Thresholds  []Threshold `json:"thresholds"`
}

// Threshold is a single measurable condition the system must satisfy.
// Operator semantics: lt (<), lte (≤), gt (>), gte (≥), eq (=).
type Threshold struct {
	Metric      string  `json:"metric"`
	Operator    string  `json:"operator"`
	Value       float64 `json:"value"`
	Description string  `json:"description,omitempty"`
}
