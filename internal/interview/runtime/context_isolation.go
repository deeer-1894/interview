package runtime

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/adk"

	"mockinterview/internal/interview/resume"
	"mockinterview/internal/interview/session"
)

type ContextIsolationMiddleware struct {
	adk.BaseChatModelAgentMiddleware
}

func NewContextIsolationMiddleware() *ContextIsolationMiddleware {
	return &ContextIsolationMiddleware{}
}

func (m *ContextIsolationMiddleware) BeforeAgent(ctx context.Context, runCtx *adk.ChatModelAgentContext) (context.Context, *adk.ChatModelAgentContext, error) {
	turn, ok := TurnContextFrom(ctx)
	if !ok {
		return ctx, runCtx, nil
	}

	forbidden := forbiddenProjectFacts(turn.OtherProjects)
	turn.ForbiddenFacts = forbidden
	turn.RecentMessages = filterCrossProjectMessages(turn.RecentMessages, turn.ActiveProject, forbidden)

	return WithTurnContext(ctx, turn), runCtx, nil
}

func forbiddenProjectFacts(projects []resume.ProjectBrief) []string {
	facts := make([]string, 0, len(projects)*2)
	for _, project := range projects {
		if project.Name != "" {
			facts = append(facts, project.Name)
		}
		if project.Domain != "" && project.Domain != project.Name {
			facts = append(facts, project.Domain)
		}
	}
	return facts
}

func filterCrossProjectMessages(messages []session.Message, active resume.Project, forbidden []string) []session.Message {
	if len(messages) == 0 || len(forbidden) == 0 {
		return messages
	}
	activeTerms := projectTerms(active)
	filtered := make([]session.Message, 0, len(messages))
	for _, msg := range messages {
		if !mentionsAny(msg.Content, forbidden) || mentionsAny(msg.Content, activeTerms) {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

func projectTerms(project resume.Project) []string {
	terms := []string{project.ID, project.Name, project.Domain}
	return compactTerms(terms)
}

func mentionsAny(text string, terms []string) bool {
	normalized := strings.ToLower(text)
	for _, term := range compactTerms(terms) {
		if strings.Contains(normalized, strings.ToLower(term)) {
			return true
		}
	}
	return false
}

func compactTerms(terms []string) []string {
	result := make([]string, 0, len(terms))
	seen := make(map[string]struct{}, len(terms))
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}
		key := strings.ToLower(term)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, term)
	}
	return result
}
