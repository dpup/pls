package llm

import (
	gocontext "context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/dpup/pls/internal/config"
	"github.com/dpup/pls/internal/context"
	"github.com/dpup/pls/internal/history"
)

// Client wraps the Anthropic API for command suggestion.
type Client struct {
	api    anthropic.Client
	config *config.Config
}

// NewClient creates a new LLM client using the provided configuration.
func NewClient(cfg *config.Config) *Client {
	api := anthropic.NewClient(
		option.WithAPIKey(cfg.LLM.APIKey),
	)
	return &Client{
		api:    api,
		config: cfg,
	}
}

// Generate calls the fast model with tools, allowing the LLM to explore the
// project before producing candidates. If the top candidate's confidence is
// below the escalation threshold, it escalates to the strong model (single-shot).
func (c *Client) Generate(snap context.Snapshot, intent string, projectHistory []history.Entry, globalHistory []history.Entry) (*Response, error) {
	prompt := BuildPrompt(intent, &snap, projectHistory, globalHistory)
	handler := &toolHandler{repoRoot: snap.RepoRoot}

	resp, err := c.callWithTools(c.config.LLM.Models.Fast, prompt, handler)
	if err != nil {
		return nil, fmt.Errorf("fast model call: %w", err)
	}

	// Escalate if top candidate confidence is below threshold.
	if len(resp.Candidates) > 0 && resp.Candidates[0].Confidence < c.config.LLM.Models.EscalationThreshold {
		escalated, err := c.call(c.config.LLM.Models.Strong, prompt)
		if err != nil {
			// Fall back to the fast model response if escalation fails.
			return resp, nil
		}
		return escalated, nil
	}

	return resp, nil
}

// Escalate calls the strong model (Sonnet) directly. This is used when the user
// exhausts the fast model's candidates via repeated 'n' presses.
func (c *Client) Escalate(snap context.Snapshot, intent string, projectHistory []history.Entry, globalHistory []history.Entry) (*Response, error) {
	prompt := BuildPrompt(intent, &snap, projectHistory, globalHistory)

	resp, err := c.call(c.config.LLM.Models.Strong, prompt)
	if err != nil {
		return nil, fmt.Errorf("strong model call: %w", err)
	}
	return resp, nil
}

// callWithTools runs a multi-turn loop: the LLM can call tools to explore the
// project, then produce a final JSON response. Capped at maxToolTurns round
// trips to bound latency.
func (c *Client) callWithTools(model, prompt string, handler *toolHandler) (*Response, error) {
	tools := toolDefs()
	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
	}

	var rounds []ToolRound

	for turn := 0; turn <= maxToolTurns; turn++ {
		// On the last turn, omit tools to force a final text response.
		var turnTools []anthropic.ToolUnionParam
		if turn < maxToolTurns {
			turnTools = tools
		}

		msg, err := c.api.Messages.New(gocontext.Background(), anthropic.MessageNewParams{
			Model:     anthropic.Model(model),
			MaxTokens: 1024,
			System: []anthropic.TextBlockParam{
				{Text: SystemPrompt()},
			},
			Messages: messages,
			Tools:    turnTools,
		})
		if err != nil {
			return nil, fmt.Errorf("API call to %s (turn %d): %w", model, turn, err)
		}

		// Separate text and tool_use blocks.
		var textContent string
		var toolUses []struct {
			ID    string
			Name  string
			Input json.RawMessage
		}

		for _, block := range msg.Content {
			switch block.Type {
			case "text":
				textContent = block.Text
			case "tool_use":
				toolUses = append(toolUses, struct {
					ID    string
					Name  string
					Input json.RawMessage
				}{
					ID:    block.ID,
					Name:  block.Name,
					Input: block.Input,
				})
			}
		}

		// No tool calls → parse as final response.
		if len(toolUses) == 0 {
			resp, err := parseTextResponse(textContent, model)
			if err != nil {
				return nil, err
			}
			resp.Rounds = rounds
			return resp, nil
		}

		// Reconstruct the assistant message (text + tool_use blocks).
		var assistantBlocks []anthropic.ContentBlockParamUnion
		for _, block := range msg.Content {
			switch block.Type {
			case "text":
				if block.Text != "" {
					assistantBlocks = append(assistantBlocks, anthropic.NewTextBlock(block.Text))
				}
			case "tool_use":
				var inputObj any
				_ = json.Unmarshal(block.Input, &inputObj)
				assistantBlocks = append(assistantBlocks, anthropic.NewToolUseBlock(block.ID, inputObj, block.Name))
			}
		}
		messages = append(messages, anthropic.NewAssistantMessage(assistantBlocks...))

		// Execute each tool, build result blocks, and record the round.
		var round ToolRound
		var resultBlocks []anthropic.ContentBlockParamUnion
		for _, tu := range toolUses {
			var inputMap map[string]any
			_ = json.Unmarshal(tu.Input, &inputMap)

			result, isError := handler.execute(tu.Name, tu.Input)
			resultBlocks = append(resultBlocks, anthropic.NewToolResultBlock(tu.ID, result, isError))

			round.Calls = append(round.Calls, ToolCall{
				Name:    tu.Name,
				Input:   inputMap,
				Result:  result,
				IsError: isError,
			})
		}
		rounds = append(rounds, round)
		messages = append(messages, anthropic.NewUserMessage(resultBlocks...))
	}

	return nil, fmt.Errorf("exceeded maximum tool turns (%d) without final response", maxToolTurns)
}

// call makes a single-shot API call (no tools). Used for escalation.
func (c *Client) call(model, prompt string) (*Response, error) {
	msg, err := c.api.Messages.New(gocontext.Background(), anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: SystemPrompt()},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("API call to %s: %w", model, err)
	}

	var text string
	for _, block := range msg.Content {
		if block.Type == "text" {
			text = block.Text
			break
		}
	}

	return parseTextResponse(text, model)
}

// parseTextResponse extracts JSON candidates from the LLM's text response.
func parseTextResponse(text, model string) (*Response, error) {
	if text == "" {
		return nil, fmt.Errorf("no text content in response from %s", model)
	}

	text = extractJSON(text)

	var resp Response
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, fmt.Errorf("parsing response JSON from %s: %w", model, err)
	}

	return &resp, nil
}

// extractJSON extracts the JSON object from a response that may contain
// markdown fences or preamble text (common after multi-turn tool use).
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	// Handle ```json ... ``` or ``` ... ```
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
		if idx := strings.LastIndex(s, "```"); idx != -1 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}

	// If the string starts with '{', it's already JSON.
	if strings.HasPrefix(s, "{") {
		return s
	}

	// Otherwise, find the first '{' and last '}' to extract embedded JSON.
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}

	return s
}
