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

// Generate calls the fast model (Haiku) first. If the top candidate's confidence
// is below the escalation threshold, it automatically escalates to the strong
// model (Sonnet).
func (c *Client) Generate(ctx context.Snapshot, intent string, projectHistory []history.Entry, globalHistory []history.Entry) (*Response, error) {
	prompt := BuildPrompt(intent, &ctx, projectHistory, globalHistory)

	resp, err := c.call(c.config.LLM.Models.Fast, prompt)
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
func (c *Client) Escalate(ctx context.Snapshot, intent string, projectHistory []history.Entry, globalHistory []history.Entry) (*Response, error) {
	prompt := BuildPrompt(intent, &ctx, projectHistory, globalHistory)

	resp, err := c.call(c.config.LLM.Models.Strong, prompt)
	if err != nil {
		return nil, fmt.Errorf("strong model call: %w", err)
	}
	return resp, nil
}

// call makes an API call to the specified model and parses the JSON response.
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

	// Extract text content from the response.
	var text string
	for _, block := range msg.Content {
		if block.Type == "text" {
			text = block.Text
			break
		}
	}

	if text == "" {
		return nil, fmt.Errorf("no text content in response from %s", model)
	}

	// Strip markdown fences if present.
	text = stripMarkdownFences(text)

	var resp Response
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, fmt.Errorf("parsing response JSON from %s: %w", model, err)
	}

	return &resp, nil
}

// stripMarkdownFences removes ```json ... ``` or ``` ... ``` wrappers if present.
func stripMarkdownFences(s string) string {
	s = strings.TrimSpace(s)

	// Handle ```json ... ``` or ``` ... ```
	if strings.HasPrefix(s, "```") {
		// Remove the opening fence line.
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
		// Remove the closing fence.
		if idx := strings.LastIndex(s, "```"); idx != -1 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}

	return s
}
