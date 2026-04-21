package lifecycle

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	workflowpkg "mockinterview/internal/control/workflow"
	domain "mockinterview/internal/interview"
	"mockinterview/internal/protocol"
)

type PlanningInput struct {
	Prompt       string
	Config       protocol.InterviewConfig
	State        protocol.RunInterviewState
	Profile      protocol.CandidateProfile
	Skill        protocol.SkillSpec
	Artifacts    []protocol.Artifact
	LatestAnswer string
}

type PlanningOutcome struct {
	Prompt        string
	PromptVersion string
	State         protocol.RunInterviewState
	Profile       protocol.CandidateProfile
	Skill         protocol.SkillSpec
	Plan          protocol.ExecutionPlan
	Analysis      domain.AnswerSignalAnalysis
	Decision      protocol.NextStepDecision
	ProfileFocus  []string
	FocusAreas    []string
}

type PlanningEventInput struct {
	Run           protocol.Run
	Task          protocol.Task
	PromptVersion string
	Outcome       PlanningOutcome
	Timestamp     time.Time
}

func PlanTurn(input PlanningInput) PlanningOutcome {
	cfg := input.Config.WithDefaults()
	state := domain.EnsureRunInterviewState(&input.State)
	analysis := domain.AnalyzeAnswerSignals(strings.TrimSpace(input.LatestAnswer), input.Skill)
	promptVersion := domain.InterviewPromptVersion
	decision := domain.DecideNextStep(input.Profile, state, analysis, cfg, input.Skill)
	state.LastDecision = &decision

	profileFocus := ResolveProfileFocus(input.Profile)
	focusAreas := ResolvePlanningFocus(input.Profile, decision, cfg.Mode)
	skill := input.Skill
	if len(focusAreas) > 0 {
		skill = domain.ApplyFocusConstraints(skill, focusAreas)
		skill = domain.ConstrainSkillSpecForDecision(skill, focusAreas, cfg.Mode)
	}

	prompt := ComposePlanningPrompt(strings.TrimSpace(input.Prompt), decision, focusAreas)
	plan := BuildExecutionPlan(cfg, input.Profile, input.Artifacts)

	return PlanningOutcome{
		Prompt:        prompt,
		PromptVersion: promptVersion,
		State:         state,
		Profile:       input.Profile,
		Skill:         skill,
		Plan:          plan,
		Analysis:      analysis,
		Decision:      decision,
		ProfileFocus:  profileFocus,
		FocusAreas:    focusAreas,
	}
}

func BuildPlanningEvents(input PlanningEventInput) (protocol.Event, protocol.Event) {
	timestamp := input.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	decisionEvent := protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: input.Run.ConversationID,
		TaskID:         input.Run.TaskID,
		RunID:          input.Run.ID,
		Type:           protocol.EventDecisionGenerated,
		Timestamp:      timestamp,
		Payload: protocol.DecisionAudit{
			State:         input.Outcome.State,
			Mode:          input.Task.Config.Mode,
			PromptVersion: input.PromptVersion,
			Analysis:      input.Outcome.Analysis.Snapshot(),
			ProfileFocus:  append([]string(nil), input.Outcome.ProfileFocus...),
			Decision:      input.Outcome.Decision,
		},
	}

	planEvent := protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: input.Run.ConversationID,
		TaskID:         input.Run.TaskID,
		RunID:          input.Run.ID,
		Type:           protocol.EventPlanGenerated,
		Timestamp:      timestamp,
		Payload:        input.Outcome.Plan,
	}

	return decisionEvent, planEvent
}

func ResolveProfileFocus(profile protocol.CandidateProfile) []string {
	values := append([]string(nil), profile.RecommendedFocus...)
	values = append(values, profile.RecurringGaps...)
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

func ResolvePlanningFocus(
	profile protocol.CandidateProfile,
	decision protocol.NextStepDecision,
	mode protocol.InterviewMode,
) []string {
	return mergePlanningFocus(
		ResolveProfileFocus(profile),
		decision.RecommendedFocus,
		modeDefaultFocus(mode),
	)
}

func ComposePlanningPrompt(prompt string, decision protocol.NextStepDecision, focusAreas []string) string {
	prompt = buildProfileAwarePrompt(prompt, focusAreas)
	return buildDecisionAwarePrompt(prompt, decision)
}

func BuildExecutionPlan(
	cfg protocol.InterviewConfig,
	profile protocol.CandidateProfile,
	artifacts []protocol.Artifact,
) protocol.ExecutionPlan {
	steps := make([]protocol.PlanStep, 0, 6)
	steps = append(steps, protocol.PlanStep{
		ID:          "frame",
		Title:       "Frame interview scenario",
		Description: fmt.Sprintf("Calibrate to %s level in %s mode for %s focus.", cfg.Level, cfg.Mode, cfg.Focus),
		Kind:        "interview_setup",
	})

	if cfg.EnableWebSearch {
		steps = append(steps, workflowpkg.ResearchWorkflow{}.Build()...)
	}

	steps = append(steps, protocol.PlanStep{
		ID:          "questions",
		Title:       "Run interview turns",
		Description: "Ask one question at a time, adapt follow-ups, and test depth against the resolved skill pack.",
		Kind:        "interview_loop",
	})
	if focusAreas := ResolveProfileFocus(profile); len(focusAreas) > 0 {
		steps = append(steps, protocol.PlanStep{
			ID:          "profile_focus",
			Title:       "Target historical weak areas",
			Description: "Bias the interview toward prior weak signals: " + strings.Join(focusAreas, ", "),
			Kind:        "profile_feedback",
		})
	}

	if len(artifacts) > 0 {
		steps = append(steps, workflowpkg.ArtifactIngestWorkflow{}.Build()...)
	}

	steps = append(steps, protocol.PlanStep{
		ID:          "score",
		Title:       "Generate evaluation",
		Description: "Produce structured scoring aligned with the resolved rubric.",
		Kind:        "evaluation",
	})

	if cfg.OutputStyle == protocol.OutputInterviewPlusStudy {
		steps = append(steps, protocol.PlanStep{
			ID:          "study_plan",
			Title:       "Generate study plan",
			Description: "Emit a follow-on study plan tied to the gaps and improvements from the scorecard.",
			Kind:        "study_plan",
		})
	}

	return protocol.ExecutionPlan{
		Title: fmt.Sprintf("%s interview execution plan", strings.ToUpper(firstNonEmptyPrompt(cfg.Skill, "default"))),
		Steps: steps,
	}
}

func buildProfileAwarePrompt(prompt string, focusAreas []string) string {
	prompt = strings.TrimSpace(prompt)
	if len(focusAreas) == 0 {
		return prompt
	}
	var b strings.Builder
	b.WriteString(prompt)
	b.WriteString("\n\nHistorical weak areas to prioritize in this interview:\n")
	for _, area := range focusAreas {
		b.WriteString("- ")
		b.WriteString(area)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func buildDecisionAwarePrompt(prompt string, decision protocol.NextStepDecision) string {
	prompt = strings.TrimSpace(prompt)
	if decision.Reason == "" && len(decision.RecommendedFocus) == 0 && strings.TrimSpace(decision.Explanation) == "" {
		return prompt
	}
	var b strings.Builder
	b.WriteString(prompt)
	b.WriteString("\n\nCurrent decision policy:\n")
	if decision.Reason != "" {
		b.WriteString("- decision reason: ")
		b.WriteString(string(decision.Reason))
		b.WriteString("\n")
	}
	if decision.KeepTopic {
		b.WriteString("- keep the topic and continue probing the same weak area\n")
	}
	if decision.SwitchTopic {
		b.WriteString("- switch to the next topic after confirming the current answer\n")
	}
	if decision.EscalatePressure {
		b.WriteString("- tighten the response budget and increase pressure\n")
	}
	if decision.TriggerAdversarial {
		b.WriteString("- use a stronger adversarial challenge on the next turn\n")
	}
	if text := strings.TrimSpace(decision.Explanation); text != "" {
		b.WriteString("- decision explanation: ")
		b.WriteString(text)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func mergePlanningFocus(groups ...[]string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, group := range groups {
		for _, value := range group {
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
		}
	}
	if len(out) > 4 {
		return out[:4]
	}
	return out
}

func modeDefaultFocus(mode protocol.InterviewMode) []string {
	switch domain.NormalizeInterviewMode(string(mode)) {
	case protocol.ModeStress:
		return []string{"tradeoff_expression", "decision_speed"}
	case protocol.ModeWeaknessFocused:
		return []string{"historical_weak_areas"}
	case protocol.ModeSystemDesign:
		return []string{"system design", "reliability", "observability"}
	case protocol.ModeResumeDeepDive:
		return []string{"resume evidence", "ownership", "impact"}
	default:
		return nil
	}
}

func firstNonEmptyPrompt(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
