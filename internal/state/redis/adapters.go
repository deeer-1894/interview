package redis

import (
	"context"

	"mockinterview/internal/protocol"
)

type CheckpointRepository struct{ repositories *Repositories }
type ClarifyRepository struct{ repositories *Repositories }

func NewAdapters(repositories *Repositories) (CheckpointRepository, ClarifyRepository) {
	return CheckpointRepository{repositories: repositories}, ClarifyRepository{repositories: repositories}
}

func (r CheckpointRepository) Save(ctx context.Context, snapshot protocol.CheckpointSnapshot) error {
	return r.repositories.Save(ctx, snapshot)
}

func (r CheckpointRepository) Load(ctx context.Context, runID string) (protocol.CheckpointSnapshot, error) {
	return r.repositories.Load(ctx, runID)
}

func (r ClarifyRepository) Save(ctx context.Context, request protocol.ClarifyRequest) error {
	return r.repositories.SaveClarify(ctx, request)
}

func (r ClarifyRepository) GetPending(ctx context.Context, runID string) (protocol.ClarifyRequest, error) {
	return r.repositories.GetPendingClarify(ctx, runID)
}

func (r ClarifyRepository) Resolve(ctx context.Context, runID string) error {
	return r.repositories.ResolveClarify(ctx, runID)
}
