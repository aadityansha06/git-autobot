package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

type AnthropicProvider struct {
	*BaseProvider
	apiKey string
}

func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{
		BaseProvider: NewBaseProvider(),
		apiKey:       apiKey,
	}
}

type AnthropicRequest struct {
	Model     string   `json:"model"`
	MaxTokens int      `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

type AnthropicResponse struct {
	Content []AnthropicContent `json:"content"`
}

type AnthropicContent struct {
	Text string `json:"text"`
}

func (a *AnthropicProvider) GenerateCommitMsg(diff string) (string, error) {
	if a.apiKey == "" {
		return "", fmt.Errorf("Anthropic API key is not set")
	}
	
	// Truncate diff if too long
	if len(diff) > 100000 {
		diff = diff[:100000] + "\n... (truncated)"
	}
	
	prompt := fmt.Sprintf("%s\n\nCode diff:\n%s", SystemPrompt, diff)
	
	url := "https://api.anthropic.com/v1/messages"
	
	reqBody := AnthropicRequest{
		Model:     "claude-3-haiku-20240307",
		MaxTokens: 1024,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}
	
	headers := map[string]string{
		"Content-Type":      "application/json",
		"x-api-key":         a.apiKey,
		"anthropic-version": "2023-06-01",
	}
	
	respBody, err := a.doRequest(url, headers, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", err
	}
	
	var resp AnthropicResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	if len(resp.Content) == 0 {
		return "", fmt.Errorf("no response from Anthropic API")
	}
	
	message := strings.TrimSpace(resp.Content[0].Text)
	// Remove quotes if present
	message = strings.Trim(message, "\"'`")
	
	return message, nil
}

