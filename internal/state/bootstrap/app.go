package bootstrap

import (
	controlmiddleware "mockinterview/internal/control/middleware"
	runtimepkg "mockinterview/internal/control/runtime"
	controlservice "mockinterview/internal/control/service"
	toolgateway "mockinterview/internal/tools/gateway"
	artifactprovider "mockinterview/internal/tools/providers/artifact"
	checkpointprovider "mockinterview/internal/tools/providers/checkpoint"
	memoryprovider "mockinterview/internal/tools/providers/memory"
	remoteprovider "mockinterview/internal/tools/providers/remote"
	rubricprovider "mockinterview/internal/tools/providers/rubric"
	skillprovider "mockinterview/internal/tools/providers/skill"
	websearchprovider "mockinterview/internal/tools/providers/websearch"
	toolregistry "mockinterview/internal/tools/registry"
)

func NewAppDependencies(
	checkpoints controlservice.CheckpointRepository,
	memories controlservice.MemoryRepository,
	artifacts controlservice.ArtifactRepository,
) (controlservice.AppDependencies, error) {
	registry := toolregistry.New()
	localSkills := skillprovider.New()
	localRubrics := rubricprovider.New()
	localCheckpoints := checkpointprovider.New(checkpoints)
	localMemories := memoryprovider.New(memories)
	localArtifacts := artifactprovider.New(artifacts)
	localWebSearch := websearchprovider.New()

	registry.RegisterSkillResolver(localSkills)
	registry.RegisterRubricResolver(localRubrics)
	registry.RegisterCheckpointStore(localCheckpoints)
	registry.RegisterMemoryStore(localMemories)
	registry.RegisterArtifactStore(localArtifacts)
	registry.RegisterWebSearch(localWebSearch)

	remoteCfg := remoteprovider.ConfigFromEnv()
	if remoteCfg.Enabled {
		remoteGateway, err := remoteprovider.New(remoteCfg)
		if err != nil {
			return controlservice.AppDependencies{}, err
		}
		registry.RegisterSkillResolver(toolregistry.ChainSkillResolvers(
			remoteprovider.NewSkillResolver(remoteGateway),
			localSkills,
		))
		registry.RegisterRubricResolver(toolregistry.ChainRubricResolvers(
			remoteprovider.NewRubricResolver(remoteGateway),
			localRubrics,
		))
		registry.RegisterCheckpointStore(toolregistry.ChainCheckpointStores(
			remoteprovider.NewCheckpointStore(remoteGateway),
			localCheckpoints,
		))
		registry.RegisterMemoryStore(toolregistry.ChainMemoryStores(
			remoteprovider.NewMemoryStore(remoteGateway),
			localMemories,
		))
		registry.RegisterArtifactStore(toolregistry.ChainArtifactStores(
			remoteprovider.NewArtifactStore(remoteGateway),
			localArtifacts,
		))
		registry.RegisterWebSearch(toolregistry.ChainWebSearchProviders(
			remoteprovider.NewWebSearchProvider(remoteGateway),
			localWebSearch,
		))
	}

	tools, err := toolgateway.New(registry)
	if err != nil {
		return controlservice.AppDependencies{}, err
	}

	return controlservice.AppDependencies{
		Engine: runtimepkg.NewStandardEngine(controlmiddleware.NewDefaultChain()),
		Tools:  tools,
	}, nil
}

func NewApp(bundle Bundle, files controlservice.ArtifactFileStore) (*controlservice.App, error) {
	deps, err := NewAppDependencies(bundle.Checkpoints, bundle.Memories, bundle.Artifacts)
	if err != nil {
		return nil, err
	}
	return controlservice.NewApp(
		bundle.Conversations,
		bundle.Tasks,
		bundle.Runs,
		bundle.Messages,
		bundle.Events,
		bundle.Profiles,
		bundle.Checkpoints,
		bundle.Clarifies,
		bundle.Memories,
		bundle.Artifacts,
		files,
		deps,
	)
}
