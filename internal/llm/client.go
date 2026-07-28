package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

func New(baseURL, apiKey, model string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		http:    http.DefaultClient,
	}
}

type chatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function toolCallFunc `json:"function"`
}

type toolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type toolDef struct {
	Type     string      `json:"type"`
	Function toolFuncDef `json:"function"`
}

type toolFuncDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  toolFuncParams `json:"parameters"`
}

type toolFuncParams struct {
	Type       string             `json:"type"`
	Properties map[string]propDef `json:"properties"`
	Required   []string           `json:"required"`
}

type propDef struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type chatRequest struct {
	Model      string        `json:"model"`
	Messages   []chatMessage `json:"messages"`
	MaxTokens  int           `json:"max_tokens,omitempty"`
	Tools      []toolDef     `json:"tools,omitempty"`
	ToolChoice string        `json:"tool_choice,omitempty"`
}

type chatChoice struct {
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Usage   *usageInfo   `json:"usage,omitempty"`
}

type usageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (c *Client) Generate(ctx context.Context, system, user string) (string, error) {
	resp, err := c.chatComplete(ctx, []chatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}, false)
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

// ============================================================
// Tool calling types and executor
// ============================================================

// ToolExecutor is implemented by git.Repo to provide file access for iterative PR generation.
type ToolExecutor interface {
	ReadCurrentFile(path string, start, end int) (string, error)
	ReadBaseFile(path string, start, end int) (string, error)
	ShowFileDiff(path string) (string, error)
	BaseBranch() string
	ChangedFiles() (string, error)
}

// ProgressFn is called after each LLM iteration with the current step and limit.
type ProgressFn func(step, maxSteps int)

func toolDefinitions() []toolDef {
	return []toolDef{{
		Type: "function",
		Function: toolFuncDef{
			Name:        "read_current",
			Description: "Read lines from a file in the current (HEAD) version. Returns content of lines [start, end] (1-based, inclusive).",
			Parameters: toolFuncParams{
				Type: "object",
				Properties: map[string]propDef{
					"path":  {Type: "string", Description: "Relative file path"},
					"start": {Type: "integer", Description: "Start line (1-based, inclusive)"},
					"end":   {Type: "integer", Description: "End line (1-based, inclusive)"},
				},
				Required: []string{"path", "start", "end"},
			},
		},
	}, {
		Type: "function",
		Function: toolFuncDef{
			Name:        "read_base",
			Description: "Read lines from the base/previous version of a file. Returns content of lines [start, end] (1-based, inclusive).",
			Parameters: toolFuncParams{
				Type: "object",
				Properties: map[string]propDef{
					"path":  {Type: "string", Description: "Relative file path"},
					"start": {Type: "integer", Description: "Start line (1-based, inclusive)"},
					"end":   {Type: "integer", Description: "End line (1-based, inclusive)"},
				},
				Required: []string{"path", "start", "end"},
			},
		},
	}, {
		Type: "function",
		Function: toolFuncDef{
			Name:        "show_diff",
			Description: "Show the unified diff of a specific file between HEAD and the base branch",
			Parameters: toolFuncParams{
				Type: "object",
				Properties: map[string]propDef{
					"path": {Type: "string", Description: "Relative file path"},
				},
				Required: []string{"path"},
			},
		},
	}}
}

// GenerateWithTools runs an iterative tool-calling loop.
func (c *Client) GenerateWithTools(ctx context.Context, system, user string, exec ToolExecutor, maxIter int, progress ProgressFn) (string, error) {
	if maxIter <= 0 {
		maxIter = 100
	}

	// First, get the changed files overview and include it in the user prompt
	changedFiles, err := exec.ChangedFiles()
	if err == nil && changedFiles != "" {
		user = changedFiles + "\n\n" + user
	}

	messages := []chatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}

	for step := 1; step <= maxIter; step++ {
		if progress != nil {
			progress(step, maxIter)
		}

		resp, err := c.chatComplete(ctx, messages, true)
		if err != nil {
			return "", err
		}

		choice := resp.Choices[0]
		msg := choice.Message

		// If the model returned content with no tool calls, it's the final answer
		if msg.Content != "" && len(msg.ToolCalls) == 0 {
			return msg.Content, nil
		}

		// Execute tool calls and append results
		if len(msg.ToolCalls) > 0 {
			messages = append(messages, msg)
			for _, tc := range msg.ToolCalls {
				result := execToolCall(tc, exec)
				messages = append(messages, chatMessage{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    result,
				})
			}
			continue
		}

		// Content is empty and no tool calls — probably length-limited
		return "", fmt.Errorf("LLM returned empty response (iteration %d, reason: %s)", step, choice.FinishReason)
	}

	return "", fmt.Errorf("exceeded max iterations (%d) without final output", maxIter)
}

func execToolCall(tc toolCall, exec ToolExecutor) string {
	var args struct {
		Path  string `json:"path"`
		Start int    `json:"start"`
		End   int    `json:"end"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return fmt.Sprintf("error: invalid arguments: %v", err)
	}

	switch tc.Function.Name {
	case "read_current":
		content, err := exec.ReadCurrentFile(args.Path, args.Start, args.End)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return content
	case "read_base":
		content, err := exec.ReadBaseFile(args.Path, args.Start, args.End)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return content
	case "show_diff":
		content, err := exec.ShowFileDiff(args.Path)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		if content == "" {
			return "(empty diff — file may be new or unchanged)"
		}
		return content
	default:
		return fmt.Sprintf("error: unknown tool %q", tc.Function.Name)
	}
}

// chatComplete sends a chat request and returns the parsed response.
func (c *Client) chatComplete(ctx context.Context, messages []chatMessage, withTools bool) (*chatResponse, error) {
	req := chatRequest{
		Model:     c.model,
		MaxTokens: 32768,
		Messages:  messages,
	}
	if withTools {
		req.Tools = toolDefinitions()
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("API returned no choices")
	}

	if chatResp.Usage != nil {
		// token info silently captured, not logged to avoid TUI corruption
	}

	return &chatResp, nil
}

// ============================================================
// Ping
// ============================================================

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.chatComplete(ctx, []chatMessage{
		{Role: "user", Content: "respond with exactly one word: ok"},
	}, false)
	return err
}
