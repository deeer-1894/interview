package remote

import (
	"context"
	"fmt"
	"strings"

	"mockinterview/internal/protocol"
	"mockinterview/internal/tools/mcp"
)

type Gateway struct {
	client *mcp.Client
	group  string
}

func New(cfg Config) (*Gateway, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("remote provider is disabled")
	}
	transport := strings.TrimSpace(cfg.Transport)
	if transport == "" || transport == string(mcp.TransportHTTP) {
		httpTransport, err := mcp.NewHTTPTransport(mcp.HTTPTransportConfig{
			BaseURL:   cfg.BaseURL,
			Token:     cfg.Token,
			Group:     cfg.Group,
			Timeout:   cfg.Timeout,
			ListPath:  cfg.ListPath,
			CallPath:  cfg.CallPath,
			UserAgent: "mockinterview-mcp-client/1.0",
		})
		if err != nil {
			return nil, err
		}
		return &Gateway{
			client: mcp.NewClient(httpTransport),
			group:  cfg.Group,
		}, nil
	}
	return nil, fmt.Errorf("unsupported mcp transport %q", transport)
}

type SkillResolver struct{ gateway *Gateway }
type RubricResolver struct{ gateway *Gateway }
type MemoryStore struct{ gateway *Gateway }
type CheckpointStore struct{ gateway *Gateway }
type ArtifactStore struct{ gateway *Gateway }
type WebSearchProvider struct{ gateway *Gateway }

func NewSkillResolver(gateway *Gateway) *SkillResolver {
	return &SkillResolver{gateway: gateway}
}

func NewRubricResolver(gateway *Gateway) *RubricResolver {
	return &RubricResolver{gateway: gateway}
}

func NewMemoryStore(gateway *Gateway) *MemoryStore {
	return &MemoryStore{gateway: gateway}
}

func NewCheckpointStore(gateway *Gateway) *CheckpointStore {
	return &CheckpointStore{gateway: gateway}
}

func NewArtifactStore(gateway *Gateway) *ArtifactStore {
	return &ArtifactStore{gateway: gateway}
}

func NewWebSearchProvider(gateway *Gateway) *WebSearchProvider {
	return &WebSearchProvider{gateway: gateway}
}

func (p *SkillResolver) Resolve(ctx context.Context, cfg protocol.InterviewConfig) (protocol.SkillSpec, error) {
	var out protocol.SkillSpec
	err := p.gateway.client.Call(ctx, "skill.resolve", map[string]any{"config": cfg}, &out, p.gateway.group)
	return out, err
}

func (p *RubricResolver) Resolve(ctx context.Context, cfg protocol.InterviewConfig) (protocol.Rubric, error) {
	var out protocol.Rubric
	err := p.gateway.client.Call(ctx, "rubric.resolve", map[string]any{"config": cfg}, &out, p.gateway.group)
	return out, err
}

func (p *MemoryStore) Append(ctx context.Context, record protocol.MemoryRecord) error {
	return p.gateway.client.Call(ctx, "memory.append", map[string]any{"record": record}, nil, p.gateway.group)
}

func (p *MemoryStore) List(ctx context.Context, runID string) ([]protocol.MemoryRecord, error) {
	var out []protocol.MemoryRecord
	err := p.gateway.client.Call(ctx, "memory.get", map[string]any{"runId": runID}, &out, p.gateway.group)
	return out, err
}

func (p *CheckpointStore) Save(ctx context.Context, snapshot protocol.CheckpointSnapshot) error {
	return p.gateway.client.Call(ctx, "checkpoint.save", map[string]any{"snapshot": snapshot}, nil, p.gateway.group)
}

func (p *CheckpointStore) Load(ctx context.Context, runID string) (protocol.CheckpointSnapshot, error) {
	var out protocol.CheckpointSnapshot
	err := p.gateway.client.Call(ctx, "checkpoint.load", map[string]any{"runId": runID}, &out, p.gateway.group)
	return out, err
}

func (p *ArtifactStore) Get(ctx context.Context, id string) (protocol.Artifact, error) {
	var out protocol.Artifact
	err := p.gateway.client.Call(ctx, "artifact.get", map[string]any{"id": id}, &out, p.gateway.group)
	return out, err
}

func (p *ArtifactStore) ListByConversation(ctx context.Context, conversationID string) ([]protocol.Artifact, error) {
	var out []protocol.Artifact
	err := p.gateway.client.Call(ctx, "artifact.list", map[string]any{"conversationId": conversationID}, &out, p.gateway.group)
	return out, err
}

func (p *WebSearchProvider) Search(ctx context.Context, query string, limit int) ([]protocol.WebSearchResult, error) {
	var out []protocol.WebSearchResult
	err := p.gateway.client.Call(ctx, "web.search", map[string]any{
		"query": query,
		"limit": limit,
	}, &out, p.gateway.group)
	return out, err
}
