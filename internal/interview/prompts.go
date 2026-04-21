package interview

import (
	"strings"

	"mockinterview/internal/protocol"
)

func BuildInterviewerInstruction(cfg InterviewConfig, runPhase protocol.RunPhase, state protocol.RunInterviewState, skill protocol.SkillSpec) string {
	return BuildPromptStrategy(cfg, runPhase, state, skill).Instruction()
}

func phaseInstructions(phase protocol.InterviewPhase) []string {
	switch phase {
	case protocol.PhaseWarmup:
		return []string{
			"Establish the scenario and allow the candidate to open with a fuller explanation.",
			"Do not pressure too early. Confirm the candidate's baseline understanding first.",
		}
	case protocol.PhaseProbe:
		return []string{
			"Probe implementation details, boundary conditions, operational concerns, and tradeoffs.",
			"Ask for concrete mechanisms instead of accepting high-level answers.",
		}
	case protocol.PhaseAdversarial:
		return []string{
			"Actively challenge assumptions and surface the most fragile part of the proposal.",
			"Use counterexamples and production edge cases when the answer sounds only superficially correct.",
		}
	case protocol.PhaseStress:
		return []string{
			"Compress the answer space and force prioritization.",
			"Prefer the shortest question that reveals decision quality under time pressure.",
		}
	case protocol.PhaseWrapup:
		return []string{
			"Stop expanding the scope and ask a final closing question only if needed.",
			"Prepare to end the interview and transition toward evaluation.",
		}
	default:
		return nil
	}
}

func skillPackInstructions(skill protocol.SkillSpec, state protocol.RunInterviewState) []string {
	lines := make([]string, 0, 6)
	if len(skill.FocusAreas) > 0 {
		lines = append(lines, "Keep the questioning centered on these focus areas: "+compactPromptList(skill.FocusAreas, 3)+".")
	}
	if scenario := SelectScenario(skill, state); scenario != "" && state.Phase != protocol.PhaseWarmup {
		lines = append(lines, "Use or reference this scenario when useful: "+compactPromptText(scenario, 180))
	}
	if state.LastDecision != nil {
		if reason := strings.TrimSpace(string(state.LastDecision.Reason)); reason != "" {
			lines = append(lines, "Current decision reason: "+reason)
		}
		if explanation := strings.TrimSpace(state.LastDecision.Explanation); explanation != "" {
			lines = append(lines, "Current decision explanation: "+compactPromptText(explanation, 180))
		}
		if len(state.LastDecision.RecommendedFocus) > 0 {
			lines = append(lines, "Bias the next question toward: "+compactPromptList(state.LastDecision.RecommendedFocus, 3))
		}
	}
	if prompt := SelectAdversarialPrompt(skill, state); prompt != "" && state.Phase == protocol.PhaseAdversarial {
		lines = append(lines, "When challenging the candidate, prefer prompts like: "+compactPromptText(prompt, 160))
	}
	if prompt := SelectPressurePrompt(skill, state); prompt != "" && (state.Phase == protocol.PhaseStress || state.Phase == protocol.PhaseWrapup) {
		lines = append(lines, "When time is tight, use pressure prompts like: "+compactPromptText(prompt, 160))
	}
	return lines
}

func firstNonEmptyPromptValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
