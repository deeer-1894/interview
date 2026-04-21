package interview

import (
	"testing"

	"mockinterview/internal/protocol"
)

func TestDefaultRunInterviewStateSetsWarmupDefaults(t *testing.T) {
	t.Parallel()

	state := DefaultRunInterviewState()
	if state.Phase != protocol.PhaseWarmup {
		t.Fatalf("expected warmup phase, got %q", state.Phase)
	}
	if state.Round != 0 || state.Difficulty != 1 {
		t.Fatalf("unexpected default state: %#v", state)
	}
}

func TestAdvancePhaseForModeUsesModeSpecificBoundaries(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		mode     protocol.InterviewMode
		round    int
		expected protocol.InterviewPhase
	}{
		{name: "standard_probe", mode: protocol.ModeStandard, round: 2, expected: protocol.PhaseProbe},
		{name: "standard_stress", mode: protocol.ModeStandard, round: 5, expected: protocol.PhaseStress},
		{name: "stress_mode_advances_earlier", mode: protocol.ModeStress, round: 2, expected: protocol.PhaseAdversarial},
		{name: "weakness_mode_stays_in_probe_longer", mode: protocol.ModeWeaknessFocused, round: 3, expected: protocol.PhaseProbe},
		{name: "system_design_reaches_adversarial_early", mode: protocol.ModeSystemDesign, round: 2, expected: protocol.PhaseAdversarial},
		{name: "resume_deep_dive_preserves_probe", mode: protocol.ModeResumeDeepDive, round: 3, expected: protocol.PhaseProbe},
		{name: "wrapup_after_budget", mode: protocol.ModeStandard, round: 6, expected: protocol.PhaseWrapup},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			state := protocol.RunInterviewState{Round: tc.round}
			AdvancePhaseForMode(&state, 6, tc.mode)
			if state.Phase != tc.expected {
				t.Fatalf("expected phase %q, got %#v", tc.expected, state)
			}
		})
	}
}

func TestAdvancePhaseWrapperUsesStandardMode(t *testing.T) {
	t.Parallel()

	state := protocol.RunInterviewState{Round: 4}
	AdvancePhase(&state, 6)
	if state.Phase != protocol.PhaseAdversarial {
		t.Fatalf("expected wrapper to follow standard mode progression, got %#v", state)
	}
}

func TestAppendRoundSnapshotInitializesHistory(t *testing.T) {
	t.Parallel()

	state := &protocol.RunInterviewState{}
	AppendRoundSnapshot(state, protocol.InterviewRoundSnapshot{
		Round:       1,
		Phase:       protocol.PhaseProbe,
		Difficulty:  2,
		Explanation: "继续追问实现细节",
	})

	if len(state.History) != 1 {
		t.Fatalf("expected one snapshot, got %#v", state.History)
	}
	if state.History[0].Explanation != "继续追问实现细节" {
		t.Fatalf("unexpected snapshot contents: %#v", state.History[0])
	}
	if state.Phase != protocol.PhaseWarmup || state.Difficulty != 1 {
		t.Fatalf("expected EnsureRunInterviewState defaults to be preserved, got %#v", state)
	}
}

func TestTopWeakAreasDedupesAndFallsBackToDimensions(t *testing.T) {
	t.Parallel()

	profile := protocol.CandidateProfile{
		RecommendedFocus: []string{"observability", "observability"},
		RecurringGaps:    []string{},
		Dimensions: []protocol.ProfileDimension{
			{Key: "resilience", Label: "弹性设计", Score: 0},
			{Key: "ownership", Label: "OwnerShip", Score: 0},
		},
	}

	areas := TopWeakAreas(profile)
	if len(areas) != 3 {
		t.Fatalf("expected top weak areas to be capped at 3, got %#v", areas)
	}
	if areas[0] != "observability" || areas[1] != "resilience" || areas[2] != "弹性设计" {
		t.Fatalf("unexpected weak area ordering: %#v", areas)
	}
}
