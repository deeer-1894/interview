package adkapp

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"

	"mockinterview/internal/interview"
	"mockinterview/internal/protocol"
)

type App struct {
	Role   protocol.AgentRole
	Runner *adk.Runner
}

func New(ctx context.Context, interviewCfg interview.InterviewConfig, modelCfg interview.ModelConfig) (*App, error) {
	return NewInterviewerApp(ctx, interviewCfg, modelCfg, Options{})
}

func NewWithOptions(ctx context.Context, interviewCfg interview.InterviewConfig, modelCfg interview.ModelConfig, opts Options) (*App, error) {
	return NewInterviewerApp(ctx, interviewCfg, modelCfg, opts)
}

func NewInterviewerApp(ctx context.Context, interviewCfg interview.InterviewConfig, modelCfg interview.ModelConfig, opts Options) (*App, error) {
	interviewCfg = interviewCfg.WithDefaults()
	modelCfg = modelCfg.WithDefaults()
	shared := opts.ResolveSharedContext()

	llm, err := NewModel(ctx, modelCfg)
	if err != nil {
		return nil, err
	}

	handlers, err := buildAgentHandlers(ctx, llm)
	if err != nil {
		return nil, fmt.Errorf("build adk handlers: %w", err)
	}

	interviewer, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "interviewer",
		Description:   "Conducts mock interviews by loading the most relevant interview skill and adapting to the requested scenario.",
		Instruction:   interview.BuildInterviewerInstruction(interviewCfg, shared.RunPhase, shared.InterviewState, shared.Skill),
		Model:         llm,
		Handlers:      handlers,
		MaxIterations: 12,
	})
	if err != nil {
		return nil, fmt.Errorf("build interviewer agent: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           interviewer,
		EnableStreaming: true,
		CheckPointStore: opts.CheckPointStore,
	})

	return &App{Role: protocol.AgentRoleInterviewer, Runner: runner}, nil
}

func NewEvaluatorApp(ctx context.Context, modelCfg interview.ModelConfig, instruction string, opts Options) (*App, error) {
	modelCfg = modelCfg.WithDefaults()

	llm, err := NewModel(ctx, modelCfg)
	if err != nil {
		return nil, err
	}

	evaluator, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "evaluator",
		Description:   "Evaluates the completed interview transcript and returns structured review artifacts.",
		Instruction:   instruction,
		Model:         llm,
		MaxIterations: 4,
	})
	if err != nil {
		return nil, fmt.Errorf("build evaluator agent: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           evaluator,
		EnableStreaming: true,
		CheckPointStore: opts.CheckPointStore,
	})

	return &App{Role: protocol.AgentRoleEvaluator, Runner: runner}, nil
}
