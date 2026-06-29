package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// ProviderConfig holds settings for a single AI provider.
type ProviderConfig struct {
	APIKey  string `yaml:"api_key" json:"api_key"`
	BaseURL string `yaml:"base_url" json:"base_url"`
	Model   string `yaml:"model" json:"model"`
}

// Config is the top-level configuration for Tynex.
type Config struct {
	DefaultProvider string                    `yaml:"default_provider" json:"default_provider"`
	Model           string                    `yaml:"model" json:"model"`
	MaxTokens       int                       `yaml:"max_tokens" json:"max_tokens"`
	Temperature     float64                   `yaml:"temperature" json:"temperature"`
	SystemPrompt    string                    `yaml:"system_prompt" json:"system_prompt"`
	Providers       map[string]ProviderConfig `yaml:"providers" json:"providers"`
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		DefaultProvider: "openai",
		Model:           "gpt-4o",
		MaxTokens:       4096,
		Temperature:     0.7,
		SystemPrompt:    "You are Tynex, an AI-powered CLI coding assistant. Help the user with coding tasks, file operations, and shell commands.",
		Providers: map[string]ProviderConfig{
			"openai": {
				BaseURL: "https://api.openai.com/v1",
				Model:   "gpt-4o",
			},
			"anthropic": {
				BaseURL: "https://api.anthropic.com/v1",
				Model:   "claude-sonnet-4-20250514",
			},
			"deepseek": {
				BaseURL: "https://api.deepseek.com/v1",
				Model:   "deepseek-chat",
			},
			"groq": {
				BaseURL: "https://api.groq.com/openai/v1",
				Model:   "llama-3.1-70b-versatile",
			},
		},
	}
}

// configDir returns the Tynex config directory (~/.config/tynex).
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home directory not found: %w", err)
	}
	dir := filepath.Join(home, ".config", "tynex")
	return dir, nil
}

// ConfigPath returns the path to the YAML config file.
func ConfigPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Load reads configuration from config.yaml, .env, and environment variables.
// Environment variables take precedence over .env, which takes precedence over config.yaml.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// 1. Load from YAML
	if err := cfg.loadYAML(); err != nil {
		// It's okay if the file doesn't exist yet
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("loading config.yaml: %w", err)
		}
	}

	// 2. Load from .env
	cfg.loadDotEnv()

	// 3. Environment variables override everything
	cfg.loadEnvVars()

	return cfg, nil
}

// loadYAML reads config from the YAML file.
func (c *Config) loadYAML() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, c)
}

// loadDotEnv reads .env files for environment variable overrides.
func (c *Config) loadDotEnv() {
	// Try current directory .env first, then config directory
	dir, _ := configDir()

	_ = godotenv.Load()                          // ./ .env
	_ = godotenv.Load(filepath.Join(dir, ".env")) // ~/.config/tynex/.env
}

// loadEnvVars reads configuration from environment variables.
func (c *Config) loadEnvVars() {
	if v := os.Getenv("TYNEX_DEFAULT_PROVIDER"); v != "" {
		c.DefaultProvider = v
	}
	if v := os.Getenv("TYNEX_MODEL"); v != "" {
		c.Model = v
	}
	if v := os.Getenv("TYNEX_MAX_TOKENS"); v != "" {
		fmt.Sscanf(v, "%d", &c.MaxTokens)
	}
	if v := os.Getenv("TYNEX_TEMPERATURE"); v != "" {
		fmt.Sscanf(v, "%f", &c.Temperature)
	}
	if v := os.Getenv("TYNEX_SYSTEM_PROMPT"); v != "" {
		c.SystemPrompt = v
	}

	// Provider-specific env vars
	for name := range c.Providers {
		keyEnv := fmt.Sprintf("TYNEX_%s_API_KEY", envKey(name))
		urlEnv := fmt.Sprintf("TYNEX_%s_BASE_URL", envKey(name))
		modelEnv := fmt.Sprintf("TYNEX_%s_MODEL", envKey(name))

		p := c.Providers[name]
		if v := os.Getenv(keyEnv); v != "" {
			p.APIKey = v
		}
		if v := os.Getenv(urlEnv); v != "" {
			p.BaseURL = v
		}
		if v := os.Getenv(modelEnv); v != "" {
			p.Model = v
		}
		c.Providers[name] = p
	}
}

// Save writes the current configuration to the YAML config file.
func (c *Config) Save() error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	path := filepath.Join(dir, "config.yaml")
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// APIKey returns the API key for the given provider name.
// It checks env vars first, then config file.
func (c *Config) APIKey(providerName string) string {
	// Check env var first
	envKey := fmt.Sprintf("TYNEX_%s_API_KEY", envKey(providerName))
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	if p, ok := c.Providers[providerName]; ok {
		return p.APIKey
	}
	return ""
}

// envKey converts a provider name to an environment variable key segment.
// e.g. "openai" -> "OPENAI"
func envKey(name string) string {
	return strings.ToUpper(name)
}

// String returns a human-readable representation of the config.
func (c *Config) String() string {
	result := fmt.Sprintf("Default Provider: %s\n", c.DefaultProvider)
	result += fmt.Sprintf("Model: %s\n", c.Model)
	result += fmt.Sprintf("Max Tokens: %d\n", c.MaxTokens)
	result += fmt.Sprintf("Temperature: %.1f\n", c.Temperature)
	result += "\nProviders:\n"
	for name, p := range c.Providers {
		keyDisplay := "[not set]"
		if p.APIKey != "" {
			keyDisplay = p.APIKey[:8] + "..."
		}
		result += fmt.Sprintf("  %s:\n", name)
		result += fmt.Sprintf("    API Key: %s\n", keyDisplay)
		result += fmt.Sprintf("    Base URL: %s\n", p.BaseURL)
		result += fmt.Sprintf("    Model: %s\n", p.Model)
	}
	return result
}
