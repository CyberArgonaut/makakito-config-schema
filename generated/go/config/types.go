package config

// SchemaVersion is the canonical schema version this package was generated for.
// It must always match the version embedded in schema/config.schema.json ($id field).
const SchemaVersion = "1.0.0"

// MakakitoConfig is the root configuration for a Makakito chaos-engineering playground session.
// Produced by the builder, consumed by the runner and CLI.
type MakakitoConfig struct {
	SchemaVersion string     `json:"schemaVersion"`
	Playground    Playground `json:"playground"`
	Services      []Service  `json:"services"`
	Experiment    Experiment `json:"experiment"`
	Report        *Report    `json:"report,omitempty"`
	Traffic       *Traffic   `json:"traffic,omitempty"`
}

// Playground describes the type and identity of the chaos-engineering environment.
type Playground struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// Service is a single containerised service in the playground.
type Service struct {
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	Type      string            `json:"type"`
	Port      int               `json:"port,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	DependsOn []string          `json:"dependsOn,omitempty"`
	Replicas  int               `json:"replicas,omitempty"`
	Resources *Resources        `json:"resources,omitempty"`
}

// Resources holds CPU and memory limits for a Service container.
type Resources struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// Experiment is the chaos experiment to execute against the playground.
type Experiment struct {
	Name            string         `json:"name"`
	Description     string         `json:"description,omitempty"`
	Scenario        string         `json:"scenario"`
	TargetService   string         `json:"targetService,omitempty"`
	DurationSeconds int            `json:"durationSeconds,omitempty"`
	Parameters      map[string]any `json:"parameters,omitempty"`
}

// Report controls how experiment results are persisted.
type Report struct {
	OutputFormat string `json:"outputFormat"`
	// IncludeMetricsSnapshot is a pointer so that an explicit false survives
	// json.Marshal with omitempty. A plain bool would be dropped as a zero value,
	// breaking config roundtrips. See docs/go-conventions.md — "The *bool trap".
	IncludeMetricsSnapshot *bool  `json:"includeMetricsSnapshot,omitempty"`
	OutputPath             string `json:"outputPath,omitempty"`
}

// Traffic holds inline traffic-generation settings for the experiment.
// A lightweight alternative to a full traffic-profile YAML (v1.1+).
type Traffic struct {
	RequestsPerSecond float64 `json:"requestsPerSecond,omitempty"`
	Connections       int     `json:"connections,omitempty"`
	TargetService     string  `json:"targetService,omitempty"`
	DurationSeconds   int     `json:"durationSeconds,omitempty"`
}
