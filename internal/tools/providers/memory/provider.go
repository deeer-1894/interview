package memory

import (
	"context"

	"mockinterview/internal/protocol"
)

type repository interface {
	Append(ctx context.Context, record protocol.MemoryRecord) error
	List(ctx context.Context, runID string) ([]protocol.MemoryRecord, error)
}

type Provider struct {
	repository repository
}

func New(repository repository) *Provider {
	return &Provider{repository: repository}
}

func (p *Provider) Append(ctx context.Context, record protocol.MemoryRecord) error {
	return p.repository.Append(ctx, record)
}

func (p *Provider) Load(ctx context.Context, runID string) ([]protocol.MemoryRecord, error) {
	return p.repository.List(ctx, runID)
}

func (p *Provider) List(ctx context.Context, runID string) ([]protocol.MemoryRecord, error) {
	return p.Load(ctx, runID)
}
