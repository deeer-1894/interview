package web

import (
	"testing"

	"mockinterview/internal/protocol"
)

func TestSanitizeTaskForClientClearsAPIKey(t *testing.T) {
	task := protocol.Task{
		ID: "task-1",
		ModelConfig: protocol.ModelConfig{
			Provider: "openai-compatible",
			Model:    "glm-4.6v",
			APIKey:   "secret-key",
			BaseURL:  "https://example.com/v1",
		},
	}

	sanitized := sanitizeTaskForClient(task)

	if sanitized.ModelConfig.APIKey != "" {
		t.Fatalf("expected API key to be cleared, got %q", sanitized.ModelConfig.APIKey)
	}
	if sanitized.ModelConfig.Provider != task.ModelConfig.Provider {
		t.Fatalf("expected provider to be preserved, got %q", sanitized.ModelConfig.Provider)
	}
	if sanitized.ModelConfig.Model != task.ModelConfig.Model {
		t.Fatalf("expected model to be preserved, got %q", sanitized.ModelConfig.Model)
	}
	if sanitized.ModelConfig.BaseURL != task.ModelConfig.BaseURL {
		t.Fatalf("expected base URL to be preserved, got %q", sanitized.ModelConfig.BaseURL)
	}
}

func TestSanitizeTasksForClientPreservesLength(t *testing.T) {
	tasks := []protocol.Task{
		{ID: "task-1", ModelConfig: protocol.ModelConfig{APIKey: "secret-1"}},
		{ID: "task-2", ModelConfig: protocol.ModelConfig{APIKey: "secret-2"}},
	}

	sanitized := sanitizeTasksForClient(tasks)

	if len(sanitized) != len(tasks) {
		t.Fatalf("expected %d tasks, got %d", len(tasks), len(sanitized))
	}
	for index := range sanitized {
		if sanitized[index].ModelConfig.APIKey != "" {
			t.Fatalf("expected task %d API key to be cleared, got %q", index, sanitized[index].ModelConfig.APIKey)
		}
	}
}
