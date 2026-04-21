package middleware

import (
	"testing"

	runtimepkg "mockinterview/internal/control/runtime"
	"mockinterview/internal/protocol"
)

func TestDefaultChainHasSatisfiedDependenciesInStandardMode(t *testing.T) {
	t.Parallel()

	specs := collectChainSpecs(t, protocol.ModeStandard)
	assertSatisfiedDependencies(t, specs)

	if len(specs) == 0 || specs[0].Name != "output" {
		t.Fatalf("expected output middleware to wrap the chain, got %#v", specs)
	}
	assertChainContainsInOrder(t, specs, "output", "setup", "checkpoint", "clarify", "skill", "planning", "tool_routing", "summarization")
	assertChainExcludes(t, specs, "adversarial")
}

func TestDefaultChainAddsAdversarialMiddlewareForStressModes(t *testing.T) {
	t.Parallel()

	specs := collectChainSpecs(t, protocol.ModeStress)
	assertSatisfiedDependencies(t, specs)
	assertChainContainsInOrder(t, specs, "planning", "adversarial", "tool_routing")
}

func collectChainSpecs(t *testing.T, mode protocol.InterviewMode) []runtimepkg.MiddlewareSpec {
	t.Helper()

	chain := NewDefaultChain()
	specs := make([]runtimepkg.MiddlewareSpec, 0, len(chain))
	for _, middleware := range chain {
		aware, ok := middleware.(runtimepkg.DependencyAwareMiddleware)
		if !ok {
			t.Fatalf("middleware %q does not declare explicit dependencies", middleware.Name())
		}
		spec := aware.Spec()
		if !enabledForMode(spec, mode) {
			continue
		}
		specs = append(specs, spec)
	}
	return specs
}

func enabledForMode(spec runtimepkg.MiddlewareSpec, mode protocol.InterviewMode) bool {
	if len(spec.Modes) == 0 {
		return true
	}
	for _, candidate := range spec.Modes {
		if candidate == mode {
			return true
		}
	}
	return false
}

func assertSatisfiedDependencies(t *testing.T, specs []runtimepkg.MiddlewareSpec) {
	t.Helper()

	seen := make(map[string]struct{}, len(specs))
	for _, spec := range specs {
		for _, requirement := range spec.Requires {
			if _, exists := seen[requirement]; !exists {
				t.Fatalf("middleware %q requires %q before it, chain=%#v", spec.Name, requirement, specs)
			}
		}
		seen[spec.Name] = struct{}{}
	}
}

func assertChainContainsInOrder(t *testing.T, specs []runtimepkg.MiddlewareSpec, names ...string) {
	t.Helper()

	index := 0
	for _, spec := range specs {
		if index < len(names) && spec.Name == names[index] {
			index++
		}
	}
	if index != len(names) {
		t.Fatalf("expected middleware order %v, got %#v", names, specs)
	}
}

func assertChainExcludes(t *testing.T, specs []runtimepkg.MiddlewareSpec, name string) {
	t.Helper()

	for _, spec := range specs {
		if spec.Name == name {
			t.Fatalf("did not expect %q in chain %#v", name, specs)
		}
	}
}
