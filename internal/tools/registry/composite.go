package registry

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"mockinterview/internal/protocol"
)

func ChainSkillResolvers(resolvers ...SkillResolver) SkillResolver {
	return skillResolverChain{resolvers: resolvers}
}

func ChainRubricResolvers(resolvers ...RubricResolver) RubricResolver {
	return rubricResolverChain{resolvers: resolvers}
}

func ChainCheckpointStores(stores ...CheckpointStore) CheckpointStore {
	return checkpointStoreChain{stores: stores}
}

func ChainMemoryStores(stores ...MemoryStore) MemoryStore {
	return memoryStoreChain{stores: stores}
}

func ChainArtifactStores(stores ...ArtifactStore) ArtifactStore {
	return artifactStoreChain{stores: stores}
}

func ChainWebSearchProviders(providers ...WebSearchProvider) WebSearchProvider {
	return webSearchProviderChain{providers: providers}
}

type skillResolverChain struct {
	resolvers []SkillResolver
}

func (c skillResolverChain) Resolve(ctx context.Context, cfg protocol.InterviewConfig) (protocol.SkillSpec, error) {
	var errs []error
	for _, resolver := range c.resolvers {
		if resolver == nil {
			continue
		}
		value, err := resolver.Resolve(ctx, cfg)
		if err == nil {
			return value, nil
		}
		errs = append(errs, err)
	}
	return protocol.SkillSpec{}, joinErrors("skill resolver chain failed", errs)
}

type rubricResolverChain struct {
	resolvers []RubricResolver
}

func (c rubricResolverChain) Resolve(ctx context.Context, cfg protocol.InterviewConfig) (protocol.Rubric, error) {
	var errs []error
	for _, resolver := range c.resolvers {
		if resolver == nil {
			continue
		}
		value, err := resolver.Resolve(ctx, cfg)
		if err == nil {
			return value, nil
		}
		errs = append(errs, err)
	}
	return protocol.Rubric{}, joinErrors("rubric resolver chain failed", errs)
}

type checkpointStoreChain struct {
	stores []CheckpointStore
}

func (c checkpointStoreChain) Save(ctx context.Context, snapshot protocol.CheckpointSnapshot) error {
	var errs []error
	for _, store := range c.stores {
		if store == nil {
			continue
		}
		if err := store.Save(ctx, snapshot); err == nil {
			return nil
		} else {
			errs = append(errs, err)
		}
	}
	return joinErrors("checkpoint store chain failed", errs)
}

func (c checkpointStoreChain) Load(ctx context.Context, runID string) (protocol.CheckpointSnapshot, error) {
	var errs []error
	for _, store := range c.stores {
		if store == nil {
			continue
		}
		value, err := store.Load(ctx, runID)
		if err == nil {
			return value, nil
		}
		errs = append(errs, err)
	}
	return protocol.CheckpointSnapshot{}, joinErrors("checkpoint store chain failed", errs)
}

type memoryStoreChain struct {
	stores []MemoryStore
}

func (c memoryStoreChain) Append(ctx context.Context, record protocol.MemoryRecord) error {
	var errs []error
	for _, store := range c.stores {
		if store == nil {
			continue
		}
		if err := store.Append(ctx, record); err == nil {
			return nil
		} else {
			errs = append(errs, err)
		}
	}
	return joinErrors("memory store chain failed", errs)
}

func (c memoryStoreChain) List(ctx context.Context, runID string) ([]protocol.MemoryRecord, error) {
	var errs []error
	for _, store := range c.stores {
		if store == nil {
			continue
		}
		value, err := store.List(ctx, runID)
		if err == nil {
			return value, nil
		}
		errs = append(errs, err)
	}
	return nil, joinErrors("memory store chain failed", errs)
}

type artifactStoreChain struct {
	stores []ArtifactStore
}

func (c artifactStoreChain) Get(ctx context.Context, id string) (protocol.Artifact, error) {
	var errs []error
	for _, store := range c.stores {
		if store == nil {
			continue
		}
		value, err := store.Get(ctx, id)
		if err == nil {
			return value, nil
		}
		errs = append(errs, err)
	}
	return protocol.Artifact{}, joinErrors("artifact store chain failed", errs)
}

func (c artifactStoreChain) ListByConversation(ctx context.Context, conversationID string) ([]protocol.Artifact, error) {
	var errs []error
	for _, store := range c.stores {
		if store == nil {
			continue
		}
		value, err := store.ListByConversation(ctx, conversationID)
		if err == nil {
			return value, nil
		}
		errs = append(errs, err)
	}
	return nil, joinErrors("artifact store chain failed", errs)
}

type webSearchProviderChain struct {
	providers []WebSearchProvider
}

func (c webSearchProviderChain) Search(ctx context.Context, query string, limit int) ([]protocol.WebSearchResult, error) {
	var errs []error
	for _, provider := range c.providers {
		if provider == nil {
			continue
		}
		value, err := provider.Search(ctx, query, limit)
		if err == nil {
			return value, nil
		}
		errs = append(errs, err)
	}
	return nil, joinErrors("web search provider chain failed", errs)
}

func joinErrors(prefix string, errs []error) error {
	if len(errs) == 0 {
		return fmt.Errorf("%s: no providers available", prefix)
	}
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		if err == nil {
			continue
		}
		parts = append(parts, err.Error())
	}
	if len(parts) == 0 {
		return fmt.Errorf("%s: no providers available", prefix)
	}
	return errors.New(prefix + ": " + strings.Join(parts, "; "))
}
