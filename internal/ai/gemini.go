package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

type GeminiProvider struct {
	*BaseProvider
	apiKey string
}

func NewGeminiProvider(apiKey string) *GeminiProvider {
	return &GeminiProvider{
		BaseProvider: NewBaseProvider(),
		apiKey:       apiKey,
	}
}

type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

func (g *GeminiProvider) GenerateCommitMsg(diff string) (string, error) {
	if g.apiKey == "" {
		return "", fmt.Errorf("Gemini API key is not set")
	}
	
	// Truncate diff if too long (Gemini has token limits)
	if len(diff) > 100000 {
		diff = diff[:100000] + "\n... (truncated)"
	}
	
	prompt := fmt.Sprintf("%s\n\nCode diff:\n%s", SystemPrompt, diff)
	
	// Use gemini-1.5-flash as it's the current recommended model
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-3-flash-preview:generateContent?key=%s", g.apiKey)
	
	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}
	
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	
	respBody, err := g.doRequest(url, headers, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", err
	}
	
	var resp GeminiResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from Gemini API")
	}
	
	message := strings.TrimSpace(resp.Candidates[0].Content.Parts[0].Text)
	// Remove quotes if present
	message = strings.Trim(message, "\"'`")
	
	return message, nil
}

