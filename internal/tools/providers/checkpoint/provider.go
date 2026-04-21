package checkpoint

import (
	"context"

	"mockinterview/internal/protocol"
)

type repository interface {
	Save(ctx context.Context, snapshot protocol.CheckpointSnapshot) error
	Load(ctx context.Context, runID string) (protocol.CheckpointSnapshot, error)
}

type Provider struct {
	repository repository
}

func New(repository repository) *Provider {
	return &Provider{repository: repository}
}

func (p *Provider) Save(ctx context.Context, snapshot protocol.CheckpointSnapshot) error {
	return p.repository.Save(ctx, snapshot)
}

func (p *Provider) Load(ctx context.Context, runID string) (protocol.CheckpointSnapshot, error) {
	return p.repository.Load(ctx, runID)
}
