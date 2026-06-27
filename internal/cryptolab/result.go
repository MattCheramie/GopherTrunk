package cryptolab

// Result is the uniform structured output every mode returns, so the CLI
// can render any tool's output as text, JSON, JSONL, YAML, or CSV without
// knowing the tool's internals.
type Result struct {
	Tool    string `json:"tool" yaml:"tool"`
	Mode    string `json:"mode" yaml:"mode"`
	Summary string `json:"summary" yaml:"summary"`

	// Fields holds the headline structured facts (counts, coverage,
	// recovered parameters). Ordered for stable rendering.
	Fields []Field `json:"fields,omitempty" yaml:"fields,omitempty"`

	// Findings are ranked candidate outputs (recovered keys, surviving
	// wirings, decoded messages). Highest score first.
	Findings []Finding `json:"findings,omitempty" yaml:"findings,omitempty"`

	// Notes are honest caveats: what is and is not established, and what
	// additional data would close the remaining gap.
	Notes []string `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// Field is one named structured fact.
type Field struct {
	Key   string `json:"key" yaml:"key"`
	Value any    `json:"value" yaml:"value"`
}

// Finding is one ranked candidate output.
type Finding struct {
	Label  string         `json:"label" yaml:"label"`
	Score  float64        `json:"score" yaml:"score"`
	Detail map[string]any `json:"detail,omitempty" yaml:"detail,omitempty"`
}

// AddField appends a structured fact.
func (r *Result) AddField(key string, value any) {
	r.Fields = append(r.Fields, Field{Key: key, Value: value})
}

// AddFinding appends a ranked candidate.
func (r *Result) AddFinding(label string, score float64, detail map[string]any) {
	r.Findings = append(r.Findings, Finding{Label: label, Score: score, Detail: detail})
}

// Note appends an honest caveat.
func (r *Result) Note(s string) { r.Notes = append(r.Notes, s) }
