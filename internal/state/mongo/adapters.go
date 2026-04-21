package mongo

import (
	"context"
	"fmt"

	"mockinterview/internal/protocol"
)

type ConversationRepository struct{ repositories *Repositories }
type TaskRepository struct{ repositories *Repositories }
type RunRepository struct{ repositories *Repositories }
type MessageRepository struct{ repositories *Repositories }
type EventRepository struct{ repositories *Repositories }
type ProfileRepository struct{ repositories *Repositories }
type CheckpointRepository struct{ repositories *Repositories }
type ClarifyRepository struct{ repositories *Repositories }
type MemoryRepository struct{ repositories *Repositories }
type ArtifactRepository struct{ repositories *Repositories }

func NewAdapters(repositories *Repositories) (ConversationRepository, TaskRepository, RunRepository, MessageRepository, EventRepository, ProfileRepository, CheckpointRepository, ClarifyRepository, MemoryRepository, ArtifactRepository) {
	return ConversationRepository{repositories: repositories},
		TaskRepository{repositories: repositories},
		RunRepository{repositories: repositories},
		MessageRepository{repositories: repositories},
		EventRepository{repositories: repositories},
		ProfileRepository{repositories: repositories},
		CheckpointRepository{repositories: repositories},
		ClarifyRepository{repositories: repositories},
		MemoryRepository{repositories: repositories},
		ArtifactRepository{repositories: repositories}
}

func (r ConversationRepository) Create(ctx context.Context, conversation protocol.Conversation) (protocol.Conversation, error) {
	return r.repositories.CreateConversation(ctx, conversation)
}

func (r ConversationRepository) Update(ctx context.Context, conversation protocol.Conversation) (protocol.Conversation, error) {
	return r.repositories.UpdateConversation(ctx, conversation)
}

func (r ConversationRepository) Get(ctx context.Context, id string) (protocol.Conversation, error) {
	return r.repositories.GetConversation(ctx, id)
}

func (r ConversationRepository) List(ctx context.Context) ([]protocol.Conversation, error) {
	return r.repositories.ListConversations(ctx)
}

func (r TaskRepository) Create(ctx context.Context, task protocol.Task) (protocol.Task, error) {
	return r.repositories.CreateTask(ctx, task)
}

func (r TaskRepository) Get(ctx context.Context, id string) (protocol.Task, error) {
	return r.repositories.GetTask(ctx, id)
}

func (r TaskRepository) ListByConversation(ctx context.Context, conversationID string) ([]protocol.Task, error) {
	return r.repositories.ListTasksByConversation(ctx, conversationID)
}

func (r RunRepository) Create(ctx context.Context, run protocol.Run) (protocol.Run, error) {
	return r.repositories.CreateRun(ctx, run)
}

func (r RunRepository) Update(ctx context.Context, run protocol.Run) (protocol.Run, error) {
	return r.repositories.UpdateRun(ctx, run)
}

func (r RunRepository) Get(ctx context.Context, id string) (protocol.Run, error) {
	return r.repositories.GetRun(ctx, id)
}

func (r RunRepository) ListByConversation(ctx context.Context, conversationID string) ([]protocol.Run, error) {
	return r.repositories.ListRunsByConversation(ctx, conversationID)
}

func (r MessageRepository) Create(ctx context.Context, message protocol.Message) (protocol.Message, error) {
	return r.repositories.CreateMessage(ctx, message)
}

func (r MessageRepository) ListByRun(ctx context.Context, runID string) ([]protocol.Message, error) {
	return r.repositories.ListMessagesByRun(ctx, runID)
}

func (r EventRepository) Create(ctx context.Context, event protocol.Event) (protocol.Event, error) {
	return r.repositories.CreateEvent(ctx, event)
}

func (r EventRepository) ListByRun(ctx context.Context, runID string) ([]protocol.Event, error) {
	return r.repositories.ListEventsByRun(ctx, runID)
}

func (r ProfileRepository) Get(ctx context.Context, id string) (protocol.CandidateProfile, error) {
	return r.repositories.GetProfile(ctx, id)
}

func (r ProfileRepository) Save(ctx context.Context, profile protocol.CandidateProfile) (protocol.CandidateProfile, error) {
	return r.repositories.SaveProfile(ctx, profile)
}

func (r CheckpointRepository) Save(ctx context.Context, snapshot protocol.CheckpointSnapshot) error {
	_ = ctx
	_ = snapshot
	return fmt.Errorf("checkpoint storage is not provided by mongo adapter")
}

func (r CheckpointRepository) Load(ctx context.Context, runID string) (protocol.CheckpointSnapshot, error) {
	_ = ctx
	return protocol.CheckpointSnapshot{}, fmt.Errorf("checkpoint storage is not provided by mongo adapter for run %s", runID)
}

func (r ClarifyRepository) Save(ctx context.Context, request protocol.ClarifyRequest) error {
	_ = ctx
	_ = request
	return fmt.Errorf("clarify storage is not provided by mongo adapter")
}

func (r ClarifyRepository) GetPending(ctx context.Context, runID string) (protocol.ClarifyRequest, error) {
	_ = ctx
	return protocol.ClarifyRequest{}, fmt.Errorf("clarify storage is not provided by mongo adapter for run %s", runID)
}

func (r ClarifyRepository) Resolve(ctx context.Context, runID string) error {
	_ = ctx
	return fmt.Errorf("clarify storage is not provided by mongo adapter for run %s", runID)
}

func (r MemoryRepository) Append(ctx context.Context, record protocol.MemoryRecord) error {
	return r.repositories.AppendMemory(ctx, record)
}

func (r MemoryRepository) List(ctx context.Context, runID string) ([]protocol.MemoryRecord, error) {
	return r.repositories.ListMemory(ctx, runID)
}

func (r ArtifactRepository) Create(ctx context.Context, artifact protocol.Artifact) (protocol.Artifact, error) {
	return r.repositories.CreateArtifact(ctx, artifact)
}

func (r ArtifactRepository) Update(ctx context.Context, artifact protocol.Artifact) (protocol.Artifact, error) {
	return r.repositories.UpdateArtifact(ctx, artifact)
}

func (r ArtifactRepository) Get(ctx context.Context, id string) (protocol.Artifact, error) {
	return r.repositories.GetArtifact(ctx, id)
}

func (r ArtifactRepository) ListByConversation(ctx context.Context, conversationID string) ([]protocol.Artifact, error) {
	return r.repositories.ListArtifactsByConversation(ctx, conversationID)
}

func (r ArtifactRepository) Delete(ctx context.Context, id string) error {
	return r.repositories.DeleteArtifact(ctx, id)
}
