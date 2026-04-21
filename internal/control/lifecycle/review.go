package lifecycle

import (
	"strings"
	"time"

	runtimepkg "mockinterview/internal/control/runtime"
	interviewexec "mockinterview/internal/executors/interview"
	domain "mockinterview/internal/interview"
	"mockinterview/internal/protocol"
)

type ReviewSnapshotInput struct {
	Run           protocol.Run
	Task          protocol.Task
	Prompt        string
	PromptVersion string
	Skill         protocol.SkillSpec
	Rubric        protocol.Rubric
	Profile       protocol.CandidateProfile
}

func BuildInterviewerReviewSnapshot(
	runCtx *runtimepkg.RunContext,
	transcript string,
	transcriptMessages []protocol.Message,
	assistantMessage protocol.Message,
	traceTree *protocol.InterviewTraceTree,
	generatedAt time.Time,
) protocol.ReviewSnapshot {
	if runCtx == nil {
		return protocol.ReviewSnapshot{}
	}
	return BuildInterviewerReviewSnapshotFromInput(
		ReviewSnapshotInput{
			Run:           runCtx.Run,
			Task:          runCtx.Task,
			Prompt:        runCtx.Prompt,
			PromptVersion: runCtx.PromptVersion,
			Skill:         runCtx.Resolved.Interview.Skill,
			Rubric:        runCtx.Resolved.Interview.Rubric,
			Profile:       runCtx.Resolved.Interview.Profile,
		},
		transcript,
		transcriptMessages,
		assistantMessage,
		traceTree,
		generatedAt,
	)
}

func BuildInterviewerReviewSnapshotFromInput(
	input ReviewSnapshotInput,
	transcript string,
	transcriptMessages []protocol.Message,
	assistantMessage protocol.Message,
	traceTree *protocol.InterviewTraceTree,
	generatedAt time.Time,
) protocol.ReviewSnapshot {
	_ = transcriptMessages
	interviewerShared := interviewexec.BuildSharedSessionContext(
		input.Run.ID,
		input.PromptVersion,
		interviewerPhase(input.Run.Phase),
		domain.EnsureRunInterviewState(input.Run.InterviewState),
		input.Skill,
		input.Rubric,
		transcript,
	)

	snapshot := protocol.ReviewSnapshot{
		RunID:          input.Run.ID,
		GeneratedAt:    generatedAt,
		InterviewState: input.Run.InterviewState,
		Decision:       BuildReviewDecisionAudit(input.Run.InterviewState, input.Task.Config.Mode, input.Profile, input.PromptVersion),
		Agents: []protocol.AgentExecution{
			interviewexec.BuildAgentExecution(
				protocol.AgentRoleInterviewer,
				protocol.RunCompleted,
				interviewerShared,
				input.Run.CreatedAt,
				assistantMessage.CreatedAt,
				input.Prompt,
				assistantMessage.Content,
				nil,
			),
		},
	}
	if traceTree != nil {
		snapshot.Trace = traceTree
		summary := domain.BuildReviewSummary(input.Run, input.Task.Config, *traceTree, snapshot.Scorecard, snapshot.Profile)
		snapshot.Summary = &summary
	}
	return snapshot
}

func BuildReviewDecisionAudit(
	state *protocol.RunInterviewState,
	mode protocol.InterviewMode,
	profile protocol.CandidateProfile,
	promptVersion string,
) *protocol.DecisionAudit {
	normalized := domain.EnsureRunInterviewState(state)
	if normalized.LastDecision == nil {
		return nil
	}

	analysis := protocol.AnswerSignalSummary{}
	if count := len(normalized.History); count > 0 {
		latest := normalized.History[count-1]
		analysis.WeakSignals = append([]string(nil), latest.WeakSignals...)
		analysis.StrongSignals = append([]string(nil), latest.StrongSignals...)
		analysis.WeakSignalConfidence = domain.CloneSignalConfidence(latest.WeakSignalConfidence)
		analysis.StrongSignalConfidence = domain.CloneSignalConfidence(latest.StrongSignalConfidence)
		analysis.TooGeneric = signalConfidenceAtOrAbove(latest.WeakSignalConfidence, "too_generic", 0.6) || containsStringFold(latest.WeakSignals, "too_generic")
		analysis.HasTradeoff = signalConfidenceAtOrAbove(latest.StrongSignalConfidence, "tradeoff_reasoning", 0.55) || containsStringFold(latest.StrongSignals, "tradeoff_reasoning")
		analysis.HasConcreteImplementation = signalConfidenceAtOrAbove(latest.StrongSignalConfidence, "implementation_detail", 0.55) || containsStringFold(latest.StrongSignals, "implementation_detail")
	}

	return &protocol.DecisionAudit{
		State:         normalized,
		Mode:          mode,
		PromptVersion: strings.TrimSpace(promptVersion),
		Analysis:      analysis,
		ProfileFocus:  append([]string(nil), ResolveProfileFocus(profile)...),
		Decision:      *normalized.LastDecision,
	}
}

func interviewerPhase(phase protocol.RunPhase) protocol.RunPhase {
	if phase == "" || phase == protocol.RunPhaseInitial || phase == protocol.RunPhaseEvaluating {
		return protocol.RunPhaseInterviewing
	}
	return phase
}

func containsStringFold(values []string, target string) bool {
	target = strings.TrimSpace(strings.ToLower(target))
	if target == "" {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(strings.ToLower(value)) == target {
			return true
		}
	}
	return false
}

func signalConfidenceAtOrAbove(values map[string]float64, target string, threshold float64) bool {
	if len(values) == 0 {
		return false
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	return values[target] >= threshold
}
