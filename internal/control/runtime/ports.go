package runtime

import (
	"context"

	"mockinterview/internal/protocol"
)

type ToolGateway interface {
	ResolveSkill(ctx context.Context, cfg protocol.InterviewConfig) (protocol.SkillSpec, error)
	ResolveRubric(ctx context.Context, cfg protocol.InterviewConfig) (protocol.Rubric, error)
	AppendMemory(ctx context.Context, record protocol.MemoryRecord) error
	LoadMemory(ctx context.Context, runID string) ([]protocol.MemoryRecord, error)
	SaveCheckpoint(ctx context.Context, snapshot protocol.CheckpointSnapshot) error
	LoadCheckpoint(ctx context.Context, runID string) (protocol.CheckpointSnapshot, error)
	ListArtifacts(ctx context.Context, conversationID string) ([]protocol.Artifact, error)
	GetArtifact(ctx context.Context, id string) (protocol.Artifact, error)
	SearchWeb(ctx context.Context, query string, limit int) ([]protocol.WebSearchResult, error)
}

type ClarifyRepository interface {
	Save(ctx context.Context, request protocol.ClarifyRequest) error
	GetPending(ctx context.Context, runID string) (protocol.ClarifyRequest, error)
	Resolve(ctx context.Context, runID string) error
}
