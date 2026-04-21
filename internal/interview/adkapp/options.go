package adkapp

import (
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"

	"mockinterview/internal/protocol"
)

type SharedSessionContext struct {
	RunID          string
	PromptVersion  string
	RunPhase       protocol.RunPhase
	InterviewState protocol.RunInterviewState
	Skill          protocol.SkillSpec
	Rubric         protocol.Rubric
	Transcript     string
}

type Options struct {
	CheckPointStore adk.CheckPointStore
	RunPhase        protocol.RunPhase
	InterviewState  protocol.RunInterviewState
	Skill           protocol.SkillSpec
	SharedContext   SharedSessionContext
}

func (o Options) ResolveSharedContext() SharedSessionContext {
	shared := o.SharedContext
	if shared.RunPhase == "" {
		shared.RunPhase = o.RunPhase
	}
	if shared.Skill.Name == "" && len(shared.Skill.FocusAreas) == 0 {
		shared.Skill = o.Skill
	}
	if shared.InterviewState.Phase == "" && shared.InterviewState.Round == 0 && shared.InterviewState.Difficulty == 0 {
		shared.InterviewState = o.InterviewState
	}
	return shared
}

func (c SharedSessionContext) Summary() string {
	parts := make([]string, 0, 4)
	if c.RunID != "" {
		parts = append(parts, "run="+c.RunID)
	}
	if c.PromptVersion != "" {
		parts = append(parts, "prompt="+c.PromptVersion)
	}
	if c.RunPhase != "" {
		parts = append(parts, "phase="+string(c.RunPhase))
	}
	if c.Skill.Name != "" {
		parts = append(parts, "skill="+c.Skill.Name)
	}
	if c.Rubric.Title != "" {
		parts = append(parts, "rubric="+c.Rubric.Title)
	}
	if c.InterviewState.Round > 0 {
		parts = append(parts, fmt.Sprintf("round=%d", c.InterviewState.Round))
	}
	return strings.Join(parts, " | ")
}
