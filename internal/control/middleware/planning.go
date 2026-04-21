package middleware

import (
	"strings"
	"time"

	lifecyclepkg "mockinterview/internal/control/lifecycle"
	runtimepkg "mockinterview/internal/control/runtime"
	"mockinterview/internal/protocol"
)

type Setup struct{}
type Planning struct{}

func (Setup) Name() string { return "setup" }

func (Setup) Spec() runtimepkg.MiddlewareSpec {
	return runtimepkg.MiddlewareSpec{
		Name:     "setup",
		Requires: []string{"output"},
	}
}

func (Setup) Handle(ctx *runtimepkg.RunContext, next runtimepkg.Next) error {
	outcome := lifecyclepkg.PrepareRunPrompt(lifecyclepkg.SetupInput{
		Task:    ctx.Task,
		Run:     ctx.Run,
		Request: ctx.Request,
		Prompt:  ctx.Prompt,
	})
	ctx.Task = outcome.Task
	ctx.Run = outcome.Run
	ctx.Prompt = outcome.Prompt
	return next(ctx)
}

func (Planning) Name() string { return "planning" }

func (Planning) Spec() runtimepkg.MiddlewareSpec {
	return runtimepkg.MiddlewareSpec{
		Name:     "planning",
		Requires: []string{"artifact_binding", "skill"},
	}
}

func (Planning) Handle(ctx *runtimepkg.RunContext, next runtimepkg.Next) error {
	profile := ctx.Resolved.Interview.Profile
	if ctx.Recorder != nil {
		if savedProfile, err := ctx.Recorder.GetCandidateProfile(ctx.Context); err == nil {
			profile = savedProfile
		}
	}
	input := lifecyclepkg.PlanningInput{
		Prompt:       ctx.Prompt,
		Config:       ctx.Task.Config,
		Profile:      profile,
		Skill:        ctx.Resolved.Interview.Skill,
		Artifacts:    ctx.Resolved.Execution.Artifacts,
		LatestAnswer: latestPlanningUserAnswer(ctx.Messages),
	}
	if ctx.Run.InterviewState != nil {
		input.State = *ctx.Run.InterviewState
	}
	outcome := lifecyclepkg.PlanTurn(input)
	ctx.PromptVersion = outcome.PromptVersion
	ctx.Run.InterviewState = &outcome.State
	ctx.Resolved.Interview.Profile = outcome.Profile
	ctx.Resolved.Interview.Skill = outcome.Skill
	ctx.Prompt = outcome.Prompt
	ctx.Resolved.Execution.Plan = outcome.Plan
	if ctx.Recorder != nil {
		decisionEvent, planEvent := lifecyclepkg.BuildPlanningEvents(lifecyclepkg.PlanningEventInput{
			Run:           ctx.Run,
			Task:          ctx.Task,
			PromptVersion: ctx.PromptVersion,
			Outcome:       outcome,
			Timestamp:     time.Now(),
		})
		_ = ctx.Recorder.RecordEvent(ctx.Context, decisionEvent)
		_ = ctx.Recorder.RecordEvent(ctx.Context, planEvent)
	}
	return next(ctx)
}

func latestPlanningUserAnswer(messages []protocol.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.EqualFold(messages[i].Role, "user") {
			return strings.TrimSpace(messages[i].Content)
		}
	}
	return ""
}
