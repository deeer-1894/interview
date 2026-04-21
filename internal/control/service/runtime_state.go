package service

import (
	"context"

	"mockinterview/internal/protocol"
)

type CheckpointRepository interface {
	Save(ctx context.Context, snapshot protocol.CheckpointSnapshot) error
	Load(ctx context.Context, runID string) (protocol.CheckpointSnapshot, error)
}

type ClarifyRequestRepository interface {
	Save(ctx context.Context, request protocol.ClarifyRequest) error
	GetPending(ctx context.Context, runID string) (protocol.ClarifyRequest, error)
	Resolve(ctx context.Context, runID string) error
}

type MemoryRepository interface {
	Append(ctx context.Context, record protocol.MemoryRecord) error
	List(ctx context.Context, runID string) ([]protocol.MemoryRecord, error)
}

type ArtifactRepository interface {
	Create(ctx context.Context, artifact protocol.Artifact) (protocol.Artifact, error)
	Update(ctx context.Context, artifact protocol.Artifact) (protocol.Artifact, error)
	Get(ctx context.Context, id string) (protocol.Artifact, error)
	ListByConversation(ctx context.Context, conversationID string) ([]protocol.Artifact, error)
	Delete(ctx context.Context, id string) error
}
