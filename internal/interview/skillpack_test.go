package interview

import (
	"strings"
	"testing"

	"mockinterview/internal/protocol"
)

func TestApplyFocusConstraintsPrependsGeneratedFocusContent(t *testing.T) {
	t.Parallel()

	spec := protocol.SkillSpec{
		FocusAreas:      []string{"reliability"},
		SampleQuestions: []string{"How do you scale the worker pool?"},
		FollowUps:       []string{"What is the fallback?"},
		Scenarios:       []string{"Traffic grows 3x overnight."},
	}

	constrained := ApplyFocusConstraints(spec, []string{"observability", "reliability"})
	if len(constrained.FocusAreas) < 2 || constrained.FocusAreas[0] != "observability" {
		t.Fatalf("expected focus areas to prioritize explicit constraints, got %#v", constrained.FocusAreas)
	}
	if !strings.Contains(constrained.SampleQuestions[0], "observability") {
		t.Fatalf("expected generated focus question first, got %#v", constrained.SampleQuestions)
	}
	if !strings.Contains(constrained.FollowUps[0], "observability") {
		t.Fatalf("expected generated follow-up first, got %#v", constrained.FollowUps)
	}
	if !strings.Contains(constrained.Scenarios[0], "observability") {
		t.Fatalf("expected generated scenario first, got %#v", constrained.Scenarios)
	}
}

func TestConstrainSkillSpecForDecisionAddsModeSpecificFallbacks(t *testing.T) {
	t.Parallel()

	spec := protocol.SkillSpec{
		FocusAreas:  []string{"tradeoff", "reliability"},
		Pressure:    []string{"Summarize the tradeoff under pressure."},
		Adversarial: []string{"Challenge the weakest assumption."},
		Scenarios:   []string{"A queue builds up under sustained load."},
		FollowUps:   []string{"How do you prove impact from the resume?"},
	}

	stressSpec := ConstrainSkillSpecForDecision(spec, []string{"observability"}, protocol.ModeStress)
	if len(stressSpec.Pressure) == 0 || !strings.Contains(strings.ToLower(stressSpec.Pressure[0]), "time pressure") {
		t.Fatalf("expected stress mode to inject pressure fallback, got %#v", stressSpec.Pressure)
	}

	systemDesignSpec := ConstrainSkillSpecForDecision(spec, []string{"observability"}, protocol.ModeSystemDesign)
	if len(systemDesignSpec.Scenarios) == 0 || !strings.Contains(strings.ToLower(systemDesignSpec.Scenarios[0]), "observability") {
		t.Fatalf("expected system design mode to prioritize observability scenario, got %#v", systemDesignSpec.Scenarios)
	}

	resumeSpec := ConstrainSkillSpecForDecision(spec, []string{"ownership"}, protocol.ModeResumeDeepDive)
	if len(resumeSpec.FollowUps) == 0 || !strings.Contains(strings.ToLower(resumeSpec.FollowUps[0]), "ownership") {
		t.Fatalf("expected resume deep dive to bias ownership follow-ups, got %#v", resumeSpec.FollowUps)
	}
}

func TestPrioritizeByFocusFallsBackAndRespectsLimit(t *testing.T) {
	t.Parallel()

	values := prioritizeByFocus(
		[]string{"scale batch jobs", "ownership retro"},
		[]string{"observability"},
		[]string{"trace saturation", "metric cardinality", "log correlation"},
		2,
	)

	if len(values) != 2 {
		t.Fatalf("expected limit 2, got %#v", values)
	}
	if values[0] != "trace saturation" || values[1] != "metric cardinality" {
		t.Fatalf("expected fallback ordering when no focus matched, got %#v", values)
	}
}
