package web

import (
	"strings"

	"mockinterview/internal/protocol"
)

// requestHelper provides utility methods for HTTP request handling
type requestHelper struct{}

// parseModelConfig extracts and normalizes model configuration from client request
func (requestHelper) parseModelConfig(defaultCfg protocol.ModelConfig, clientCfg *ClientModelConfig) protocol.ModelConfig {
	cfg := defaultCfg

	if clientCfg == nil {
		return cfg
	}

	if provider := strings.TrimSpace(clientCfg.Provider); provider != "" {
		cfg.Provider = provider
	}
	if model := strings.TrimSpace(clientCfg.Model); model != "" {
		cfg.Model = model
	}
	if apiKey := strings.TrimSpace(clientCfg.APIKey); apiKey != "" {
		cfg.APIKey = apiKey
	}
	if baseURL := strings.TrimSpace(clientCfg.BaseURL); baseURL != "" {
		cfg.BaseURL = baseURL
	}

	return cfg
}
