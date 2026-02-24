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
	Rounds     []ToolRound `json:"-"` // tool-use metadata, omitted from JSON output
}

// ToolRound captures one round of tool calls in the multi-turn loop.
type ToolRound struct {
	Calls []ToolCall
}

// ToolCall captures a single tool invocation and its result.
type ToolCall struct {
	Name    string
	Input   map[string]any
	Result  string
	IsError bool
}
