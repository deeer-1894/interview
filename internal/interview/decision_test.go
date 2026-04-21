package interview

import (
	"reflect"
	"strings"
	"testing"

	"mockinterview/internal/protocol"
)

func TestDecideNextStepAddsStressExplanation(t *testing.T) {
	t.Parallel()

	decision := DecideNextStep(
		protocol.CandidateProfile{},
		protocol.RunInterviewState{Phase: protocol.PhaseProbe, Round: 1},
		AnswerSignalAnalysis{},
		protocol.InterviewConfig{Mode: protocol.ModeStress},
		protocol.SkillSpec{},
	)

	if !decision.EscalatePressure {
		t.Fatalf("expected stress mode to escalate pressure")
	}
	if strings.TrimSpace(decision.Explanation) == "" {
		t.Fatalf("expected explanation to be populated")
	}
}

func TestDecideNextStepMissingTradeoffSetsReasonAndExplanation(t *testing.T) {
	t.Parallel()

	decision := DecideNextStep(
		protocol.CandidateProfile{},
		protocol.RunInterviewState{Phase: protocol.PhaseProbe, Round: 2},
		AnswerSignalAnalysis{
			WeakSignals:          []string{"missing_tradeoff"},
			WeakSignalConfidence: map[string]float64{"missing_tradeoff": 0.9},
		},
		protocol.InterviewConfig{Mode: protocol.ModeStandard},
		protocol.SkillSpec{},
	)

	if decision.Reason != protocol.ReasonMissingTradeoff {
		t.Fatalf("expected missing tradeoff reason, got %q", decision.Reason)
	}
	if !strings.Contains(decision.Explanation, "取舍") {
		t.Fatalf("expected explanation to mention tradeoff pressure, got %q", decision.Explanation)
	}
}

func TestDecideNextStepUsesConfidenceThresholdsForAdversarial(t *testing.T) {
	t.Parallel()

	decision := DecideNextStep(
		protocol.CandidateProfile{},
		protocol.RunInterviewState{Phase: protocol.PhaseProbe, Round: 2},
		AnswerSignalAnalysis{
			WeakSignals:          []string{"too_generic"},
			WeakSignalConfidence: map[string]float64{"too_generic": 0.7},
			TooGeneric:           true,
		},
		protocol.InterviewConfig{Mode: protocol.ModeStandard},
		protocol.SkillSpec{},
	)

	if !decision.EscalatePressure {
		t.Fatalf("expected generic answer to escalate pressure")
	}
	if decision.TriggerAdversarial {
		t.Fatalf("did not expect adversarial trigger below configured threshold")
	}
}

func TestDecideNextStepUsesSkillAwarePolicy(t *testing.T) {
	t.Parallel()

	decision := DecideNextStep(
		protocol.CandidateProfile{},
		protocol.RunInterviewState{Phase: protocol.PhaseProbe, Round: 1},
		AnswerSignalAnalysis{
			WeakSignals:          []string{"missing_observability_detail"},
			WeakSignalConfidence: map[string]float64{"missing_observability_detail": 0.52},
		},
		protocol.InterviewConfig{Mode: protocol.ModeStandard},
		protocol.SkillSpec{FocusAreas: []string{"observability", "reliability"}},
	)

	if decision.Reason != protocol.ReasonWeakSignalObservability {
		t.Fatalf("expected observability-focused skill to lower threshold, got %q", decision.Reason)
	}
	if len(decision.RecommendedFocus) == 0 {
		t.Fatalf("expected policy recommended focus to be populated")
	}
}

func TestResolveStrategyConfigDefaultsToDefaultStrategy(t *testing.T) {
	t.Parallel()

	strategy := ResolveStrategyConfig(protocol.InterviewConfig{}, protocol.SkillSpec{})

	if strategy.Name != StrategyDefault {
		t.Fatalf("expected default strategy, got %q", strategy.Name)
	}
	if strategy.Policy.EscalatePressureFromRound != 2 {
		t.Fatalf("expected default pressure round, got %d", strategy.Policy.EscalatePressureFromRound)
	}
}

func TestResolveStrategyConfigUsesWeaknessFocusedStrategy(t *testing.T) {
	t.Parallel()

	strategy := ResolveStrategyConfig(
		protocol.InterviewConfig{
			Mode:    protocol.ModeWeaknessFocused,
			Persona: protocol.PersonaSupportive,
		},
		protocol.SkillSpec{},
	)

	if strategy.Name != StrategyWeaknessFocused {
		t.Fatalf("expected weakness-focused strategy, got %q", strategy.Name)
	}
	if !strategy.Policy.PreferWeaknessFocus {
		t.Fatalf("expected weakness-focused policy to prefer weak areas")
	}
	if strategy.Policy.AdversarialFromPhase != protocol.PhaseAdversarial {
		t.Fatalf("expected supportive persona to delay adversarial phase, got %q", strategy.Policy.AdversarialFromPhase)
	}
}

func TestDecideNextStepStrategySelectionIsDeterministic(t *testing.T) {
	t.Parallel()

	cfg := protocol.InterviewConfig{
		Mode:    protocol.ModeWeaknessFocused,
		Persona: protocol.PersonaManager,
	}
	profile := protocol.CandidateProfile{RecurringGaps: []string{"observability"}}
	state := protocol.RunInterviewState{Phase: protocol.PhaseProbe, Round: 2}
	analysis := AnswerSignalAnalysis{
		WeakSignals:          []string{"missing_observability_detail"},
		WeakSignalConfidence: map[string]float64{"missing_observability_detail": 0.7},
	}
	skill := protocol.SkillSpec{FocusAreas: []string{"observability"}}

	left := DecideNextStep(profile, state, analysis, cfg, skill)
	right := DecideNextStep(profile, state, analysis, cfg, skill)

	if !reflect.DeepEqual(left, right) {
		t.Fatalf("expected deterministic decisions, got %#v vs %#v", left, right)
	}
}

func TestDecideNextStepSwitchesTopicForSystemDesignMode(t *testing.T) {
	t.Parallel()

	decision := DecideNextStep(
		protocol.CandidateProfile{},
		protocol.RunInterviewState{Phase: protocol.PhaseProbe, Round: 2},
		AnswerSignalAnalysis{
			StrongSignals:          []string{"tradeoff_reasoning", "implementation_detail"},
			StrongSignalConfidence: map[string]float64{"tradeoff_reasoning": 0.82, "implementation_detail": 0.88},
		},
		protocol.InterviewConfig{Mode: protocol.ModeSystemDesign},
		protocol.SkillSpec{},
	)

	if !decision.SwitchTopic {
		t.Fatalf("expected system design mode to switch topic after strong answer")
	}
	if decision.KeepTopic {
		t.Fatalf("did not expect keep topic when switching")
	}
	if !decision.IncreaseDifficulty {
		t.Fatalf("expected strong answer to increase difficulty")
	}
	if decision.Reason != protocol.ReasonTopicSwitch {
		t.Fatalf("expected topic switch reason, got %q", decision.Reason)
	}
}

func TestDecideNextStepUsesWrapupBudgetReason(t *testing.T) {
	t.Parallel()

	decision := DecideNextStep(
		protocol.CandidateProfile{},
		protocol.RunInterviewState{Phase: protocol.PhaseWrapup, Round: 5},
		AnswerSignalAnalysis{},
		protocol.InterviewConfig{Mode: protocol.ModeStandard, TimeBudget: "25 minutes"},
		protocol.SkillSpec{},
	)

	if decision.SwitchTopic {
		t.Fatalf("did not expect wrapup to switch topic")
	}
	if decision.KeepTopic {
		t.Fatalf("did not expect wrapup to keep current topic")
	}
	if !decision.EscalatePressure {
		t.Fatalf("expected wrapup phase to keep pressure on")
	}
	if decision.Reason != protocol.ReasonWrapupDueToBudget {
		t.Fatalf("expected wrapup reason, got %q", decision.Reason)
	}
}

func TestDecideNextStepAvoidsEarlyWrapupForLongBudget(t *testing.T) {
	t.Parallel()

	decision := DecideNextStep(
		protocol.CandidateProfile{},
		protocol.RunInterviewState{Phase: protocol.PhaseWrapup, Round: 3},
		AnswerSignalAnalysis{},
		protocol.InterviewConfig{Mode: protocol.ModeStandard, TimeBudget: "45 minutes"},
		protocol.SkillSpec{},
	)

	if decision.Reason == protocol.ReasonWrapupDueToBudget {
		t.Fatalf("did not expect early wrapup reason for long-budget interview: %#v", decision)
	}
}

func TestDecideNextStepAddsResumeDeepDiveFocus(t *testing.T) {
	t.Parallel()

	decision := DecideNextStep(
		protocol.CandidateProfile{},
		protocol.RunInterviewState{Phase: protocol.PhaseProbe, Round: 1},
		AnswerSignalAnalysis{},
		protocol.InterviewConfig{Mode: protocol.ModeResumeDeepDive},
		protocol.SkillSpec{},
	)

	if !decision.KeepTopic {
		t.Fatalf("expected resume deep dive to keep current topic")
	}
	if decision.Reason != protocol.ReasonProfileWeaknessFocus {
		t.Fatalf("expected resume deep dive to focus on configured follow-up areas, got %q", decision.Reason)
	}
	if !containsFocus(decision.RecommendedFocus, "resume evidence") {
		t.Fatalf("expected resume evidence focus, got %#v", decision.RecommendedFocus)
	}
	if !containsFocus(decision.RecommendedFocus, "ownership") {
		t.Fatalf("expected ownership focus, got %#v", decision.RecommendedFocus)
	}
	if !containsFocus(decision.RecommendedFocus, "decision tradeoffs") {
		t.Fatalf("expected decision tradeoffs focus, got %#v", decision.RecommendedFocus)
	}
}

func containsFocus(values []string, target string) bool {
	t := strings.TrimSpace(strings.ToLower(target))
	for _, value := range values {
		if strings.TrimSpace(strings.ToLower(value)) == t {
			return true
		}
	}
	return false
}
