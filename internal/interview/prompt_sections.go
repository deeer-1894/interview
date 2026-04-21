package interview

import (
	"fmt"
	"strings"

	"mockinterview/internal/protocol"
)

func buildPromptSections(
	cfg InterviewConfig,
	runPhase protocol.RunPhase,
	state protocol.RunInterviewState,
	skill protocol.SkillSpec,
) []string {
	return sectionContents(BuildPromptStrategy(cfg, runPhase, state, skill).Sections)
}

func joinPromptSections(sections []string) string {
	return strings.TrimSpace(strings.Join(sections, "\n\n"))
}

func buildIdentityPromptSection(cfg InterviewConfig) string {
	lines := []string{
		"You are a rigorous mock interviewer for agent engineering roles.",
		strings.TrimSpace(BuildPersonaInstruction(cfg.Persona)),
		strings.TrimSpace(ModeInstruction(protocol.InterviewMode(cfg.Mode))),
		"Run the interview interactively.",
		"Ask one question at a time and wait for the candidate before continuing.",
		"Prefer scenario-based prompts over textbook trivia.",
		"Use follow-up questions to test depth, tradeoffs, production judgment, and failure handling.",
		"Keep your bar calibrated to the configured target level.",
		"Do not attempt the whole interview in one response.",
		"Your normal-turn output must contain exactly one main question.",
		"Do not ask multiple sub-questions in the same turn.",
		"Do not output numbered questions, bullet questions, or a list of prompts.",
		"Do not end a turn with more than one question mark.",
		"If you need to probe multiple dimensions, pick the single highest-value next question first.",
	}
	return strings.Join(filterPromptLines(lines), "\n")
}

func buildTurnContractPromptSection(runPhase protocol.RunPhase, style OutputStyle) string {
	lines := make([]string, 0, 4)
	if runPhase == protocol.RunPhaseInitial {
		lines = append(lines, "For the first interview turn, produce one short framing sentence and exactly one substantive first question.")
	} else {
		lines = append(lines,
			"For a follow-up interview turn, continue from the existing conversation state instead of restarting.",
			"Do not repeat an already asked question unless the user explicitly asked for repetition.",
			"Prefer one follow-up or one next question based on the user's latest answer.",
		)
	}
	if style == OutputInterviewOnly {
		lines = append(lines, "Do not generate scoring or a study plan unless the user explicitly asks for evaluation.")
	} else {
		lines = append(lines, "Only generate scoring or a study plan when the interview is explicitly ending or the user asks for evaluation.")
	}
	return strings.Join(filterPromptLines(lines), "\n")
}

func buildSkillRoutingPromptSection(cfg InterviewConfig, requestedSkill string) string {
	lines := []string{
		"Before asking the first question, use the skill tool to load the most relevant interview skill.",
	}
	if strings.TrimSpace(cfg.Skill) != "" {
		lines = append(lines, fmt.Sprintf("The configured interview skill is %q. Load that exact skill unless the user explicitly asks to change it.", cfg.Skill))
	} else {
		lines = append(lines, fmt.Sprintf("If no better match is obvious, start from the default skill %q.", requestedSkill))
	}
	lines = append(lines, "Do not invent a skill name that is not exposed by the skill tool.")
	return strings.Join(filterPromptLines(lines), "\n")
}

func buildInterviewSetupPromptSection(cfg InterviewConfig, state protocol.RunInterviewState, requestedSkill string) string {
	lines := []string{
		"Interview setup:",
		fmt.Sprintf("- requested skill: %s", firstNonEmptyPromptValue(cfg.Skill, requestedSkill)),
		fmt.Sprintf("- interviewer persona: %s", cfg.Persona),
		fmt.Sprintf("- target level: %s", cfg.Level),
		fmt.Sprintf("- role focus: %s", cfg.Focus),
		fmt.Sprintf("- interview mode: %s", cfg.Mode),
		fmt.Sprintf("- time budget: %s", cfg.TimeBudget),
		fmt.Sprintf("- output style: %s", cfg.OutputStyle),
		fmt.Sprintf("- interview phase: %s", state.Phase),
	}
	if state.Difficulty > 0 {
		lines = append(lines, fmt.Sprintf("- pressure level: %d", state.Difficulty))
	}
	if state.LastDecision != nil {
		lines = append(lines, fmt.Sprintf("- latest decision reason: %s", state.LastDecision.Reason))
		if strings.TrimSpace(state.LastDecision.Explanation) != "" {
			lines = append(lines, fmt.Sprintf("- latest decision explanation: %s", compactPromptText(state.LastDecision.Explanation, 180)))
		}
		if len(state.LastDecision.RecommendedFocus) > 0 {
			lines = append(lines, fmt.Sprintf("- next focus bias: %s", compactPromptList(state.LastDecision.RecommendedFocus, 3)))
		}
	}
	if len(state.WeakSignals) > 0 {
		lines = append(lines, fmt.Sprintf("- candidate weak signals: %s", compactPromptList(state.WeakSignals, 4)))
	}
	if len(state.StrongSignals) > 0 {
		lines = append(lines, fmt.Sprintf("- candidate strong signals: %s", compactPromptList(state.StrongSignals, 4)))
	}
	return strings.Join(filterPromptLines(lines), "\n")
}

func buildBehaviorPromptSection(state protocol.RunInterviewState) string {
	lines := []string{"Behavior rules:"}
	for _, line := range phaseInstructions(state.Phase) {
		lines = append(lines, "- "+line)
	}
	lines = append(lines,
		"- Keep each turn compact enough to stream quickly.",
		"- Prefer one strong question over multiple weaker questions.",
		"- Normal interview turns should usually be 1 short transition sentence plus 1 question sentence.",
		"- Never include a second question sentence in the same turn.",
		"- Avoid reusing near-duplicate question stems across consecutive turns; if you stay on the same topic, change the angle and ask for a different concrete artifact.",
		"- Avoid phrasing like 'please also explain A, B, and C' in one prompt.",
		"- Do not dump the whole rubric before the interview.",
		"- Avoid long lectures; keep turns interview-like.",
		"- If the user asks for only scoring or a study plan, adapt accordingly.",
	)
	return strings.Join(filterPromptLines(lines), "\n")
}

func buildSkillPromptSection(state protocol.RunInterviewState, skill protocol.SkillSpec) string {
	items := skillPackInstructions(skill, state)
	if len(items) == 0 {
		return ""
	}
	lines := []string{"Skill context:"}
	for _, line := range items {
		lines = append(lines, "- "+line)
	}
	return strings.Join(filterPromptLines(lines), "\n")
}

func filterPromptLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}
