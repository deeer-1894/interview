package gateway

import (
	"context"

	"mockinterview/internal/protocol"
	"mockinterview/internal/tools/registry"
)

type Gateway struct {
	skills      registry.SkillResolver
	rubrics     registry.RubricResolver
	checkpoints registry.CheckpointStore
	memories    registry.MemoryStore
	artifacts   registry.ArtifactStore
	webSearch   registry.WebSearchProvider
}

func New(reg *registry.Registry) (*Gateway, error) {
	skills, err := reg.SkillResolver()
	if err != nil {
		return nil, err
	}
	rubrics, err := reg.RubricResolver()
	if err != nil {
		return nil, err
	}
	checkpoints, err := reg.CheckpointStore()
	if err != nil {
		return nil, err
	}
	memories, err := reg.MemoryStore()
	if err != nil {
		return nil, err
	}
	artifacts, err := reg.ArtifactStore()
	if err != nil {
		return nil, err
	}
	webSearch, err := reg.WebSearch()
	if err != nil {
		return nil, err
	}
	return &Gateway{
		skills:      skills,
		rubrics:     rubrics,
		checkpoints: checkpoints,
		memories:    memories,
		artifacts:   artifacts,
		webSearch:   webSearch,
	}, nil
}

func (g *Gateway) ResolveSkill(ctx context.Context, cfg protocol.InterviewConfig) (protocol.SkillSpec, error) {
	return g.skills.Resolve(ctx, cfg)
}

func (g *Gateway) ResolveRubric(ctx context.Context, cfg protocol.InterviewConfig) (protocol.Rubric, error) {
	return g.rubrics.Resolve(ctx, cfg)
}

func (g *Gateway) AppendMemory(ctx context.Context, record protocol.MemoryRecord) error {
	return g.memories.Append(ctx, record)
}

func (g *Gateway) LoadMemory(ctx context.Context, runID string) ([]protocol.MemoryRecord, error) {
	return g.memories.List(ctx, runID)
}

func (g *Gateway) SaveCheckpoint(ctx context.Context, snapshot protocol.CheckpointSnapshot) error {
	return g.checkpoints.Save(ctx, snapshot)
}

func (g *Gateway) LoadCheckpoint(ctx context.Context, runID string) (protocol.CheckpointSnapshot, error) {
	return g.checkpoints.Load(ctx, runID)
}

func (g *Gateway) ListArtifacts(ctx context.Context, conversationID string) ([]protocol.Artifact, error) {
	return g.artifacts.ListByConversation(ctx, conversationID)
}

func (g *Gateway) GetArtifact(ctx context.Context, id string) (protocol.Artifact, error) {
	return g.artifacts.Get(ctx, id)
}

func (g *Gateway) SearchWeb(ctx context.Context, query string, limit int) ([]protocol.WebSearchResult, error) {
	return g.webSearch.Search(ctx, query, limit)
}
