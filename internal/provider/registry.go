package provider

import (
	"fmt"
	"strings"
)

// ProviderRegistry manages available AI providers.
type ProviderRegistry struct {
	providers map[string]func(apiKey, baseURL, model string) Provider
}

// NewRegistry creates a new provider registry with default providers.
func NewRegistry() *ProviderRegistry {
	r := &ProviderRegistry{
		providers: make(map[string]func(apiKey, baseURL, model string) Provider),
	}
	r.Register("openai", func(apiKey, baseURL, model string) Provider {
		return NewOpenAIProvider(apiKey, baseURL, model)
	})
	r.Register("deepseek", func(apiKey, baseURL, model string) Provider {
		return NewOpenAIProvider(apiKey, baseURL, model)
	})
	r.Register("groq", func(apiKey, baseURL, model string) Provider {
		return NewOpenAIProvider(apiKey, baseURL, model)
	})
	r.Register("together", func(apiKey, baseURL, model string) Provider {
		return NewOpenAIProvider(apiKey, baseURL, model)
	})
	r.Register("anthropic", func(apiKey, baseURL, model string) Provider {
		return NewAnthropicProvider(apiKey, baseURL, model)
	})
	return r
}

// Register adds a new provider constructor to the registry.
func (r *ProviderRegistry) Register(name string, constructor func(apiKey, baseURL, model string) Provider) {
	r.providers[strings.ToLower(name)] = constructor
}

// Get creates a provider instance by name.
func (r *ProviderRegistry) Get(name, apiKey, baseURL, model string) (Provider, error) {
	constructor, ok := r.providers[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s (available: %s)", name, r.List())
	}
	return constructor(apiKey, baseURL, model), nil
}

// List returns a comma-separated list of registered provider names.
func (r *ProviderRegistry) List() string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

// AvailableProviders returns all registered provider names.
func (r *ProviderRegistry) AvailableProviders() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}
