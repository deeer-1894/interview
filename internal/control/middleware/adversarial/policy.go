package adversarial

import (
	"strings"

	runtimepkg "mockinterview/internal/control/runtime"
	domain "mockinterview/internal/interview"
	"mockinterview/internal/protocol"
)

func ShouldApply(ctx *runtimepkg.RunContext) (protocol.RunInterviewState, domain.AnswerSignalAnalysis, bool) {
	if ctx == nil || ctx.Run.InterviewState == nil {
		return protocol.RunInterviewState{}, domain.AnswerSignalAnalysis{}, false
	}
	state := domain.EnsureRunInterviewState(ctx.Run.InterviewState)
	answer := latestUserMessage(ctx.Messages)
	analysis := domain.AnalyzeAnswerSignals(answer, ctx.Resolved.Interview.Skill)
	if state.LastDecision != nil && state.LastDecision.TriggerAdversarial {
		return state, analysis, true
	}
	return state, analysis, domain.ShouldTriggerAdversarial(state, analysis)
}

func latestUserMessage(messages []protocol.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.EqualFold(messages[i].Role, "user") {
			return strings.TrimSpace(messages[i].Content)
		}
	}
	return ""
}
