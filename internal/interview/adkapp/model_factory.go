package adkapp

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/components/model"
	"google.golang.org/genai"

	"mockinterview/internal/interview"
)

func NewModel(ctx context.Context, cfg interview.ModelConfig) (model.ToolCallingChatModel, error) {
	cfg = cfg.WithDefaults()

	switch normalizeProvider(cfg.Provider) {
	case interview.ProviderOpenAI, interview.ProviderOpenAICompatible:
		return newOpenAIModel(ctx, cfg)
	case interview.ProviderClaude:
		return newClaudeModel(ctx, cfg)
	case interview.ProviderGemini:
		return newGeminiModel(ctx, cfg)
	case interview.ProviderDeepSeek:
		return newDeepSeekModel(ctx, cfg)
	case interview.ProviderOllama:
		return newOllamaModel(ctx, cfg)
	case interview.ProviderQwen:
		return newQwenModel(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported model provider %q", cfg.Provider)
	}
}

func newOpenAIModel(ctx context.Context, cfg interview.ModelConfig) (model.ToolCallingChatModel, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("api key is required for provider %q", cfg.Provider)
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("model is required for provider %q", cfg.Provider)
	}

	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		BaseURL: cfg.BaseURL,
		ByAzure: cfg.ByAzure,
		Timeout: cfg.Timeout,
	})
}

func newClaudeModel(ctx context.Context, cfg interview.ModelConfig) (model.ToolCallingChatModel, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("model is required for provider %q", cfg.Provider)
	}

	var baseURL *string
	if cfg.BaseURL != "" {
		baseURL = &cfg.BaseURL
	}

	return claude.NewChatModel(ctx, &claude.Config{
		APIKey:    cfg.APIKey,
		BaseURL:   baseURL,
		Model:     cfg.Model,
		MaxTokens: 4096,
		ByBedrock: cfg.ClaudeByBedrock,
		ByVertex:  cfg.ClaudeByVertex,
	})
}

func newGeminiModel(ctx context.Context, cfg interview.ModelConfig) (model.ToolCallingChatModel, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("api key is required for provider %q", cfg.Provider)
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("model is required for provider %q", cfg.Provider)
	}

	clientConfig := &genai.ClientConfig{
		APIKey: cfg.APIKey,
	}
	if cfg.BaseURL != "" {
		clientConfig.HTTPOptions = genai.HTTPOptions{
			BaseURL: cfg.BaseURL,
		}
	}

	client, err := genai.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("create gemini client: %w", err)
	}

	return gemini.NewChatModel(ctx, &gemini.Config{
		Client: client,
		Model:  cfg.Model,
	})
}

func newDeepSeekModel(ctx context.Context, cfg interview.ModelConfig) (model.ToolCallingChatModel, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("api key is required for provider %q", cfg.Provider)
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("model is required for provider %q", cfg.Provider)
	}

	return deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		BaseURL: cfg.BaseURL,
		Timeout: cfg.Timeout,
	})
}

func newOllamaModel(ctx context.Context, cfg interview.ModelConfig) (model.ToolCallingChatModel, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("model is required for provider %q", cfg.Provider)
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base url is required for provider %q", cfg.Provider)
	}

	return ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
		Timeout: cfg.Timeout,
	})
}

func newQwenModel(ctx context.Context, cfg interview.ModelConfig) (model.ToolCallingChatModel, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("api key is required for provider %q", cfg.Provider)
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("model is required for provider %q", cfg.Provider)
	}

	return qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
		Timeout: cfg.Timeout,
	})
}

func normalizeProvider(provider interview.ModelProvider) interview.ModelProvider {
	value := strings.TrimSpace(strings.ToLower(string(provider)))
	switch value {
	case "openai-compatible", "openai_compatible", "compatible":
		return interview.ProviderOpenAICompatible
	case "anthropic":
		return interview.ProviderClaude
	default:
		return interview.ModelProvider(value)
	}
}
