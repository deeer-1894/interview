package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"mockinterview/internal/protocol"
)

func TestUpdateRunSyncsConversationProjection(t *testing.T) {
	t.Parallel()

	now := time.Now()
	conversations := &memoryConversationRepo{
		items: map[string]protocol.Conversation{
			"conv_1": {
				ID:        "conv_1",
				Title:     "Projection Sync",
				Status:    "active",
				CreatedAt: now.Add(-time.Minute),
				UpdatedAt: now.Add(-time.Minute),
			},
		},
	}
	app := &App{
		conversations: conversations,
		runs: &memoryRunProjectionRepo{
			items: map[string]protocol.Run{
				"run_1": {
					ID:             "run_1",
					ConversationID: "conv_1",
					TaskID:         "task_1",
					Status:         protocol.RunCreated,
				},
			},
		},
	}

	err := app.UpdateRun(context.Background(), protocol.Run{
		ID:             "run_1",
		ConversationID: "conv_1",
		TaskID:         "task_1",
		Status:         protocol.RunWaitingClarify,
	})
	if err != nil {
		t.Fatalf("UpdateRun returned error: %v", err)
	}

	conversation, err := conversations.Get(context.Background(), "conv_1")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if conversation.LatestRunID != "run_1" {
		t.Fatalf("expected latest run id to be synced, got %#v", conversation)
	}
	if conversation.LatestRunStatus != protocol.RunWaitingClarify {
		t.Fatalf("expected latest run status to be synced, got %#v", conversation)
	}
}

type memoryConversationRepo struct {
	items map[string]protocol.Conversation
}

func (r *memoryConversationRepo) Create(ctx context.Context, conversation protocol.Conversation) (protocol.Conversation, error) {
	_ = ctx
	if r.items == nil {
		r.items = make(map[string]protocol.Conversation)
	}
	r.items[conversation.ID] = conversation
	return conversation, nil
}

func (r *memoryConversationRepo) Update(ctx context.Context, conversation protocol.Conversation) (protocol.Conversation, error) {
	_ = ctx
	if r.items == nil {
		r.items = make(map[string]protocol.Conversation)
	}
	r.items[conversation.ID] = conversation
	return conversation, nil
}

func (r *memoryConversationRepo) Get(ctx context.Context, id string) (protocol.Conversation, error) {
	_ = ctx
	conversation, ok := r.items[id]
	if !ok {
		return protocol.Conversation{}, fmt.Errorf("conversation %s not found", id)
	}
	return conversation, nil
}

func (r *memoryConversationRepo) List(ctx context.Context) ([]protocol.Conversation, error) {
	_ = ctx
	out := make([]protocol.Conversation, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, item)
	}
	return out, nil
}

type memoryRunProjectionRepo struct {
	items map[string]protocol.Run
}

func (r *memoryRunProjectionRepo) Create(ctx context.Context, run protocol.Run) (protocol.Run, error) {
	_ = ctx
	if r.items == nil {
		r.items = make(map[string]protocol.Run)
	}
	r.items[run.ID] = run
	return run, nil
}

func (r *memoryRunProjectionRepo) Update(ctx context.Context, run protocol.Run) (protocol.Run, error) {
	_ = ctx
	if r.items == nil {
		r.items = make(map[string]protocol.Run)
	}
	r.items[run.ID] = run
	return run, nil
}

func (r *memoryRunProjectionRepo) Get(ctx context.Context, id string) (protocol.Run, error) {
	_ = ctx
	run, ok := r.items[id]
	if !ok {
		return protocol.Run{}, fmt.Errorf("run %s not found", id)
	}
	return run, nil
}

func (r *memoryRunProjectionRepo) ListByConversation(ctx context.Context, conversationID string) ([]protocol.Run, error) {
	_ = ctx
	runs := make([]protocol.Run, 0, len(r.items))
	for _, run := range r.items {
		if run.ConversationID == conversationID {
			runs = append(runs, run)
		}
	}
	return runs, nil
}
