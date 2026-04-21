package artifact

import (
	"context"

	"mockinterview/internal/protocol"
)

type repository interface {
	Get(ctx context.Context, id string) (protocol.Artifact, error)
	ListByConversation(ctx context.Context, conversationID string) ([]protocol.Artifact, error)
}

type Provider struct {
	repository repository
}

func New(repository repository) *Provider {
	return &Provider{repository: repository}
}

func (p *Provider) Get(ctx context.Context, id string) (protocol.Artifact, error) {
	return p.repository.Get(ctx, id)
}

func (p *Provider) ListByConversation(ctx context.Context, conversationID string) ([]protocol.Artifact, error) {
	return p.repository.ListByConversation(ctx, conversationID)
}
