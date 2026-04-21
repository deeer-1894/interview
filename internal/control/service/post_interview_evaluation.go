package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	lifecyclepkg "mockinterview/internal/control/lifecycle"
	runtimepkg "mockinterview/internal/control/runtime"
	"mockinterview/internal/protocol"
)

func (a *App) completePostInterviewEvaluation(ctx context.Context, runCtx *runtimepkg.RunContext) error {
	if !shouldCompletePostInterviewEvaluation(runCtx) {
		return nil
	}

	run, err := a.runs.Get(ctx, runCtx.Run.ID)
	if err != nil {
		return fmt.Errorf("load run for evaluation: %w", err)
	}
	if run.Status == protocol.RunCancelled || errors.Is(ctx.Err(), context.Canceled) {
		return context.Canceled
	}
	if run.Status == protocol.RunCompleted || run.Status == protocol.RunFailed {
		return nil
	}

	messages, err := a.messages.ListByRun(ctx, run.ID)
	if err != nil {
		return fmt.Errorf("load messages for evaluation: %w", err)
	}
	events, _ := a.events.ListByRun(ctx, run.ID)
	completedAt := time.Now()
	scorecardGenerator := a.scorecards
	if err := lifecyclepkg.CompletePostInterviewEvaluation(ctx, a, runCtx, run, messages, events, scorecardGenerator, completedAt); err != nil {
		return fmt.Errorf("complete run after evaluation: %w", err)
	}
	return nil
}

func (a *App) finalizeRunFailure(ctx context.Context, runCtx *runtimepkg.RunContext, err error) {
	_ = lifecyclepkg.FinalizeRunFailure(ctx, a, runCtx, err)
}

func shouldCompletePostInterviewEvaluation(runCtx *runtimepkg.RunContext) bool {
	if runCtx == nil {
		return false
	}
	if runCtx.Task.Config.OutputStyle == protocol.OutputInterviewOnly {
		return false
	}
	if len(runCtx.Resolved.Interview.Rubric.Anchors) == 0 {
		return false
	}
	if runCtx.Run.Phase != protocol.RunPhaseEvaluating {
		return false
	}
	return strings.TrimSpace(runCtx.Result.Output) != "" || lifecyclepkg.ShouldGenerateEvaluationRequest(runCtx.Run.Input)
}
