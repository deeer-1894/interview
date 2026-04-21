package interview

import (
	"strings"
	"time"

	"mockinterview/internal/interview/adkapp"
	"mockinterview/internal/protocol"
)

func BuildSharedSessionContext(
	runID string,
	promptVersion string,
	runPhase protocol.RunPhase,
	interviewState protocol.RunInterviewState,
	skill protocol.SkillSpec,
	rubric protocol.Rubric,
	transcript string,
) adkapp.SharedSessionContext {
	return adkapp.SharedSessionContext{
		RunID:          strings.TrimSpace(runID),
		PromptVersion:  strings.TrimSpace(promptVersion),
		RunPhase:       runPhase,
		InterviewState: interviewState,
		Skill:          skill,
		Rubric:         rubric,
		Transcript:     strings.TrimSpace(transcript),
	}
}

func BuildAgentExecution(
	role protocol.AgentRole,
	status protocol.RunStatus,
	shared adkapp.SharedSessionContext,
	startedAt time.Time,
	completedAt time.Time,
	inputSummary string,
	outputSummary string,
	err error,
) protocol.AgentExecution {
	return protocol.AgentExecution{
		Role:                 role,
		Status:               status,
		PromptVersion:        shared.PromptVersion,
		StartedAt:            startedAt,
		CompletedAt:          completedAt,
		SharedContextSummary: shared.Summary(),
		InputSummary:         summarizeAgentText(inputSummary, 180),
		OutputSummary:        summarizeAgentText(outputSummary, 180),
		Error:                protocol.ErrorPayloadFromError(err),
	}
}

func summarizeAgentText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if limit <= 0 || len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "..."
}
