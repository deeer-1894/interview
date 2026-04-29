package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type QuestionQualityMiddleware struct {
	adk.BaseChatModelAgentMiddleware
}

type QuestionQualityEvent struct {
	Type       string   `json:"type"`
	Passed     bool     `json:"passed"`
	Violations []string `json:"violations,omitempty"`
	TextHash   string   `json:"textHash,omitempty"`
}

func init() {
	schema.RegisterName[QuestionQualityEvent]("_offerbot_question_quality_event")
}

func NewQuestionQualityMiddleware() *QuestionQualityMiddleware {
	return &QuestionQualityMiddleware{}
}

func (m *QuestionQualityMiddleware) AfterModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	if state == nil || len(state.Messages) == 0 {
		return ctx, state, nil
	}
	last := state.Messages[len(state.Messages)-1]
	if last == nil || last.Role != schema.Assistant || strings.TrimSpace(last.Content) == "" {
		return ctx, state, nil
	}

	turn, _ := TurnContextFrom(ctx)
	violations := validateQuestionQuality(last.Content, turn.ForbiddenFacts)
	textHash := questionHash(last.Content)
	for _, recent := range turn.RecentQuestions {
		if recent.TextHash != "" && recent.TextHash == textHash {
			violations = append(violations, "duplicate_question")
			break
		}
	}

	_ = adk.SendEvent(ctx, &adk.AgentEvent{Action: &adk.AgentAction{CustomizedAction: QuestionQualityEvent{
		Type:       "question_quality",
		Passed:     len(violations) == 0,
		Violations: violations,
		TextHash:   textHash,
	}}})
	return ctx, state, nil
}

func validateQuestionQuality(text string, forbiddenFacts []string) []string {
	var violations []string
	if countQuestionMarks(text) > 1 {
		violations = append(violations, "multiple_questions")
	}
	for _, fact := range forbiddenFacts {
		fact = strings.TrimSpace(fact)
		if fact == "" {
			continue
		}
		if strings.Contains(strings.ToLower(text), strings.ToLower(fact)) {
			violations = append(violations, "forbidden_project_fact")
			break
		}
	}
	return violations
}

func countQuestionMarks(text string) int {
	count := 0
	for _, r := range text {
		if r == '?' || r == '？' {
			count++
		}
	}
	return count
}

func questionHash(text string) string {
	normalized := normalizeQuestion(text)
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func normalizeQuestion(text string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(text) {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
