package lifecycle

import (
	"fmt"
	"strings"

	"mockinterview/internal/protocol"
)

type ArtifactBindingInput struct {
	Run    protocol.Run
	Task   protocol.Task
	Prompt string
}

type ArtifactBindingPlan struct {
	ArtifactIDs []string
	Prompt      string
}

func BuildArtifactBindingPlan(input ArtifactBindingInput, artifacts []protocol.Artifact) ArtifactBindingPlan {
	artifactIDs := ResolveBoundArtifactIDs(input.Run, input.Task)
	prompt := strings.TrimSpace(input.Prompt)
	if len(artifacts) > 0 {
		prompt = BuildArtifactAwarePrompt(prompt, artifacts)
	}
	return ArtifactBindingPlan{
		ArtifactIDs: artifactIDs,
		Prompt:      prompt,
	}
}

func ResolveBoundArtifactIDs(run protocol.Run, task protocol.Task) []string {
	if len(run.ArtifactIDs) > 0 {
		return append([]string(nil), run.ArtifactIDs...)
	}
	if len(task.ArtifactIDs) > 0 {
		return append([]string(nil), task.ArtifactIDs...)
	}
	return nil
}

func BuildArtifactAwarePrompt(prompt string, artifacts []protocol.Artifact) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(prompt))
	b.WriteString("\n\nRelevant workspace artifacts:\n")
	limit := 3
	if len(artifacts) < limit {
		limit = len(artifacts)
	}
	for _, artifact := range artifacts[:limit] {
		b.WriteString(fmt.Sprintf("- %s (%s, %d bytes)\n", artifact.Name, artifact.ContentType, artifact.Size))
	}
	if len(artifacts) > limit {
		b.WriteString(fmt.Sprintf("- %d more artifacts omitted for brevity\n", len(artifacts)-limit))
	}
	b.WriteString("Use artifact names only as lightweight context. Do not assume file contents unless the user discussed them explicitly.\n")
	return b.String()
}
