package service

import (
	"context"

	"mockinterview/internal/protocol"
)

type ConversationRepository interface {
	Create(ctx context.Context, conversation protocol.Conversation) (protocol.Conversation, error)
	Update(ctx context.Context, conversation protocol.Conversation) (protocol.Conversation, error)
	Get(ctx context.Context, id string) (protocol.Conversation, error)
	List(ctx context.Context) ([]protocol.Conversation, error)
}

type TaskRepository interface {
	Create(ctx context.Context, task protocol.Task) (protocol.Task, error)
	Get(ctx context.Context, id string) (protocol.Task, error)
	ListByConversation(ctx context.Context, conversationID string) ([]protocol.Task, error)
}

type RunRepository interface {
	Create(ctx context.Context, run protocol.Run) (protocol.Run, error)
	Update(ctx context.Context, run protocol.Run) (protocol.Run, error)
	Get(ctx context.Context, id string) (protocol.Run, error)
	ListByConversation(ctx context.Context, conversationID string) ([]protocol.Run, error)
}

type MessageRepository interface {
	Create(ctx context.Context, message protocol.Message) (protocol.Message, error)
	ListByRun(ctx context.Context, runID string) ([]protocol.Message, error)
}

type EventRepository interface {
	Create(ctx context.Context, event protocol.Event) (protocol.Event, error)
	ListByRun(ctx context.Context, runID string) ([]protocol.Event, error)
}

type ProfileRepository interface {
	Get(ctx context.Context, id string) (protocol.CandidateProfile, error)
	Save(ctx context.Context, profile protocol.CandidateProfile) (protocol.CandidateProfile, error)
}
