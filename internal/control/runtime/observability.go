package runtime

import (
	"strings"
	"time"

	"mockinterview/internal/protocol"
)

func EstimateTextTokens(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	count := len([]rune(value)) / 4
	if count == 0 {
		return 1
	}
	return count
}

func BuildRunMetrics(ctx *RunContext, status protocol.RunStatus, completedAt time.Time, err error) protocol.RunMetrics {
	conversationTokens := 0
	for _, message := range ctx.Messages {
		conversationTokens += EstimateTextTokens(message.Content)
	}
	outputTokens := EstimateTextTokens(ctx.Result.Output)
	promptTokens := EstimateTextTokens(ctx.Prompt)

	assistantTurns := 0
	for _, message := range ctx.Messages {
		if strings.EqualFold(message.Role, "assistant") {
			assistantTurns++
		}
	}
	if strings.TrimSpace(ctx.Result.Output) != "" {
		assistantTurns++
	}

	metrics := protocol.RunMetrics{
		RunID:                       ctx.Run.ID,
		Status:                      status,
		Success:                     status == protocol.RunCompleted,
		DurationMs:                  completedAt.Sub(ctx.Run.CreatedAt).Milliseconds(),
		MessageCount:                len(ctx.Messages),
		AssistantTurnCount:          assistantTurns,
		EstimatedPromptTokens:       promptTokens,
		EstimatedConversationTokens: conversationTokens,
		EstimatedOutputTokens:       outputTokens,
		EstimatedTotalTokens:        promptTokens + conversationTokens + outputTokens,
		PromptVersion:               ctx.PromptVersion,
		Error:                       protocol.ErrorPayloadFromError(err),
	}
	if metrics.DurationMs < 0 {
		metrics.DurationMs = 0
	}
	return metrics
}

func buildMiddlewareSummary(name string, ctx *RunContext, duration time.Duration, err error) protocol.MiddlewareSummary {
	spec := ctx.MiddlewareSpec(name)
	chainIndex := 0
	for index, item := range ctx.MiddlewareChain {
		if item == spec.Name {
			chainIndex = index + 1
			break
		}
	}
	return protocol.MiddlewareSummary{
		Name:           name,
		Requires:       append([]string(nil), spec.Requires...),
		ChainIndex:     chainIndex,
		ChainSize:      len(ctx.MiddlewareChain),
		Status:         middlewareStatus(err),
		DurationMs:     duration.Milliseconds(),
		PromptSummary:  summarizeForTelemetry(ctx.Prompt, 140),
		OutputSummary:  summarizeForTelemetry(firstNonEmptyTelemetry(ctx.Result.Output, ctx.Result.Summary), 140),
		PlanTitle:      strings.TrimSpace(ctx.Resolved.Execution.Plan.Title),
		PlanStepCount:  len(ctx.Resolved.Execution.Plan.Steps),
		ArtifactCount:  len(ctx.Resolved.Execution.Artifacts),
		MemoryCount:    len(ctx.Resolved.Execution.Memory),
		WebResultCount: len(ctx.Resolved.Execution.WebResults),
		Error:          protocol.ErrorPayloadFromError(err),
	}
}

func middlewareStatus(err error) string {
	if err != nil {
		return "error"
	}
	return "ok"
}

func firstNonEmptyTelemetry(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func summarizeForTelemetry(value string, limit int) string {
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
