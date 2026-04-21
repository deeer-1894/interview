package workflow

import "mockinterview/internal/protocol"

type Workflow interface {
	Name() string
	Build() []protocol.PlanStep
}

type ArtifactIngestWorkflow struct{}
type ResearchWorkflow struct{}

func (ArtifactIngestWorkflow) Name() string { return "artifact_ingest" }

func (ArtifactIngestWorkflow) Build() []protocol.PlanStep {
	return []protocol.PlanStep{
		{
			ID:          "artifact_resolve",
			Title:       "Resolve attached artifacts",
			Description: "Resolve task/run-scoped artifact bindings and prepare compact material context.",
			Kind:        "artifact_context",
		},
	}
}

func (ResearchWorkflow) Name() string { return "research" }

func (ResearchWorkflow) Build() []protocol.PlanStep {
	return []protocol.PlanStep{
		{
			ID:          "research",
			Title:       "Gather supporting context",
			Description: "Use configured search capability to gather lightweight external context.",
			Kind:        "research",
		},
	}
}
