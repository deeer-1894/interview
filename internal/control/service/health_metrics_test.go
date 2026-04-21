package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"mockinterview/internal/protocol"
)

func TestGetHealthMetricsAggregatesRuns(t *testing.T) {
	t.Parallel()

	now := time.Now()
	completedAt := now.Add(-2 * time.Minute)
	app := &App{
		conversations: stubConversationRepo{
			items: []protocol.Conversation{
				{ID: "conv_1"},
				{ID: "conv_deleted", Status: "deleted"},
			},
		},
		runs: stubRunRepo{
			byConversation: map[string][]protocol.Run{
				"conv_1": {
					{
						ID:          "run_ok",
						TaskID:      "task_1",
						Status:      protocol.RunCompleted,
						Input:       "请模拟一场 Go 面试",
						Output:      "这里是完整回答",
						CreatedAt:   now.Add(-5 * time.Minute),
						UpdatedAt:   completedAt,
						CompletedAt: &completedAt,
					},
					{
						ID:        "run_failed",
						TaskID:    "task_2",
						Status:    protocol.RunFailed,
						Input:     "继续",
						Output:    "",
						CreatedAt: now.Add(-4 * time.Minute),
						UpdatedAt: now.Add(-3 * time.Minute),
					},
					{
						ID:        "run_active",
						TaskID:    "task_3",
						Status:    protocol.RunRunning,
						Input:     "还在执行",
						CreatedAt: now.Add(-1 * time.Minute),
						UpdatedAt: now,
					},
					{
						ID:        "run_stale",
						TaskID:    "task_4",
						Status:    protocol.RunRunning,
						Input:     "状态脏数据",
						CreatedAt: now.Add(-90 * time.Second),
						UpdatedAt: now.Add(-30 * time.Second),
					},
				},
				"conv_deleted": {
					{
						ID:        "run_deleted",
						TaskID:    "task_5",
						Status:    protocol.RunRunning,
						Input:     "deleted conv stale run",
						CreatedAt: now.Add(-2 * time.Minute),
						UpdatedAt: now.Add(-1 * time.Minute),
					},
				},
			},
		},
		messages: stubMessageRepo{
			byRun: map[string][]protocol.Message{
				"run_ok": {
					{Content: "用户回答包括 goroutine、timeout、trace"},
					{Content: "面试官继续追问 tradeoff"},
				},
				"run_failed": {
					{Content: "这里也会算进 token 估算"},
				},
			},
		},
		activeRuns: NewActiveRuns(),
	}
	app.activeRuns.Set("run_active", func() {})
	defer app.activeRuns.Delete("run_active")

	metrics, err := app.GetHealthMetrics(context.Background())
	if err != nil {
		t.Fatalf("GetHealthMetrics returned error: %v", err)
	}
	if metrics.Status != "ok" {
		t.Fatalf("expected ok status, got %s", metrics.Status)
	}
	if metrics.RunCount != 4 {
		t.Fatalf("expected 4 visible runs, got %d", metrics.RunCount)
	}
	if metrics.VisibleConversationCount != 1 || metrics.DeletedConversationCount != 1 {
		t.Fatalf("unexpected conversation visibility counts: %#v", metrics)
	}
	if metrics.StoredRunCount != 5 {
		t.Fatalf("expected 5 stored runs, got %d", metrics.StoredRunCount)
	}
	if metrics.StoredTerminalRuns != 2 || metrics.StoredCompletedRuns != 1 || metrics.StoredFailedRuns != 1 {
		t.Fatalf("unexpected stored terminal counts: %#v", metrics)
	}
	if metrics.ActiveCount != 1 {
		t.Fatalf("expected 1 active run, got %d", metrics.ActiveCount)
	}
	if metrics.TerminalRuns != 2 {
		t.Fatalf("expected 2 terminal runs, got %d", metrics.TerminalRuns)
	}
	if metrics.CompletedRuns != 1 || metrics.FailedRuns != 1 {
		t.Fatalf("unexpected completed/failed counts: %#v", metrics)
	}
	if metrics.SuccessRate != 0.5 {
		t.Fatalf("expected success rate 0.5, got %v", metrics.SuccessRate)
	}
	if metrics.AverageDurationMs <= 0 {
		t.Fatalf("expected positive average duration, got %d", metrics.AverageDurationMs)
	}
	if metrics.TotalEstimatedTokens <= 0 || metrics.AverageEstimatedTokens <= 0 {
		t.Fatalf("expected token metrics to be populated, got %#v", metrics)
	}
}

type stubConversationRepo struct {
	items []protocol.Conversation
}

func (s stubConversationRepo) Create(ctx context.Context, conversation protocol.Conversation) (protocol.Conversation, error) {
	_ = ctx
	return conversation, nil
}

func (s stubConversationRepo) Update(ctx context.Context, conversation protocol.Conversation) (protocol.Conversation, error) {
	_ = ctx
	return conversation, nil
}

func (s stubConversationRepo) Get(ctx context.Context, id string) (protocol.Conversation, error) {
	_ = ctx
	for _, item := range s.items {
		if item.ID == id {
			return item, nil
		}
	}
	return protocol.Conversation{}, fmt.Errorf("conversation %s not found", id)
}

func (s stubConversationRepo) List(ctx context.Context) ([]protocol.Conversation, error) {
	_ = ctx
	return append([]protocol.Conversation(nil), s.items...), nil
}

type stubRunRepo struct {
	byConversation map[string][]protocol.Run
}

func (s stubRunRepo) Create(ctx context.Context, run protocol.Run) (protocol.Run, error) {
	_ = ctx
	return run, nil
}

func (s stubRunRepo) Update(ctx context.Context, run protocol.Run) (protocol.Run, error) {
	_ = ctx
	return run, nil
}

func (s stubRunRepo) Get(ctx context.Context, id string) (protocol.Run, error) {
	_ = ctx
	for _, runs := range s.byConversation {
		for _, run := range runs {
			if run.ID == id {
				return run, nil
			}
		}
	}
	return protocol.Run{}, fmt.Errorf("run %s not found", id)
}

func (s stubRunRepo) ListByConversation(ctx context.Context, conversationID string) ([]protocol.Run, error) {
	_ = ctx
	return append([]protocol.Run(nil), s.byConversation[conversationID]...), nil
}

type stubMessageRepo struct {
	byRun map[string][]protocol.Message
}

func (s stubMessageRepo) Create(ctx context.Context, message protocol.Message) (protocol.Message, error) {
	_ = ctx
	return message, nil
}

func (s stubMessageRepo) ListByRun(ctx context.Context, runID string) ([]protocol.Message, error) {
	_ = ctx
	return append([]protocol.Message(nil), s.byRun[runID]...), nil
}
