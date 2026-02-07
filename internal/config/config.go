package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

const (
	DefaultCheckInterval = 10 * time.Minute
	ConfigFileName       = "config.json"
	DaemonFileName      = "daemon.json"
)

type Config struct {
	AIProvider   string `json:"ai_provider" mapstructure:"ai_provider"`     // "gemini", "openai", "anthropic", "openrouter"
	APIKey       string `json:"api_key" mapstructure:"api_key"`
	BaseURL      string `json:"base_url" mapstructure:"base_url"`           // For OpenRouter or custom OpenAI-compatible
	CheckIntervalMinutes int `json:"check_interval_minutes" mapstructure:"check_interval_minutes"`
	RootPath     string `json:"root_path" mapstructure:"root_path"`         // Git root path
}

type DaemonInfo struct {
	PID      int    `json:"pid"`
	RepoPath string `json:"repo_path"`
	Status   string `json:"status"` // "running", "error", "paused"
}

var configDir string

func init() {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to current directory
		configDir = "."
	} else {
		configDir = filepath.Join(userConfigDir, "autogit")
	}
	
	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create config directory: %v", err))
	}
}

func GetConfigDir() string {
	return configDir
}

func GetConfigPath() string {
	return filepath.Join(configDir, ConfigFileName)
}

func GetDaemonPath() string {
	return filepath.Join(configDir, DaemonFileName)
}

func LoadConfig() (*Config, error) {
	// Initialize viper
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(configDir)
	
	// Set defaults
	viper.SetDefault("ai_provider", "gemini")
	viper.SetDefault("check_interval_minutes", 10)
	viper.SetDefault("base_url", "")
	
	// Read from file if exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; create default
			cfg := &Config{
				AIProvider:          "gemini",
				CheckIntervalMinutes: 10,
			}
			if err := SaveConfig(cfg); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	
	// Also read from environment variables
	viper.SetEnvPrefix("AUTOGIT")
	viper.AutomaticEnv()
	
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	configPath := GetConfigPath()
	
	// Convert to JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	
	return nil
}

func LoadDaemonInfo() (*DaemonInfo, error) {
	daemonPath := GetDaemonPath()
	
	data, err := os.ReadFile(daemonPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No daemon running
		}
		return nil, fmt.Errorf("failed to read daemon info: %w", err)
	}
	
	var info DaemonInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal daemon info: %w", err)
	}
	
	return &info, nil
}

func SaveDaemonInfo(info *DaemonInfo) error {
	daemonPath := GetDaemonPath()
	
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal daemon info: %w", err)
	}
	
	if err := os.WriteFile(daemonPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write daemon info: %w", err)
	}
	
	return nil
}

func DeleteDaemonInfo() error {
	daemonPath := GetDaemonPath()
	return os.Remove(daemonPath)
}

func (c *Config) GetCheckInterval() time.Duration {
	if c.CheckIntervalMinutes <= 0 {
		return DefaultCheckInterval
	}
	return time.Duration(c.CheckIntervalMinutes) * time.Minute
}

