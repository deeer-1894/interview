package skill

import (
	"context"

	domain "mockinterview/internal/interview"
	"mockinterview/internal/protocol"
)

type Provider struct{}

func New() *Provider {
	return &Provider{}
}

func (p *Provider) Resolve(ctx context.Context, cfg protocol.InterviewConfig) (protocol.SkillSpec, error) {
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
	pack, err := domain.LoadSkillPack(name)
	if err != nil {
		return protocol.SkillSpec{}, err
	}

	focusAreas := domain.ResolveInterviewFocusAreas(domain.InterviewConfig{
		Skill:        cfg.Skill,
		SkillFocuses: append([]string(nil), cfg.SkillFocuses...),
		Focus:        cfg.Focus,
	})
	spec := domain.ApplyFocusConstraints(pack.ToSkillSpec(), focusAreas)
	spec.FocusAreas = append(spec.FocusAreas,
		"Run a "+string(cfg.Mode)+" interview calibrated to "+cfg.Level+" level",
		"Use role focus "+cfg.Focus+" as the interview lens",
	)
	return spec, nil
}
