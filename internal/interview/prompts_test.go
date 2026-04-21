package interview

import (
	"strings"
	"testing"

	"mockinterview/internal/protocol"
)

func TestBuildPromptSectionsProducesOrderedBlocks(t *testing.T) {
	t.Parallel()

	cfg := InterviewConfig{
		Mode:        ModeStress,
		OutputStyle: OutputInterviewPlusScore,
		Level:       "senior",
		Focus:       "agent runtime",
		TimeBudget:  "25 minutes",
	}
	state := protocol.RunInterviewState{
		Phase:      protocol.PhaseProbe,
		Difficulty: 3,
	}
	sections := buildPromptSections(cfg, protocol.RunPhaseInitial, state, protocol.SkillSpec{FocusAreas: []string{"observability"}})

	if len(sections) != 6 {
		t.Fatalf("expected 6 sections, got %d", len(sections))
	}
	if !strings.Contains(sections[0], "rigorous mock interviewer") {
		t.Fatalf("expected identity section first, got %q", sections[0])
	}
	if !strings.Contains(sections[1], "first interview turn") {
		t.Fatalf("expected turn contract section second, got %q", sections[1])
	}
	if !strings.Contains(sections[2], "skill tool") {
		t.Fatalf("expected skill routing section third, got %q", sections[2])
	}
	if !strings.Contains(sections[3], "Interview setup:") {
		t.Fatalf("expected setup section fourth, got %q", sections[3])
	}
	if !strings.Contains(sections[4], "Behavior rules:") {
		t.Fatalf("expected behavior section fifth, got %q", sections[4])
	}
	if !strings.Contains(sections[5], "Skill context:") {
		t.Fatalf("expected skill section sixth, got %q", sections[5])
	}
}

func TestBuildInterviewerInstructionIncludesDecisionExplanation(t *testing.T) {
	t.Parallel()

	instruction := BuildInterviewerInstruction(
		InterviewConfig{
			Mode:        ModeWeaknessFocused,
			OutputStyle: OutputInterviewPlusScore,
			Level:       "mid",
			Focus:       "go agent",
			TimeBudget:  "25 minutes",
		},
		protocol.RunPhaseInterviewing,
		protocol.RunInterviewState{
			Phase: protocol.PhaseProbe,
			LastDecision: &protocol.NextStepDecision{
				Reason:             protocol.ReasonProfileWeaknessFocus,
				Explanation:        "系统命中了历史弱项，下一问会继续围绕 observability 深挖。",
				RecommendedFocus:   []string{"observability"},
				KeepTopic:          true,
				TriggerAdversarial: false,
			},
		},
		protocol.SkillSpec{FocusAreas: []string{"observability", "reliability"}},
	)

	if !strings.Contains(instruction, "latest decision explanation") {
		t.Fatalf("expected setup block to include decision explanation, got %q", instruction)
	}
	if !strings.Contains(instruction, "Current decision explanation") {
		t.Fatalf("expected behavior block to include decision explanation, got %q", instruction)
	}
	if !strings.Contains(instruction, "observability") {
		t.Fatalf("expected instruction to retain focus bias, got %q", instruction)
	}
}

func TestBuildPromptStrategyIncludesVersionAndLayerOrdering(t *testing.T) {
	t.Parallel()

	strategy := BuildPromptStrategy(
		InterviewConfig{
			Mode:        ModeStandard,
			OutputStyle: OutputInterviewPlusScore,
			Level:       "senior",
			Focus:       "agent runtime",
			TimeBudget:  "25 minutes",
		},
		protocol.RunPhaseInterviewing,
		protocol.RunInterviewState{Phase: protocol.PhaseProbe},
		protocol.SkillSpec{FocusAreas: []string{"observability", "reliability"}},
	)

	if strategy.Version != InterviewPromptVersion {
		t.Fatalf("expected prompt version %q, got %q", InterviewPromptVersion, strategy.Version)
	}
	if len(strategy.Sections) != 6 {
		t.Fatalf("expected 6 prompt sections, got %d", len(strategy.Sections))
	}
	if strategy.Sections[0].Layer != PromptLayerSystem || strategy.Sections[1].Layer != PromptLayerSystem {
		t.Fatalf("expected first two sections to be system-layer, got %#v", strategy.Sections[:2])
	}
	if strategy.Sections[5].Layer != PromptLayerSkill {
		t.Fatalf("expected final section to be skill-layer, got %q", strategy.Sections[5].Layer)
	}
}

func TestBuildPromptStrategyCompressesLongLists(t *testing.T) {
	t.Parallel()

	instruction := BuildInterviewerInstruction(
		InterviewConfig{
			Mode:        ModeStandard,
			OutputStyle: OutputInterviewPlusScore,
			Level:       "mid",
			Focus:       "go agent",
			TimeBudget:  "25 minutes",
		},
		protocol.RunPhaseInterviewing,
		protocol.RunInterviewState{
			Phase:       protocol.PhaseProbe,
			WeakSignals: []string{"missing_tradeoff", "too_generic", "missing_timeout_detail", "missing_observability_detail", "partial_answer"},
		},
		protocol.SkillSpec{FocusAreas: []string{"observability", "reliability", "system design", "ownership"}},
	)

	if !strings.Contains(instruction, "and 1 more") {
		t.Fatalf("expected long prompt lists to be compressed, got %q", instruction)
	}
}
