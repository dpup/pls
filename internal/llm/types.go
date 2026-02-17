package llm

// Candidate represents a single command suggestion from the LLM.
type Candidate struct {
	Cmd        string  `json:"cmd"`
	Reason     string  `json:"reason"`
	Confidence float64 `json:"confidence"`
	Risk       string  `json:"risk"` // "safe", "moderate", "dangerous"
}

// Response holds the list of candidate commands returned by the LLM.
type Response struct {
	Candidates []Candidate `json:"candidates"`
}
