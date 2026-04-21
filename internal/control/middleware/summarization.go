package middleware

import (
	"strings"

	lifecyclepkg "mockinterview/internal/control/lifecycle"
	runtimepkg "mockinterview/internal/control/runtime"
)

type Reduction struct{}
type Summarization struct{}

func (Reduction) Name() string { return "reduction" }

func (Reduction) Spec() runtimepkg.MiddlewareSpec {
	return runtimepkg.MiddlewareSpec{
		Name:     "reduction",
		Requires: []string{"memory"},
	}
}

func (Reduction) Handle(ctx *runtimepkg.RunContext, next runtimepkg.Next) error {
	outcome := lifecyclepkg.BuildReducedPromptContext(lifecyclepkg.ReductionInput{
		Prompt:   ctx.Prompt,
		Memories: ctx.Resolved.Execution.Memory,
		Limit:    2,
	})
	ctx.Prompt = outcome.Prompt
	return next(ctx)
}

func (Summarization) Name() string { return "summarization" }

func (Summarization) Spec() runtimepkg.MiddlewareSpec {
	return runtimepkg.MiddlewareSpec{
		Name:     "summarization",
		Requires: []string{"tool_routing"},
	}
}

func (Summarization) Handle(ctx *runtimepkg.RunContext, next runtimepkg.Next) error {
	if err := next(ctx); err != nil {
		return err
	}
	if strings.TrimSpace(ctx.Result.Output) == "" {
		return nil
	}
	ctx.Result.Summary = lifecyclepkg.BuildOutputSummary(lifecyclepkg.OutputSummaryInput{
		Output:   ctx.Result.Output,
		MaxLines: 4,
	})
	return nil
}
