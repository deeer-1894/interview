package lifecycle

import (
	"fmt"
	"strings"

	"mockinterview/internal/protocol"
)

type ToolRoutingInput struct {
	Config protocol.InterviewConfig
	Prompt string
	Plan   protocol.ExecutionPlan
}

func ShouldRunWebResearch(input ToolRoutingInput) bool {
	return input.Config.EnableWebSearch &&
		strings.TrimSpace(input.Prompt) != "" &&
		PlanNeedsResearch(input.Plan)
}

func BuildWebAwarePrompt(prompt string, results []protocol.WebSearchResult) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(prompt))
	b.WriteString("\n\nRecent web references:\n")
	limit := 3
	if len(results) < limit {
		limit = len(results)
	}
	for _, result := range results[:limit] {
		b.WriteString(fmt.Sprintf("- %s\n  Summary: %s\n", result.Title, trimSnippet(result.Snippet, 180)))
	}
	b.WriteString("Use these references only when directly relevant to the current interview turn.\n")
	return b.String()
}

func PlanNeedsResearch(plan protocol.ExecutionPlan) bool {
	for _, step := range plan.Steps {
		if step.Kind == "research" {
			return true
		}
	}
	return false
}

func trimSnippet(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return strings.TrimSpace(value[:limit]) + "..."
}
