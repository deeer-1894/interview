package lifecycle

import (
	"strings"

	domain "mockinterview/internal/interview"
	"mockinterview/internal/protocol"
)

func ShouldDeferEvaluation(
	cfg protocol.InterviewConfig,
	runInput string,
	messages []protocol.Message,
	state *protocol.RunInterviewState,
	rubric protocol.Rubric,
) bool {
	if cfg.OutputStyle == protocol.OutputInterviewOnly {
		return false
	}

	interviewTurnCount := countAssistantTurns(messages) + 1
	turnLimit := domain.DeriveInterviewTurnLimit(cfg.TimeBudget)
	evaluationRequested := shouldGenerateEvaluation(runInput)
	if evaluationRequested {
		return true
	}
	if len(rubric.Anchors) == 0 {
		return false
	}
	budgetReached := turnLimit > 0 && interviewTurnCount >= turnLimit
	if state != nil && state.Round > 0 {
		budgetReached = budgetReached || (turnLimit > 0 && state.Round >= turnLimit)
	}

	return budgetReached
}

func ShouldGenerateEvaluationRequest(input string) bool {
	return shouldGenerateEvaluation(input)
}

func BuildEvaluationTranscript(messages []protocol.Message, currentAssistant protocol.Message) string {
	all := make([]protocol.Message, 0, len(messages)+1)
	all = append(all, messages...)
	all = append(all, currentAssistant)

	var b strings.Builder
	for _, message := range all {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		role := "Assistant"
		if strings.EqualFold(message.Role, "user") {
			role = "User"
		}
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(content)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func countAssistantTurns(messages []protocol.Message) int {
	count := 0
	for _, message := range messages {
		if strings.EqualFold(message.Role, "assistant") && domain.CountsAsInterviewTurn(message.Content) {
			count++
		}
	}
	return count
}

func shouldGenerateEvaluation(input string) bool {
	normalized := normalizeEvaluationText(input)
	if normalized == "" {
		return false
	}

	explicitEndRequests := []string{
		"请结束",
		"现在结束",
		"到这里结束",
		"到这结束",
		"到这就行",
		"就到这里",
		"就到这",
		"结束这场面试",
		"结束面试",
		"面试到这里结束",
		"别继续了",
		"不要继续了",
		"不用继续了",
		"别再问了",
		"不要再问了",
		"停止追问",
		"只问一个问题并结束",
		"wrap up",
		"end the interview",
		"finish the interview",
		"give me the final score",
		"give me a final score",
		"give me feedback",
		"give me the feedback",
	}

	for _, keyword := range explicitEndRequests {
		if strings.Contains(normalized, normalizeEvaluationText(keyword)) {
			return true
		}
	}
	if isShortWrapupRequest(normalized) {
		return true
	}

	directResultCue := containsAnyEvaluationKeyword(normalized, []string{
		"最终评分",
		"最终反馈",
		"评分",
		"打分",
		"学习计划",
		"final score",
		"final feedback",
	})

	deferredExpectationCues := []string{
		"在最后",
		"最后给出",
		"最后再给",
		"面试结束后",
		"最后提供",
		"最后输出",
		"at the end",
		"after the interview",
		"once the interview ends",
	}
	if containsAnyEvaluationKeyword(normalized, deferredExpectationCues) &&
		!containsAnyEvaluationKeyword(normalized, []string{
			"现在",
			"直接",
			"立刻",
			"马上",
			"到这里",
			"收尾",
			"结束",
			"please give me",
			"give me now",
			"wrap up",
			"end the interview",
			"finish the interview",
		}) {
		return false
	}

	if looksLikeTechnicalAnswer(input) && !directResultCue {
		return false
	}

	hasEvaluationCue := containsAnyEvaluationKeyword(normalized, []string{
		"总结",
		"评价",
		"评估",
		"打分",
		"评分",
		"总分",
		"复盘",
		"学习计划",
		"review",
		"evaluate",
		"evaluation",
		"score",
		"feedback",
	})
	if !hasEvaluationCue {
		return false
	}

	hasRequestCue := containsAnyEvaluationKeyword(normalized, []string{
		"请",
		"请你",
		"麻烦",
		"帮我",
		"给我",
		"直接给",
		"现在给",
		"请给我",
		"please",
		"can you",
		"could you",
		"give me",
	})
	if !hasRequestCue {
		return false
	}

	hasSpecificInterviewTarget := containsAnyEvaluationKeyword(normalized, []string{
		"这场面试",
		"本场面试",
		"当前面试",
		"this interview",
		"the interview",
	})
	if hasSpecificInterviewTarget && directResultCue {
		return true
	}

	hasImmediateWrapupCue := containsAnyEvaluationKeyword(normalized, []string{
		"现在",
		"直接",
		"立刻",
		"马上",
		"到这里",
		"收尾",
		"结束",
		"别再问",
		"停止追问",
		"please give me",
		"give me now",
		"wrap up",
		"end",
		"finish",
	})
	return hasImmediateWrapupCue
}

func looksLikeTechnicalAnswer(input string) bool {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return false
	}

	signals := 0
	if strings.Contains(trimmed, "```") {
		signals++
	}
	if strings.Count(trimmed, "\n") >= 4 {
		signals++
	}
	if strings.Contains(trimmed, "func ") || strings.Contains(trimmed, "type ") || strings.Contains(trimmed, "return ") {
		signals++
	}
	if strings.Contains(trimmed, "->") || strings.Contains(trimmed, "DAG") || strings.Contains(trimmed, "checkpoint") {
		signals++
	}
	if strings.Contains(trimmed, "- ") || strings.Contains(trimmed, "1.") || strings.Contains(trimmed, "2.") {
		signals++
	}

	return len([]rune(trimmed)) >= 120 && signals >= 2
}

func containsAnyEvaluationKeyword(input string, keywords []string) bool {
	for _, keyword := range keywords {
		if matchesEvaluationKeyword(input, keyword) {
			return true
		}
	}
	return false
}

func isShortWrapupRequest(normalized string) bool {
	if normalized == "" {
		return false
	}
	if len([]rune(normalized)) > 24 {
		return false
	}
	if !containsAnyEvaluationKeyword(normalized, []string{
		"结束",
		"收尾",
		"别继续",
		"不要继续",
		"不用继续",
		"别再问",
		"不要再问",
	}) {
		return false
	}
	return !looksLikeTechnicalAnswer(normalized)
}

func matchesEvaluationKeyword(input string, keyword string) bool {
	keyword = strings.TrimSpace(strings.ToLower(keyword))
	if keyword == "" {
		return false
	}
	if isASCIIKeyword(keyword) {
		if strings.Contains(keyword, " ") {
			return strings.Contains(normalizeEvaluationText(input), normalizeEvaluationText(keyword))
		}
		for _, token := range evaluationTokens(input) {
			if token == keyword {
				return true
			}
		}
		return false
	}
	return strings.Contains(input, keyword)
}

func normalizeEvaluationText(value string) string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(value)))
	return strings.Join(fields, " ")
}

func evaluationTokens(input string) []string {
	return strings.FieldsFunc(strings.ToLower(input), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
}

func isASCIIKeyword(value string) bool {
	for _, r := range value {
		if r > 127 {
			return false
		}
	}
	return true
}
