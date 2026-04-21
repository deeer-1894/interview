package interview

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type ModelProvider string

const (
	ProviderOpenAI           ModelProvider = "openai"
	ProviderOpenAICompatible ModelProvider = "openai-compatible"
	ProviderClaude           ModelProvider = "claude"
	ProviderGemini           ModelProvider = "gemini"
	ProviderDeepSeek         ModelProvider = "deepseek"
	ProviderOllama           ModelProvider = "ollama"
	ProviderQwen             ModelProvider = "qwen"
)

type ModelConfig struct {
	Provider        ModelProvider
	Model           string
	APIKey          string
	BaseURL         string
	Timeout         time.Duration
	ByAzure         bool
	ClaudeByBedrock bool
	ClaudeByVertex  bool
}

func (c ModelConfig) WithDefaults() ModelConfig {
	if c.Provider == "" {
		c.Provider = ProviderOpenAI
	}
	if c.Timeout <= 0 {
		c.Timeout = 180 * time.Second
	}

	return c
}

func LoadModelConfigFromEnv() ModelConfig {
	cfg := ModelConfig{
		Provider:        ModelProvider(firstNonEmpty(os.Getenv("MODEL_PROVIDER"), os.Getenv("LLM_PROVIDER"), string(ProviderOpenAI))),
		Model:           firstNonEmpty(os.Getenv("MODEL_NAME"), os.Getenv("LLM_MODEL")),
		APIKey:          firstNonEmpty(os.Getenv("MODEL_API_KEY"), os.Getenv("LLM_API_KEY")),
		BaseURL:         firstNonEmpty(os.Getenv("MODEL_BASE_URL"), os.Getenv("LLM_BASE_URL")),
		ByAzure:         parseBoolEnv("OPENAI_BY_AZURE"),
		ClaudeByBedrock: parseBoolEnv("CLAUDE_BY_BEDROCK"),
		ClaudeByVertex:  parseBoolEnv("CLAUDE_BY_VERTEX"),
	}

	if timeoutStr := firstNonEmpty(os.Getenv("MODEL_TIMEOUT_SECONDS"), os.Getenv("LLM_TIMEOUT_SECONDS")); timeoutStr != "" {
		if seconds, err := strconv.Atoi(timeoutStr); err == nil && seconds > 0 {
			cfg.Timeout = time.Duration(seconds) * time.Second
		}
	}

	cfg = cfg.WithDefaults()

	switch cfg.Provider {
	case ProviderOpenAI, ProviderOpenAICompatible:
		cfg.APIKey = firstNonEmpty(cfg.APIKey, os.Getenv("OPENAI_API_KEY"))
		cfg.Model = firstNonEmpty(cfg.Model, os.Getenv("OPENAI_MODEL"))
		cfg.BaseURL = firstNonEmpty(cfg.BaseURL, os.Getenv("OPENAI_BASE_URL"))
	case ProviderClaude:
		cfg.APIKey = firstNonEmpty(cfg.APIKey, os.Getenv("CLAUDE_API_KEY"), os.Getenv("ANTHROPIC_API_KEY"))
		cfg.Model = firstNonEmpty(cfg.Model, os.Getenv("CLAUDE_MODEL"), os.Getenv("ANTHROPIC_MODEL"))
		cfg.BaseURL = firstNonEmpty(cfg.BaseURL, os.Getenv("CLAUDE_BASE_URL"))
	case ProviderGemini:
		cfg.APIKey = firstNonEmpty(cfg.APIKey, os.Getenv("GEMINI_API_KEY"))
		cfg.Model = firstNonEmpty(cfg.Model, os.Getenv("GEMINI_MODEL"))
		cfg.BaseURL = firstNonEmpty(cfg.BaseURL, os.Getenv("GEMINI_BASE_URL"))
	case ProviderDeepSeek:
		cfg.APIKey = firstNonEmpty(cfg.APIKey, os.Getenv("DEEPSEEK_API_KEY"))
		cfg.Model = firstNonEmpty(cfg.Model, os.Getenv("DEEPSEEK_MODEL"), os.Getenv("MODEL_NAME"))
		cfg.BaseURL = firstNonEmpty(cfg.BaseURL, os.Getenv("DEEPSEEK_BASE_URL"))
	case ProviderOllama:
		cfg.Model = firstNonEmpty(cfg.Model, os.Getenv("OLLAMA_MODEL"), os.Getenv("MODEL_NAME"))
		cfg.BaseURL = firstNonEmpty(cfg.BaseURL, os.Getenv("OLLAMA_BASE_URL"), "http://localhost:11434")
	case ProviderQwen:
		cfg.APIKey = firstNonEmpty(cfg.APIKey, os.Getenv("DASHSCOPE_API_KEY"), os.Getenv("QWEN_API_KEY"))
		cfg.Model = firstNonEmpty(cfg.Model, os.Getenv("QWEN_MODEL"), os.Getenv("MODEL_NAME"))
		cfg.BaseURL = firstNonEmpty(cfg.BaseURL, os.Getenv("QWEN_BASE_URL"), "https://dashscope.aliyuncs.com/compatible-mode/v1")
	}

	return cfg.WithDefaults()
}

func parseBoolEnv(key string) bool {
	v := strings.TrimSpace(os.Getenv(key))
	ok, err := strconv.ParseBool(v)
	return err == nil && ok
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}
