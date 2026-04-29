package runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"

	"mockinterview/internal/interview/store"
)

const (
	defaultMaxIterations = 4
	defaultCheckpointTTL = 24 * time.Hour
)

type Config struct {
	Name            string
	Description     string
	Instruction     string
	Model           model.BaseChatModel
	Tools           []tool.BaseTool
	Handlers        []adk.ChatModelAgentMiddleware
	Stores          store.Bundle
	EnableStreaming bool
	MaxIterations   int
}

type Runtime struct {
	runner *adk.Runner
	stores store.Bundle
}

type TurnRequest struct {
	UserID        string
	SessionID     string
	Input         string
	CheckpointID  string
	SessionValues map[string]any
}

type ResumeRequest struct {
	UserID        string
	SessionID     string
	CheckpointID  string
	ResumeData    map[string]any
	SessionValues map[string]any
}

func New(ctx context.Context, cfg Config) (*Runtime, error) {
	if cfg.Model == nil {
		return nil, errors.New("interview runtime requires a chat model")
	}
	if cfg.Stores.Checkpoints == nil {
		return nil, errors.New("interview runtime requires a checkpoint store")
	}
	if cfg.MaxIterations <= 0 {
		cfg.MaxIterations = defaultMaxIterations
	}
	if cfg.Name == "" {
		cfg.Name = "offerbot_interviewer"
	}
	if cfg.Description == "" {
		cfg.Description = "Runs one structured interview turn."
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          cfg.Name,
		Description:   cfg.Description,
		Instruction:   cfg.Instruction,
		Model:         cfg.Model,
		ToolsConfig:   adk.ToolsConfig{ToolsNodeConfig: compose.ToolsNodeConfig{Tools: cfg.Tools}, ReturnDirectly: map[string]bool{}},
		GenModelInput: BuildModelInput,
		MaxIterations: cfg.MaxIterations,
		Handlers:      cfg.Handlers,
	})
	if err != nil {
		return nil, err
	}

	agentConfig := adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: cfg.EnableStreaming,
		CheckPointStore: cfg.Stores.Checkpoints,
	}
	return &Runtime{
		runner: adk.NewRunner(ctx, agentConfig),
		stores: cfg.Stores,
	}, nil
}

func (r *Runtime) RunTurn(ctx context.Context, req TurnRequest) (*adk.AsyncIterator[*adk.AgentEvent], string, error) {
	if req.UserID == "" || req.SessionID == "" {
		return nil, "", errors.New("run turn requires userID and sessionID")
	}
	if req.Input == "" {
		return nil, "", errors.New("run turn requires input")
	}
	checkpointID := req.CheckpointID
	if checkpointID == "" {
		checkpointID = fmt.Sprintf("%s:%s", req.SessionID, uuid.NewString())
	}
	if r.stores.Runs != nil {
		if err := r.stores.Runs.SetCheckpointID(ctx, req.SessionID, checkpointID, defaultCheckpointTTL); err != nil {
			return nil, "", err
		}
	}

	values := baseSessionValues(req.UserID, req.SessionID, req.SessionValues)
	values["last_answer"] = req.Input
	iter := r.runner.Run(
		ctx,
		[]adk.Message{schema.UserMessage(req.Input)},
		adk.WithSessionValues(values),
		adk.WithCheckPointID(checkpointID),
	)
	return iter, checkpointID, nil
}

func (r *Runtime) ResumeTurn(ctx context.Context, req ResumeRequest) (*adk.AsyncIterator[*adk.AgentEvent], string, error) {
	if req.UserID == "" || req.SessionID == "" {
		return nil, "", errors.New("resume turn requires userID and sessionID")
	}

	checkpointID := req.CheckpointID
	if checkpointID == "" {
		if r.stores.Runs == nil {
			return nil, "", errors.New("resume turn requires checkpointID when run store is unavailable")
		}
		value, ok, err := r.stores.Runs.GetCheckpointID(ctx, req.SessionID)
		if err != nil {
			return nil, "", err
		}
		if !ok {
			return nil, "", store.ErrNotFound
		}
		checkpointID = value
	}

	values := baseSessionValues(req.UserID, req.SessionID, req.SessionValues)
	var (
		iter *adk.AsyncIterator[*adk.AgentEvent]
		err  error
	)
	if len(req.ResumeData) > 0 {
		iter, err = r.runner.ResumeWithParams(
			ctx,
			checkpointID,
			&adk.ResumeParams{Targets: req.ResumeData},
			adk.WithSessionValues(values),
		)
	} else {
		iter, err = r.runner.Resume(ctx, checkpointID, adk.WithSessionValues(values))
	}
	if err != nil {
		return nil, "", err
	}
	return iter, checkpointID, nil
}

func baseSessionValues(userID string, sessionID string, extra map[string]any) map[string]any {
	values := make(map[string]any, len(extra)+2)
	values["user_id"] = userID
	values["session_id"] = sessionID
	for key, value := range extra {
		values[key] = value
	}
	return values
}
