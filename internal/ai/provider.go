package ai

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	SystemPrompt = "You are a git automation bot. Analyze the provided code diff. Respond ONLY with a concise, Conventional Commit message (e.g., 'fix(ui): adjust button padding'). Do not add quotes or markdown."
)

// AIProvider defines the interface for AI commit message generation
type AIProvider interface {
	GenerateCommitMsg(diff string) (string, error)
}

// NewProvider creates an AI provider based on the provider name
func NewProvider(provider, apiKey, baseURL string) (AIProvider, error) {
	switch strings.ToLower(provider) {
	case "gemini":
		return NewGeminiProvider(apiKey), nil
	case "openai", "openrouter":
		if baseURL == "" && provider == "openai" {
			baseURL = "https://api.openai.com/v1"
		} else if baseURL == "" {
			baseURL = "https://openrouter.ai/api/v1"
		}
		return NewOpenAIProvider(apiKey, baseURL), nil
	case "anthropic", "claude":
		return NewAnthropicProvider(apiKey), nil
	default:
		return nil, fmt.Errorf("unknown AI provider: %s", provider)
	}
}

// BaseProvider provides common HTTP client functionality
type BaseProvider struct {
	client *http.Client
}

func NewBaseProvider() *BaseProvider {
	return &BaseProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (b *BaseProvider) doRequest(url string, headers map[string]string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}
	
	return respBody, nil
}

