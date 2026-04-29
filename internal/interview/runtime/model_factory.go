package runtime

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	deepseekmodel "github.com/cloudwego/eino-ext/components/model/deepseek"
	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

type EnvModelConfig struct {
	Provider string
	Name     string
	APIKey   string
	BaseURL  string
	Timeout  time.Duration
}

func NewModelFromEnv(ctx context.Context) (model.BaseChatModel, EnvModelConfig, error) {
	cfg := EnvModelConfig{
		Provider: strings.TrimSpace(os.Getenv("MODEL_PROVIDER")),
		Name:     strings.TrimSpace(os.Getenv("MODEL_NAME")),
		APIKey:   strings.TrimSpace(firstNonEmpty(os.Getenv("MODEL_API_KEY"), os.Getenv("LLM_API_KEY"))),
		BaseURL:  strings.TrimSpace(os.Getenv("MODEL_BASE_URL")),
		Timeout:  envDurationSeconds("MODEL_TIMEOUT_SECONDS", 180*time.Second),
	}
	if cfg.Provider == "" {
		cfg.Provider = "openai-compatible"
	}
	if cfg.Name == "" || cfg.APIKey == "" {
		return nil, cfg, fmt.Errorf("MODEL_NAME and MODEL_API_KEY are required")
	}

	switch strings.ToLower(cfg.Provider) {
	case "deepseek":
		chatModel, err := deepseekmodel.NewChatModel(ctx, &deepseekmodel.ChatModelConfig{
			APIKey:      cfg.APIKey,
			Model:       cfg.Name,
			BaseURL:     cfg.BaseURL,
			Timeout:     cfg.Timeout,
			Temperature: 0.75,
		})
		return chatModel, cfg, err
	case "openai", "openai-compatible":
		temperature := float32(0.75)
		chatModel, err := openaimodel.NewChatModel(ctx, &openaimodel.ChatModelConfig{
			APIKey:      cfg.APIKey,
			Model:       cfg.Name,
			BaseURL:     cfg.BaseURL,
			Timeout:     cfg.Timeout,
			Temperature: &temperature,
		})
		return chatModel, cfg, err
	default:
		return nil, cfg, fmt.Errorf("unsupported MODEL_PROVIDER %q", cfg.Provider)
	}
}

func envDurationSeconds(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
