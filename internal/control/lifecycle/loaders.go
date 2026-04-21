package lifecycle

import (
	"context"

	runtimepkg "mockinterview/internal/control/runtime"
	"mockinterview/internal/protocol"
)

type SkillResolutionInput struct {
	Config protocol.InterviewConfig
}

type SkillResolutionOutput struct {
	Skill  protocol.SkillSpec
	Rubric protocol.Rubric
}

func ResolveInterviewAssets(
	ctx context.Context,
	tools runtimepkg.ToolGateway,
	input SkillResolutionInput,
) (SkillResolutionOutput, error) {
	if tools == nil {
		return SkillResolutionOutput{}, nil
	}

	skill, err := tools.ResolveSkill(ctx, input.Config)
	if err != nil {
		return SkillResolutionOutput{}, err
	}
	rubric, err := tools.ResolveRubric(ctx, input.Config)
	if err != nil {
		return SkillResolutionOutput{}, err
	}

	return SkillResolutionOutput{
		Skill:  skill,
		Rubric: rubric,
	}, nil
}

type MemoryLoadInput struct {
	Run protocol.Run
}

type MemoryLoadOutput struct {
	Records []protocol.MemoryRecord
}

func LoadRunMemory(
	ctx context.Context,
	tools runtimepkg.ToolGateway,
	input MemoryLoadInput,
) (MemoryLoadOutput, error) {
	if tools == nil {
		return MemoryLoadOutput{}, nil
	}

	records, err := tools.LoadMemory(ctx, input.Run.ID)
	if err != nil {
		return MemoryLoadOutput{}, err
	}

	return MemoryLoadOutput{
		Records: records,
	}, nil
}
