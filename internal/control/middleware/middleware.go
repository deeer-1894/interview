package middleware

import (
	"context"
	"errors"
	"strings"
	"time"

	lifecyclepkg "mockinterview/internal/control/lifecycle"
	adversarialmw "mockinterview/internal/control/middleware/adversarial"
	runtimepkg "mockinterview/internal/control/runtime"
	"mockinterview/internal/protocol"
)

type Skill struct{}
type Clarify struct{}
type Memory struct{}
type ArtifactBinding struct{}
type ToolRouting struct{}
type Output struct{}
type Checkpoint struct{}

func NewDefaultChain() []runtimepkg.Middleware {
	return []runtimepkg.Middleware{
		Output{},
		Setup{},
		Checkpoint{},
		Clarify{},
		Skill{},
		Memory{},
		Reduction{},
		ArtifactBinding{},
		Planning{},
		adversarialmw.Middleware{},
		ToolRouting{},
		Summarization{},
	}
}

func (Skill) Name() string { return "skill" }

func (Skill) Spec() runtimepkg.MiddlewareSpec {
	return runtimepkg.MiddlewareSpec{
		Name:     "skill",
		Requires: []string{"checkpoint"},
	}
}

func (Skill) Handle(ctx *runtimepkg.RunContext, next runtimepkg.Next) error {
	if ctx.Tools != nil {
		resolved, err := lifecyclepkg.ResolveInterviewAssets(ctx.Context, ctx.Tools, lifecyclepkg.SkillResolutionInput{
			Config: ctx.Task.Config,
		})
		if err != nil {
			return err
		}
		ctx.Resolved.Interview.Skill = resolved.Skill
		ctx.Resolved.Interview.Rubric = resolved.Rubric
	}
	return next(ctx)
}

func (Clarify) Name() string { return "clarify" }

func (Clarify) Spec() runtimepkg.MiddlewareSpec {
	return runtimepkg.MiddlewareSpec{
		Name:     "clarify",
		Requires: []string{"setup"},
	}
}

func (Clarify) Handle(ctx *runtimepkg.RunContext, next runtimepkg.Next) error {
	decision := lifecyclepkg.BuildPromptClarifyDecision(lifecyclepkg.ClarifyDecisionInput{
		Run:    ctx.Run,
		Prompt: ctx.Prompt,
		Now:    time.Now(),
	})
	if decision.NeedsClarify {
		if ctx.ClarifyRequests != nil {
			_ = ctx.ClarifyRequests.Save(ctx.Context, decision.Request)
		}
		if ctx.Recorder != nil {
			_ = ctx.Recorder.RecordEvent(ctx.Context, decision.Event)
			ctx.Run = decision.Run
			_ = ctx.Recorder.UpdateRun(ctx.Context, ctx.Run)
		}
		return nil
	}
	return next(ctx)
}

func (Memory) Name() string { return "memory" }

func (Memory) Spec() runtimepkg.MiddlewareSpec {
	return runtimepkg.MiddlewareSpec{
		Name:     "memory",
		Requires: []string{"clarify"},
	}
}

func (Memory) Handle(ctx *runtimepkg.RunContext, next runtimepkg.Next) error {
	if ctx.Tools != nil {
		loaded, err := lifecyclepkg.LoadRunMemory(ctx.Context, ctx.Tools, lifecyclepkg.MemoryLoadInput{
			Run: ctx.Run,
		})
		if err != nil {
			return err
		}
		ctx.Resolved.Execution.Memory = loaded.Records
	}
	return next(ctx)
}

func (ArtifactBinding) Name() string { return "artifact_binding" }

func (ArtifactBinding) Spec() runtimepkg.MiddlewareSpec {
	return runtimepkg.MiddlewareSpec{
		Name:     "artifact_binding",
		Requires: []string{"reduction"},
	}
}

func (ArtifactBinding) Handle(ctx *runtimepkg.RunContext, next runtimepkg.Next) error {
	if ctx.Tools != nil {
		plan := lifecyclepkg.BuildArtifactBindingPlan(lifecyclepkg.ArtifactBindingInput{
			Run:    ctx.Run,
			Task:   ctx.Task,
			Prompt: ctx.Prompt,
		}, nil)
		if len(plan.ArtifactIDs) > 0 {
			artifacts, err := loadArtifactsByID(ctx, plan.ArtifactIDs)
			if err != nil {
				return err
			}
			ctx.Resolved.Execution.Artifacts = artifacts
			plan = lifecyclepkg.BuildArtifactBindingPlan(lifecyclepkg.ArtifactBindingInput{
				Run:    ctx.Run,
				Task:   ctx.Task,
				Prompt: ctx.Prompt,
			}, artifacts)
		}
		ctx.Prompt = plan.Prompt
	}
	return next(ctx)
}

func (ToolRouting) Name() string { return "tool_routing" }

func (ToolRouting) Spec() runtimepkg.MiddlewareSpec {
	return runtimepkg.MiddlewareSpec{
		Name:     "tool_routing",
		Requires: []string{"planning"},
	}
}

func (ToolRouting) Handle(ctx *runtimepkg.RunContext, next runtimepkg.Next) error {
	if ctx.Tools != nil && lifecyclepkg.ShouldRunWebResearch(lifecyclepkg.ToolRoutingInput{
		Config: ctx.Task.Config,
		Prompt: ctx.Prompt,
		Plan:   ctx.Resolved.Execution.Plan,
	}) {
		results, err := ctx.Tools.SearchWeb(ctx.Context, ctx.Prompt, 5)
		if err == nil && len(results) > 0 {
			ctx.Resolved.Execution.WebResults = results
			ctx.Prompt = lifecyclepkg.BuildWebAwarePrompt(ctx.Prompt, results)
		}
	}
	return next(ctx)
}

func (Output) Name() string { return "output" }

func (Output) Spec() runtimepkg.MiddlewareSpec {
	return runtimepkg.MiddlewareSpec{Name: "output"}
}

func (Output) Handle(ctx *runtimepkg.RunContext, next runtimepkg.Next) error {
	ctx.Task.Config = ctx.Task.Config.WithDefaults()
	start := lifecyclepkg.BuildRunStartOutcome(lifecyclepkg.RunStartInput{
		Task:      ctx.Task,
		Run:       ctx.Run,
		StartedAt: time.Now(),
	})
	if ctx.Recorder != nil {
		ctx.Run = start.Run
		_ = ctx.Recorder.UpdateRun(ctx.Context, ctx.Run)
		_ = ctx.Recorder.RecordEvent(ctx.Context, start.StartedEvent)
		_ = ctx.Recorder.RecordEvent(ctx.Context, start.PersonaEvent)
	}

	if err := next(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			if ctx.Recorder != nil {
				_ = lifecyclepkg.FinalizeRunCancellation(ctx.Context, ctx.Recorder, ctx, err)
			}
			return err
		}
		if ctx.Recorder != nil {
			_ = lifecyclepkg.FinalizeRunFailure(ctx.Context, ctx.Recorder, ctx, err)
		}
		return err
	}

	if strings.TrimSpace(ctx.Result.Output) == "" {
		return nil
	}

	if ctx.Recorder != nil {
		if err := lifecyclepkg.PersistRunOutput(ctx.Context, ctx, time.Now()); err != nil {
			return err
		}
	}
	return nil
}

func BuildReviewDecisionAudit(
	state *protocol.RunInterviewState,
	mode protocol.InterviewMode,
	profile protocol.CandidateProfile,
	promptVersion string,
) *protocol.DecisionAudit {
	return lifecyclepkg.BuildReviewDecisionAudit(state, mode, profile, promptVersion)
}

func (Checkpoint) Name() string { return "checkpoint" }

func (Checkpoint) Spec() runtimepkg.MiddlewareSpec {
	return runtimepkg.MiddlewareSpec{
		Name:     "checkpoint",
		Requires: []string{"setup"},
	}
}

func (Checkpoint) Handle(ctx *runtimepkg.RunContext, next runtimepkg.Next) error {
	if ctx.Tools != nil && ctx.Request.Resume {
		snapshot, err := ctx.Tools.LoadCheckpoint(ctx.Context, ctx.Run.ID)
		if err == nil {
			ctx.Resolved.Interview.Checkpoint = snapshot
			outcome := lifecyclepkg.ApplyCheckpointSnapshot(lifecyclepkg.CheckpointResumeInput{
				Request: ctx.Request,
				Task:    ctx.Task,
				Run:     ctx.Run,
				Prompt:  ctx.Prompt,
			}, snapshot)
			ctx.Task = outcome.Task
			ctx.Run = outcome.Run
			ctx.Prompt = outcome.Prompt
			if ctx.Recorder != nil {
				_ = ctx.Recorder.RecordEvent(ctx.Context, lifecyclepkg.BuildCheckpointLoadedEvent(ctx.Run, snapshot, time.Now()))
			}
		}
	}

	err := next(ctx)
	if ctx.Tools != nil && (ctx.Run.Status == protocol.RunWaitingClarify || ctx.Run.Status == protocol.RunCompleted) {
		var previous *protocol.CheckpointSnapshot
		if loaded, loadErr := ctx.Tools.LoadCheckpoint(ctx.Context, ctx.Run.ID); loadErr == nil {
			previous = &loaded
		}
		snapshot := lifecyclepkg.BuildCheckpointSnapshot(lifecyclepkg.CheckpointSaveInput{
			Request:  ctx.Request,
			Task:     ctx.Task,
			Run:      ctx.Run,
			Prompt:   ctx.Prompt,
			Previous: previous,
			SavedAt:  time.Now(),
		})
		if saveErr := ctx.Tools.SaveCheckpoint(ctx.Context, snapshot); saveErr == nil && ctx.Recorder != nil {
			_ = ctx.Recorder.RecordEvent(ctx.Context, lifecyclepkg.BuildCheckpointSavedEvent(ctx.Run, snapshot, time.Now()))
		}
	}
	return err
}

func loadArtifactsByID(ctx *runtimepkg.RunContext, ids []string) ([]protocol.Artifact, error) {
	artifacts := make([]protocol.Artifact, 0, len(ids))
	for _, id := range ids {
		artifact, err := ctx.Tools.GetArtifact(ctx.Context, id)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}

func appendResumeResponse(prompt, response string) string {
	return lifecyclepkg.AppendResumeResponse(prompt, response)
}
