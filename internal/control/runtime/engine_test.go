package runtime

import (
	"strings"
	"testing"

	"mockinterview/internal/protocol"
)

type specMiddleware struct {
	spec MiddlewareSpec
}

func (m specMiddleware) Name() string { return m.spec.Name }

func (m specMiddleware) Handle(ctx *RunContext, next Next) error {
	if next == nil {
		return nil
	}
	return next(ctx)
}

func (m specMiddleware) Spec() MiddlewareSpec { return m.spec }

func TestValidateMiddlewareChainRejectsOutOfOrderRequirement(t *testing.T) {
	t.Parallel()

	err := validateMiddlewareChain([]MiddlewareSpec{
		{Name: "planning", Requires: []string{"skill"}},
		{Name: "skill"},
	})
	if err == nil {
		t.Fatalf("expected dependency validation error")
	}
	if !strings.Contains(err.Error(), `requires "skill"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveChainFiltersModeSpecificMiddlewares(t *testing.T) {
	t.Parallel()

	engine := NewStandardEngine([]Middleware{
		specMiddleware{spec: MiddlewareSpec{Name: "output"}},
		specMiddleware{spec: MiddlewareSpec{Name: "setup", Requires: []string{"output"}}},
		specMiddleware{spec: MiddlewareSpec{Name: "planning", Requires: []string{"setup"}}},
		specMiddleware{spec: MiddlewareSpec{
			Name:     "adversarial",
			Requires: []string{"planning"},
			Modes:    []protocol.InterviewMode{protocol.ModeStress},
		}},
	})

	standardChain, standardSpecs, err := engine.resolveChain(protocol.ModeStandard)
	if err != nil {
		t.Fatalf("resolveChain standard returned error: %v", err)
	}
	if len(standardChain) != 3 || len(standardSpecs) != 3 {
		t.Fatalf("expected 3 middlewares in standard mode, got %d / %d", len(standardChain), len(standardSpecs))
	}
	for _, spec := range standardSpecs {
		if spec.Name == "adversarial" {
			t.Fatalf("did not expect adversarial middleware in standard mode")
		}
	}

	stressChain, stressSpecs, err := engine.resolveChain(protocol.ModeStress)
	if err != nil {
		t.Fatalf("resolveChain stress returned error: %v", err)
	}
	if len(stressChain) != 4 || len(stressSpecs) != 4 {
		t.Fatalf("expected 4 middlewares in stress mode, got %d / %d", len(stressChain), len(stressSpecs))
	}
	if stressSpecs[3].Name != "adversarial" {
		t.Fatalf("expected adversarial middleware to be appended in stress mode, got %#v", stressSpecs)
	}
}

func TestRunContextConfigureMiddlewareChainStoresNormalizedSpecs(t *testing.T) {
	t.Parallel()

	var ctx RunContext
	ctx.ConfigureMiddlewareChain([]MiddlewareSpec{
		{Name: " Output ", Requires: []string{"Setup"}},
		{Name: "Setup"},
	})

	if len(ctx.MiddlewareChain) != 2 {
		t.Fatalf("expected configured chain, got %#v", ctx.MiddlewareChain)
	}
	if ctx.MiddlewareChain[0] != "output" {
		t.Fatalf("expected normalized middleware name, got %#v", ctx.MiddlewareChain)
	}
	spec := ctx.MiddlewareSpec("output")
	if len(spec.Requires) != 1 || spec.Requires[0] != "setup" {
		t.Fatalf("expected normalized requirements, got %#v", spec)
	}
}
