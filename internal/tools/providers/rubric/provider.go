package rubric

import (
	"context"
	"fmt"
	"strings"

	domain "mockinterview/internal/interview"
	"mockinterview/internal/protocol"
)

type Provider struct{}

func New() *Provider {
	return &Provider{}
}

func (p *Provider) Resolve(ctx context.Context, cfg protocol.InterviewConfig) (protocol.Rubric, error) {
	_ = ctx

	name := domain.ResolveSkillName(domain.InterviewConfig{
		Skill:        cfg.Skill,
		SkillFocuses: append([]string(nil), cfg.SkillFocuses...),
		Level:        cfg.Level,
		Focus:        cfg.Focus,
		Mode:         domain.Mode(cfg.Mode),
		TimeBudget:   cfg.TimeBudget,
		OutputStyle:  domain.OutputStyle(cfg.OutputStyle),
	})
	meta, err := domain.LoadSkillMetadata(name)
	if err != nil {
		return protocol.Rubric{}, err
	}

	focusAreas := domain.ResolveInterviewFocusAreas(domain.InterviewConfig{
		Skill:        cfg.Skill,
		SkillFocuses: append([]string(nil), cfg.SkillFocuses...),
		Focus:        cfg.Focus,
	})
	focusLabel := "generalist"
	if len(focusAreas) > 0 {
		focusLabel = strings.Join(focusAreas, ", ")
	}

	return protocol.Rubric{
		Title: fmt.Sprintf("%s rubric", meta.Name),
		Anchors: []string{
			fmt.Sprintf("Target level fit: %s", cfg.Level),
			fmt.Sprintf("Active skill focus: %s", focusLabel),
			"Question depth, follow-up quality, and tradeoff rigor",
			"Operational realism: reliability, observability, and safety judgment",
			"Actionable feedback quality and calibration",
		},
	}, nil
}
