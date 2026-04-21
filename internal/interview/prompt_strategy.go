package interview

import (
	"fmt"
	"strings"

	"mockinterview/internal/protocol"
)

const InterviewPromptVersion = "interviewer.v2"

type PromptLayer string

const (
	PromptLayerSystem  PromptLayer = "system"
	PromptLayerContext PromptLayer = "context"
	PromptLayerSkill   PromptLayer = "skill"
)

type PromptSection struct {
	Layer   PromptLayer
	Name    string
	Content string
}

type PromptStrategy struct {
	Version  string
	Sections []PromptSection
}

func BuildPromptStrategy(
	cfg InterviewConfig,
	runPhase protocol.RunPhase,
	state protocol.RunInterviewState,
	skill protocol.SkillSpec,
) PromptStrategy {
	cfg = cfg.WithDefaults()
	state = EnsureRunInterviewState(&state)
	if runPhase == "" {
		runPhase = protocol.RunPhaseInitial
	}

	requestedSkill := ResolveSkillName(cfg)
	sections := []PromptSection{
		{Layer: PromptLayerSystem, Name: "identity", Content: buildIdentityPromptSection(cfg)},
		{Layer: PromptLayerSystem, Name: "turn_contract", Content: buildTurnContractPromptSection(runPhase, cfg.OutputStyle)},
		{Layer: PromptLayerContext, Name: "skill_routing", Content: buildSkillRoutingPromptSection(cfg, requestedSkill)},
		{Layer: PromptLayerContext, Name: "interview_setup", Content: buildInterviewSetupPromptSection(cfg, state, requestedSkill)},
		{Layer: PromptLayerContext, Name: "behavior", Content: buildBehaviorPromptSection(state)},
		{Layer: PromptLayerSkill, Name: "skill_pack", Content: buildSkillPromptSection(state, skill)},
	}

	return PromptStrategy{
		Version:  InterviewPromptVersion,
		Sections: compactPromptSections(sections),
	}
}

func (s PromptStrategy) Instruction() string {
	return strings.TrimSpace(strings.Join(sectionContents(s.Sections), "\n\n"))
}

func compactPromptSections(sections []PromptSection) []PromptSection {
	out := make([]PromptSection, 0, len(sections))
	for _, section := range sections {
		section.Content = strings.TrimSpace(section.Content)
		if section.Content == "" {
			continue
		}
		out = append(out, section)
	}
	return out
}

func sectionContents(sections []PromptSection) []string {
	out := make([]string, 0, len(sections))
	for _, section := range sections {
		if strings.TrimSpace(section.Content) == "" {
			continue
		}
		out = append(out, section.Content)
	}
	return out
}

func compactPromptList(items []string, limit int) string {
	items = normalizeBulletList(items)
	if len(items) == 0 {
		return ""
	}
	if limit <= 0 || len(items) <= limit {
		return strings.Join(items, ", ")
	}
	head := strings.Join(items[:limit], ", ")
	return fmt.Sprintf("%s, and %d more", head, len(items)-limit)
}

func compactPromptText(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	if value == "" || maxRunes <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return strings.TrimSpace(string(runes[:maxRunes])) + "..."
}
