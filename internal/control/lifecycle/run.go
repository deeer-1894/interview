package lifecycle

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	runtimepkg "mockinterview/internal/control/runtime"
	domain "mockinterview/internal/interview"
	"mockinterview/internal/protocol"
)

type RunOutputInput struct {
	Task          protocol.Task
	Run           protocol.Run
	Messages      []protocol.Message
	Prompt        string
	PromptVersion string
	Output        string
	Summary       string
	Profile       protocol.CandidateProfile
	Rubric        protocol.Rubric
	Skill         protocol.SkillSpec
}

type RunOutputOutcome struct {
	Transcript        string
	Run               protocol.Run
	EvaluationPending bool
	AssistantMessage  protocol.Message
	TraceTree         protocol.InterviewTraceTree
	ReviewSnapshot    *protocol.ReviewSnapshot
}

type RunStartInput struct {
	Task      protocol.Task
	Run       protocol.Run
	StartedAt time.Time
}

type RunStartOutcome struct {
	Run          protocol.Run
	StartedEvent protocol.Event
	PersonaEvent protocol.Event
}

func BuildRunStartOutcome(input RunStartInput) RunStartOutcome {
	startedAt := input.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now()
	}

	run := input.Run
	run.Status = protocol.RunRunning

	return RunStartOutcome{
		Run: run,
		StartedEvent: protocol.Event{
			ID:             uuid.NewString(),
			ConversationID: run.ConversationID,
			TaskID:         run.TaskID,
			RunID:          run.ID,
			Type:           protocol.EventRunStarted,
			Timestamp:      startedAt,
			Payload: map[string]string{
				"status": string(protocol.RunRunning),
			},
		},
		PersonaEvent: protocol.Event{
			ID:             uuid.NewString(),
			ConversationID: run.ConversationID,
			TaskID:         run.TaskID,
			RunID:          run.ID,
			Type:           protocol.EventPersonaSelected,
			Timestamp:      startedAt,
			Payload: map[string]string{
				"persona": string(input.Task.Config.Persona),
			},
		},
	}
}

func RecordRunMetricsEvent(ctx *runtimepkg.RunContext, completedAt time.Time, err error) error {
	if ctx == nil || ctx.Recorder == nil {
		return nil
	}
	return ctx.Recorder.RecordEvent(ctx.Context, protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: ctx.Run.ConversationID,
		TaskID:         ctx.Run.TaskID,
		RunID:          ctx.Run.ID,
		Type:           protocol.EventRunMetrics,
		Timestamp:      completedAt,
		Payload:        runtimepkg.BuildRunMetrics(ctx, ctx.Run.Status, completedAt, err),
	})
}

func FinalizeRunCancellation(ctx context.Context, recorder runtimepkg.EventRecorder, runCtx *runtimepkg.RunContext, err error) error {
	if recorder == nil || runCtx == nil {
		return err
	}

	now := time.Now()
	runCtx.Run.Status = protocol.RunCancelled
	runCtx.Run.LastError = "run cancelled"
	runCtx.Run.CompletedAt = &now
	runCtx.Run.UpdatedAt = now
	if updateErr := recorder.UpdateRun(ctx, runCtx.Run); updateErr != nil {
		return updateErr
	}
	if metricsErr := RecordRunMetricsEvent(runCtx, now, err); metricsErr != nil {
		return metricsErr
	}
	return recorder.RecordEvent(ctx, protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: runCtx.Run.ConversationID,
		TaskID:         runCtx.Run.TaskID,
		RunID:          runCtx.Run.ID,
		Type:           protocol.EventRunCancelled,
		Timestamp:      now,
		Payload: map[string]any{
			"status": string(protocol.RunCancelled),
			"error":  runCtx.Run.LastError,
		},
	})
}

func FinalizeRunFailure(ctx context.Context, recorder runtimepkg.EventRecorder, runCtx *runtimepkg.RunContext, err error) error {
	if recorder == nil || runCtx == nil {
		return err
	}

	now := time.Now()
	payload := protocol.ErrorPayloadFromError(err)
	runCtx.Run.Status = protocol.RunFailed
	runCtx.Run.LastError = err.Error()
	if payload != nil && strings.TrimSpace(payload.Message) != "" {
		runCtx.Run.LastError = payload.Message
	}
	runCtx.Run.CompletedAt = &now
	runCtx.Run.UpdatedAt = now
	if updateErr := recorder.UpdateRun(ctx, runCtx.Run); updateErr != nil {
		return updateErr
	}
	if eventErr := recorder.RecordEvent(ctx, protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: runCtx.Run.ConversationID,
		TaskID:         runCtx.Run.TaskID,
		RunID:          runCtx.Run.ID,
		Type:           protocol.EventRunFailed,
		Timestamp:      now,
		Payload: map[string]any{
			"error":     runCtx.Run.LastError,
			"kind":      payloadKind(payload),
			"stage":     payloadStage(payload),
			"operation": payloadOperation(payload),
			"retryable": payloadRetryable(payload),
		},
	}); eventErr != nil {
		return eventErr
	}
	return RecordRunMetricsEvent(runCtx, now, err)
}

func FinalizeRunCompleted(
	ctx context.Context,
	recorder runtimepkg.EventRecorder,
	runCtx *runtimepkg.RunContext,
	completedAt time.Time,
	reviewSnapshot *protocol.ReviewSnapshot,
) error {
	if recorder == nil || runCtx == nil {
		return nil
	}

	runCtx.Run.Status = protocol.RunCompleted
	runCtx.Run.Phase = protocol.RunPhaseCompleted
	runCtx.Run.CompletedAt = &completedAt
	runCtx.Run.UpdatedAt = completedAt
	if updateErr := recorder.UpdateRun(ctx, runCtx.Run); updateErr != nil {
		return updateErr
	}
	if reviewSnapshot != nil {
		if eventErr := recorder.RecordEvent(ctx, protocol.Event{
			ID:             uuid.NewString(),
			ConversationID: runCtx.Run.ConversationID,
			TaskID:         runCtx.Run.TaskID,
			RunID:          runCtx.Run.ID,
			Type:           protocol.EventReviewGenerated,
			Timestamp:      completedAt,
			Payload:        *reviewSnapshot,
		}); eventErr != nil {
			return eventErr
		}
	}
	if eventErr := recorder.RecordEvent(ctx, protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: runCtx.Run.ConversationID,
		TaskID:         runCtx.Run.TaskID,
		RunID:          runCtx.Run.ID,
		Type:           protocol.EventRunCompleted,
		Timestamp:      completedAt,
		Payload: map[string]string{
			"status": string(protocol.RunCompleted),
		},
	}); eventErr != nil {
		return eventErr
	}
	return RecordRunMetricsEvent(runCtx, completedAt, nil)
}

func PersistRunOutput(ctx context.Context, runCtx *runtimepkg.RunContext, completedAt time.Time) error {
	if runCtx == nil || runCtx.Recorder == nil || strings.TrimSpace(runCtx.Result.Output) == "" {
		return nil
	}

	outcome := BuildRunOutputOutcome(RunOutputInput{
		Task:          runCtx.Task,
		Run:           runCtx.Run,
		Messages:      runCtx.Messages,
		Prompt:        runCtx.Prompt,
		PromptVersion: runCtx.PromptVersion,
		Output:        runCtx.Result.Output,
		Summary:       runCtx.Result.Summary,
		Profile:       runCtx.Resolved.Interview.Profile,
		Rubric:        runCtx.Resolved.Interview.Rubric,
		Skill:         runCtx.Resolved.Interview.Skill,
	}, completedAt)
	runCtx.Run = outcome.Run
	if err := runCtx.Recorder.UpdateRun(ctx, outcome.Run); err != nil {
		return err
	}

	if err := runCtx.Recorder.RecordMessage(ctx, outcome.AssistantMessage); err != nil {
		return err
	}
	if runCtx.Tools != nil {
		if err := runCtx.Tools.AppendMemory(ctx, protocol.MemoryRecord{
			RunID:      outcome.Run.ID,
			Content:    firstNonEmptyString(runCtx.Result.Summary, runCtx.Result.Output),
			RecordedAt: completedAt,
		}); err != nil {
			return err
		}
	}
	if err := runCtx.Recorder.RecordEvent(ctx, protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: outcome.Run.ConversationID,
		TaskID:         outcome.Run.TaskID,
		RunID:          outcome.Run.ID,
		Type:           protocol.EventMessageCompleted,
		Timestamp:      completedAt,
		Payload: map[string]string{
			"content": runCtx.Result.Output,
		},
	}); err != nil {
		return err
	}

	runCtx.Run.TraceTree = &outcome.TraceTree
	if err := runCtx.Recorder.UpdateRun(ctx, runCtx.Run); err != nil {
		return err
	}
	if err := runCtx.Recorder.RecordEvent(ctx, protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: outcome.Run.ConversationID,
		TaskID:         outcome.Run.TaskID,
		RunID:          outcome.Run.ID,
		Type:           protocol.EventTraceGenerated,
		Timestamp:      completedAt,
		Payload:        outcome.TraceTree,
	}); err != nil {
		return err
	}

	if outcome.EvaluationPending {
		return nil
	}

	if outcome.Run.Phase != protocol.RunPhaseCompleted {
		return nil
	}

	return FinalizeRunCompleted(ctx, runCtx.Recorder, runCtx, completedAt, outcome.ReviewSnapshot)
}

func BuildRunOutputOutcome(input RunOutputInput, completedAt time.Time) RunOutputOutcome {
	transcript := BuildEvaluationTranscript(input.Messages, protocol.Message{})
	run := input.Run
	if run.Phase == "" || run.Phase == protocol.RunPhaseInitial {
		run.Phase = protocol.RunPhaseInterviewing
	}

	evaluationPending := ShouldDeferEvaluation(
		input.Task.Config,
		run.Input,
		input.Messages,
		run.InterviewState,
		input.Rubric,
	)
	evaluationRequested := shouldGenerateEvaluation(run.Input)
	if evaluationPending {
		if evaluationRequested {
			run = ApplyWrapupRequestedState(run)
		}
		run.Phase = protocol.RunPhaseEvaluating
		run.Status = protocol.RunRunning
		run.CompletedAt = nil
	} else {
		run.Phase = protocol.RunPhaseInterviewing
		run.Status = protocol.RunWaitingClarify
		run.CompletedAt = nil
		if evaluationRequested && input.Task.Config.OutputStyle == protocol.OutputInterviewOnly {
			run.Phase = protocol.RunPhaseCompleted
			run.Status = protocol.RunCompleted
			run.CompletedAt = &completedAt
		}
	}
	run.Output = input.Output

	assistantMessage := protocol.Message{
		ID:             uuid.NewString(),
		ConversationID: run.ConversationID,
		TaskID:         run.TaskID,
		RunID:          run.ID,
		Role:           "assistant",
		Content:        input.Output,
		CreatedAt:      completedAt,
	}
	traceTree := domain.BuildTraceTree(
		input.Task.Config.Persona,
		input.Messages,
		assistantMessage,
		run.InterviewState,
		&input.Profile,
	)
	run.TraceTree = &traceTree

	outcome := RunOutputOutcome{
		Transcript:        transcript,
		Run:               run,
		EvaluationPending: evaluationPending,
		AssistantMessage:  assistantMessage,
		TraceTree:         traceTree,
	}
	if evaluationPending || run.Phase != protocol.RunPhaseCompleted {
		return outcome
	}

	reviewSnapshot := BuildInterviewerReviewSnapshotFromInput(
		ReviewSnapshotInput{
			Run:           run,
			Task:          input.Task,
			Prompt:        input.Prompt,
			PromptVersion: input.PromptVersion,
			Skill:         input.Skill,
			Rubric:        input.Rubric,
			Profile:       input.Profile,
		},
		transcript,
		append(append([]protocol.Message(nil), input.Messages...), assistantMessage),
		assistantMessage,
		&traceTree,
		completedAt,
	)
	outcome.ReviewSnapshot = &reviewSnapshot
	return outcome
}

func ApplyWrapupRequestedState(run protocol.Run) protocol.Run {
	state := domain.EnsureRunInterviewState(run.InterviewState)
	state.Phase = protocol.PhaseWrapup
	recommendedFocus := []string(nil)
	if state.LastDecision != nil {
		recommendedFocus = append(recommendedFocus, state.LastDecision.RecommendedFocus...)
	}
	state.LastDecision = &protocol.NextStepDecision{
		Reason:           protocol.ReasonWrapupRequested,
		Explanation:      "用户明确要求结束当前面试，进入总结与评分阶段。",
		RecommendedFocus: recommendedFocus,
	}
	run.InterviewState = &state
	run.Phase = protocol.RunPhaseEvaluating
	run.Status = protocol.RunRunning
	run.CompletedAt = nil
	return run
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func payloadKind(payload *protocol.ErrorPayload) protocol.ErrorKind {
	if payload == nil {
		return ""
	}
	return payload.Kind
}

func payloadStage(payload *protocol.ErrorPayload) string {
	if payload == nil {
		return ""
	}
	return payload.Stage
}

func payloadOperation(payload *protocol.ErrorPayload) string {
	if payload == nil {
		return ""
	}
	return payload.Operation
}

func payloadRetryable(payload *protocol.ErrorPayload) bool {
	return payload != nil && payload.Retryable
}
