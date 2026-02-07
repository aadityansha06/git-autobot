package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

type OpenAIProvider struct {
	*BaseProvider
	apiKey string
	baseURL string
}

func NewOpenAIProvider(apiKey, baseURL string) *OpenAIProvider {
	return &OpenAIProvider{
		BaseProvider: NewBaseProvider(),
		apiKey:       apiKey,
		baseURL:      baseURL,
	}
}

type OpenAIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

func (o *OpenAIProvider) GenerateCommitMsg(diff string) (string, error) {
	if o.apiKey == "" {
		return "", fmt.Errorf("OpenAI API key is not set")
	}
	
	// Truncate diff if too long
	if len(diff) > 100000 {
		diff = diff[:100000] + "\n... (truncated)"
	}
	
	prompt := fmt.Sprintf("%s\n\nCode diff:\n%s", SystemPrompt, diff)
	
	url := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(o.baseURL, "/"))
	
	// Determine model based on base URL
	model := "gpt-3.5-turbo"
	if strings.Contains(o.baseURL, "openrouter") {
		model = "openai/gpt-3.5-turbo" // OpenRouter format
	}
	
	reqBody := OpenAIRequest{
		Model: model,
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
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", o.apiKey),
	}
	
	// OpenRouter requires additional header
	if strings.Contains(o.baseURL, "openrouter") {
		headers["HTTP-Referer"] = "https://github.com/aadityansha/autogit"
		headers["X-Title"] = "Autogit"
	}
	
	respBody, err := o.doRequest(url, headers, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", err
	}
	
	var resp OpenAIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI API")
	}
	
	message := strings.TrimSpace(resp.Choices[0].Message.Content)
	// Remove quotes if present
	message = strings.Trim(message, "\"'`")
	
	return message, nil
}

