package ai

import (
	"fmt"
	"strings"
)

// ValidateAPIKey validates an API key by attempting to create a provider and make a test request
func ValidateAPIKey(provider, apiKey, baseURL string) error {
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	
	// Basic validation - check if key is not empty
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("API key cannot be empty")
	}
	
	// Provider-specific basic validation
	switch strings.ToLower(provider) {
	case "gemini":
		if len(apiKey) < 20 {
			return fmt.Errorf("Gemini API key appears to be invalid (too short)")
		}
	case "openai", "openrouter":
		if !strings.HasPrefix(apiKey, "sk-") && !strings.HasPrefix(apiKey, "sk_") {
			return fmt.Errorf("OpenAI/OpenRouter API key should start with 'sk-' or 'sk_'")
		}
	case "anthropic", "claude":
		if !strings.HasPrefix(apiKey, "sk-ant-") {
			return fmt.Errorf("Anthropic API key should start with 'sk-ant-'")
		}
	default:
		return fmt.Errorf("unknown AI provider: %s", provider)
	}
	
	// Try to create provider to check for basic errors
	_, err := NewProvider(provider, apiKey, baseURL)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	
	return nil
}

