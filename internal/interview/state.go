package interview

import (
	"strings"

	"mockinterview/internal/protocol"
)

type InterviewPhase = protocol.InterviewPhase

const (
	PhaseWarmup      = protocol.PhaseWarmup
	PhaseProbe       = protocol.PhaseProbe
	PhaseAdversarial = protocol.PhaseAdversarial
	PhaseStress      = protocol.PhaseStress
	PhaseWrapup      = protocol.PhaseWrapup
)

type RunInterviewState = protocol.RunInterviewState
type InterviewRoundSnapshot = protocol.InterviewRoundSnapshot

func DefaultRunInterviewState() protocol.RunInterviewState {
	return protocol.RunInterviewState{
		Phase:      protocol.PhaseWarmup,
		Round:      0,
		Difficulty: 1,
	}
}

func EnsureRunInterviewState(state *protocol.RunInterviewState) protocol.RunInterviewState {
	if state == nil {
		return DefaultRunInterviewState()
	}
	normalized := *state
	if normalized.Phase == "" {
		normalized.Phase = protocol.PhaseWarmup
	}
	if normalized.Difficulty <= 0 {
		normalized.Difficulty = 1
	}
	if normalized.WeakSignals == nil {
		normalized.WeakSignals = []string{}
	}
	if normalized.StrongSignals == nil {
		normalized.StrongSignals = []string{}
	}
	if normalized.History == nil {
		normalized.History = []protocol.InterviewRoundSnapshot{}
	}
	return normalized
}

func AdvancePhase(state *protocol.RunInterviewState, maxRounds int) {
	AdvancePhaseForMode(state, maxRounds, protocol.ModeStandard)
}

func AdvancePhaseForMode(state *protocol.RunInterviewState, maxRounds int, mode protocol.InterviewMode) {
	if state == nil {
		return
	}
	*state = EnsureRunInterviewState(state)
	if maxRounds < 5 {
		maxRounds = 5
	}
	mode = NormalizeInterviewMode(string(mode))
	probeUntil := maxInt(2, maxRounds/3)
	adversarialUntil := maxInt(probeUntil+1, (maxRounds*2)/3)
	stressUntil := maxInt(adversarialUntil+1, maxRounds-1)

	switch mode {
	case protocol.ModeStress:
		probeUntil = maxInt(1, maxRounds/4)
		adversarialUntil = maxInt(probeUntil+1, maxRounds/2)
		stressUntil = maxInt(adversarialUntil+1, maxRounds-1)
	case protocol.ModeWeaknessFocused:
		probeUntil = maxInt(2, maxRounds/2)
		adversarialUntil = maxInt(probeUntil+1, (maxRounds*4)/5)
	case protocol.ModeSystemDesign:
		probeUntil = maxInt(1, maxRounds/4)
		adversarialUntil = maxInt(probeUntil+1, (maxRounds*3)/4)
	case protocol.ModeResumeDeepDive:
		probeUntil = maxInt(2, maxRounds/2)
		adversarialUntil = maxInt(probeUntil+1, (maxRounds*3)/4)
	}
	switch {
	case state.Round <= 1:
		state.Phase = protocol.PhaseWarmup
	case state.Round <= probeUntil:
		state.Phase = protocol.PhaseProbe
	case state.Round <= adversarialUntil:
		state.Phase = protocol.PhaseAdversarial
	case state.Round <= stressUntil:
		state.Phase = protocol.PhaseStress
	default:
		state.Phase = protocol.PhaseWrapup
	}
}

func AppendRoundSnapshot(state *protocol.RunInterviewState, snapshot protocol.InterviewRoundSnapshot) {
	if state == nil {
		return
	}
	*state = EnsureRunInterviewState(state)
	state.History = append(state.History, snapshot)
}

func TopWeakAreas(profile protocol.CandidateProfile) []string {
	values := make([]string, 0, len(profile.RecommendedFocus)+len(profile.RecurringGaps))
	values = append(values, profile.RecommendedFocus...)
	values = append(values, profile.RecurringGaps...)
	if len(values) < 3 {
		for _, dimension := range profile.Dimensions {
			if dimension.Score <= 0 {
				values = append(values, dimension.Key)
				if strings.TrimSpace(dimension.Label) != "" {
					values = append(values, dimension.Label)
				}
			}
		}
	}
	out := make([]string, 0, 3)
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
		if len(out) == 3 {
			break
		}
	}
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
