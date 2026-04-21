package registry

import (
	"context"
	"fmt"

	"mockinterview/internal/protocol"
)

type SkillResolver interface {
	Resolve(ctx context.Context, cfg protocol.InterviewConfig) (protocol.SkillSpec, error)
}

type RubricResolver interface {
	Resolve(ctx context.Context, cfg protocol.InterviewConfig) (protocol.Rubric, error)
}

type CheckpointStore interface {
	Save(ctx context.Context, snapshot protocol.CheckpointSnapshot) error
	Load(ctx context.Context, runID string) (protocol.CheckpointSnapshot, error)
}

type MemoryStore interface {
	Append(ctx context.Context, record protocol.MemoryRecord) error
	List(ctx context.Context, runID string) ([]protocol.MemoryRecord, error)
}

type ArtifactStore interface {
	Get(ctx context.Context, id string) (protocol.Artifact, error)
	ListByConversation(ctx context.Context, conversationID string) ([]protocol.Artifact, error)
}

type WebSearchProvider interface {
	Search(ctx context.Context, query string, limit int) ([]protocol.WebSearchResult, error)
}

type Registry struct {
	skillResolver   SkillResolver
	rubricResolver  RubricResolver
	checkpointStore CheckpointStore
	memoryStore     MemoryStore
	artifactStore   ArtifactStore
	webSearch       WebSearchProvider
}

func New() *Registry {
	return &Registry{}
}

func (r *Registry) RegisterSkillResolver(resolver SkillResolver) {
	r.skillResolver = resolver
}

func (r *Registry) RegisterRubricResolver(resolver RubricResolver) {
	r.rubricResolver = resolver
}

func (r *Registry) RegisterCheckpointStore(store CheckpointStore) {
	r.checkpointStore = store
}

func (r *Registry) RegisterMemoryStore(store MemoryStore) {
	r.memoryStore = store
}

func (r *Registry) RegisterArtifactStore(store ArtifactStore) {
	r.artifactStore = store
}

func (r *Registry) RegisterWebSearch(provider WebSearchProvider) {
	r.webSearch = provider
}

func (r *Registry) SkillResolver() (SkillResolver, error) {
	if r.skillResolver == nil {
		return nil, fmt.Errorf("skill resolver is not registered")
	}
	return r.skillResolver, nil
}

func (r *Registry) RubricResolver() (RubricResolver, error) {
	if r.rubricResolver == nil {
		return nil, fmt.Errorf("rubric resolver is not registered")
	}
	return r.rubricResolver, nil
}

func (r *Registry) CheckpointStore() (CheckpointStore, error) {
	if r.checkpointStore == nil {
		return nil, fmt.Errorf("checkpoint store is not registered")
	}
	return r.checkpointStore, nil
}

func (r *Registry) MemoryStore() (MemoryStore, error) {
	if r.memoryStore == nil {
		return nil, fmt.Errorf("memory store is not registered")
	}
	return r.memoryStore, nil
}

func (r *Registry) ArtifactStore() (ArtifactStore, error) {
	if r.artifactStore == nil {
		return nil, fmt.Errorf("artifact store is not registered")
	}
	return r.artifactStore, nil
}

func (r *Registry) WebSearch() (WebSearchProvider, error) {
	if r.webSearch == nil {
		return nil, fmt.Errorf("web search provider is not registered")
	}
	return r.webSearch, nil
}
