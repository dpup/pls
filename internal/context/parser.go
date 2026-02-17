package context

// Result holds the output of a single parser.
type Result struct {
	Name string         `json:"name"`
	Data map[string]any `json:"data"`
}

// Snapshot is the full context collected from all parsers.
type Snapshot struct {
	RepoRoot string   `json:"repo_root"`
	CwdRel   string   `json:"cwd_rel"`
	Results  []Result `json:"results"`
}

// Parser detects project tooling and extracts relevant info.
// Returns nil result if this parser doesn't apply.
type Parser interface {
	Name() string
	Parse(repoRoot, cwd string) (*Result, error)
}
