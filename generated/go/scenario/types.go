package scenario

// MakakitoScenario defines a chaos scenario: an ordered list of faults to inject
// into a playground service. Consumed by the runner; referenced by Experiment.Scenario.
type MakakitoScenario struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Defaults    *Defaults `json:"defaults,omitempty"`
	Faults      []Fault   `json:"faults"`
}

// Defaults provides timing values applied to every Fault that omits its own.
type Defaults struct {
	DurationSeconds int `json:"durationSeconds,omitempty"`
	// CooldownSeconds has schema minimum: 0, so 0 is a valid value.
	// omitempty drops 0, but 0 means "no cooldown" — semantically identical to omitted.
	// If a future roundtrip test covers scenarios, this field must become *int.
	CooldownSeconds int `json:"cooldownSeconds,omitempty"`
}

// Fault is a single fault injection step within a scenario.
type Fault struct {
	Type            string         `json:"type"`
	Description     string         `json:"description,omitempty"`
	DurationSeconds int            `json:"durationSeconds,omitempty"`
	Parameters      map[string]any `json:"parameters,omitempty"`
}
