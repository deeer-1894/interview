package adversarial

import (
	"strings"

	runtimepkg "mockinterview/internal/control/runtime"
	domain "mockinterview/internal/interview"
	"mockinterview/internal/protocol"
)

type Middleware struct{}

func (Middleware) Name() string { return "adversarial" }

func (Middleware) Spec() runtimepkg.MiddlewareSpec {
	return runtimepkg.MiddlewareSpec{
		Name:     "adversarial",
		Requires: []string{"planning"},
		Modes: []protocol.InterviewMode{
			protocol.ModeStress,
			protocol.ModeWeaknessFocused,
			protocol.ModeSystemDesign,
		},
	}
}

func (Middleware) Handle(ctx *runtimepkg.RunContext, next runtimepkg.Next) error {
	state, _, ok := ShouldApply(ctx)
	if !ok {
		return next(ctx)
	}

	additions := make([]string, 0, 3)
	if scenario := domain.SelectScenario(ctx.Resolved.Interview.Skill, state); scenario != "" && state.Phase != protocol.PhaseWarmup {
		additions = append(additions, "Scenario pressure: "+scenario)
	}
	if prompt := domain.SelectAdversarialPrompt(ctx.Resolved.Interview.Skill, state); prompt != "" {
		additions = append(additions, "Adversarial angle: "+prompt)
	}
	if prompt := domain.SelectPressurePrompt(ctx.Resolved.Interview.Skill, state); prompt != "" {
		additions = append(additions, "Pressure constraint: "+prompt)
	}
	if len(additions) == 0 {
		return next(ctx)
	}

	var b strings.Builder
	b.WriteString(strings.TrimSpace(ctx.Prompt))
	b.WriteString("\n\nInterviewer pressure policy:\n")
	for _, line := range additions {
		b.WriteString("- ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	ctx.Prompt = strings.TrimSpace(b.String())
	return next(ctx)
}
