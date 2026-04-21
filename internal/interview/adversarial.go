package interview

import (
	"strings"

	"mockinterview/internal/protocol"
)

func ShouldTriggerAdversarial(state protocol.RunInterviewState, analysis AnswerSignalAnalysis) bool {
	state = EnsureRunInterviewState(&state)
	if state.Phase == protocol.PhaseWarmup {
		return false
	}
	if state.Phase == protocol.PhaseAdversarial || state.Phase == protocol.PhaseStress {
		return true
	}
	if analysis.TooGeneric {
		return true
	}
	if !analysis.HasTradeoff {
		return true
	}
	return containsSignal(state.WeakSignals, "missing_tradeoff") || containsSignal(state.WeakSignals, "too_generic")
}

func SelectAdversarialPrompt(skill protocol.SkillSpec, state protocol.RunInterviewState) string {
	state = EnsureRunInterviewState(&state)
	if len(skill.Adversarial) == 0 {
		return ""
	}
	index := state.Round % len(skill.Adversarial)
	return strings.TrimSpace(skill.Adversarial[index])
}

func SelectPressurePrompt(skill protocol.SkillSpec, state protocol.RunInterviewState) string {
	state = EnsureRunInterviewState(&state)
	if state.Phase != protocol.PhaseStress && state.Phase != protocol.PhaseWrapup {
		return ""
	}
	if len(skill.Pressure) == 0 {
		return ""
	}
	index := state.Round % len(skill.Pressure)
	return strings.TrimSpace(skill.Pressure[index])
}

func SelectScenario(skill protocol.SkillSpec, state protocol.RunInterviewState) string {
	state = EnsureRunInterviewState(&state)
	if len(skill.Scenarios) == 0 {
		return ""
	}
	index := state.Round % len(skill.Scenarios)
	return strings.TrimSpace(skill.Scenarios[index])
}

func containsSignal(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}
