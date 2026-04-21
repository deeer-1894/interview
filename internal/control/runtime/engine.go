package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/google/uuid"

	"mockinterview/internal/executors/interview"
	"mockinterview/internal/protocol"
)

type Next func(ctx *RunContext) error

type Middleware interface {
	Name() string
	Handle(ctx *RunContext, next Next) error
}

type DependencyAwareMiddleware interface {
	Middleware
	Spec() MiddlewareSpec
}

type MiddlewareSpec struct {
	Name     string
	Requires []string
	Modes    []protocol.InterviewMode
}

type Executor interface {
	Execute(
		ctx context.Context,
		prompt string,
		cfg protocol.InterviewConfig,
		modelCfg protocol.ModelConfig,
		run *protocol.Run,
		skill protocol.SkillSpec,
		messages []protocol.Message,
		resume bool,
		checkpoint protocol.CheckpointSnapshot,
		checkPointStore adk.CheckPointStore,
		onAssistantDelta func(delta string, content string),
	) (string, error)
}

type EventRecorder interface {
	RecordEvent(ctx context.Context, event protocol.Event) error
	RecordMessage(ctx context.Context, message protocol.Message) error
	UpdateRun(ctx context.Context, run protocol.Run) error
	GetCandidateProfile(ctx context.Context) (protocol.CandidateProfile, error)
	SaveCandidateProfile(ctx context.Context, profile protocol.CandidateProfile) (protocol.CandidateProfile, error)
}

type Engine interface {
	Run(ctx context.Context, runCtx *RunContext) error
	Resume(ctx context.Context, runCtx *RunContext, input protocol.ResumeInput) error
}

type StandardEngine struct {
	middlewares []Middleware
	executor    Executor
}

func NewStandardEngine(middlewares []Middleware) *StandardEngine {
	return &StandardEngine{
		middlewares: middlewares,
		executor:    interview.NewExecutor(),
	}
}

func (e *StandardEngine) Run(ctx context.Context, runCtx *RunContext) error {
	runCtx.Context = ctx
	runCtx.Executor = e.executor
	chain, specs, err := e.resolveChain(runCtx.Task.Config.WithDefaults().Mode)
	if err != nil {
		return protocol.WrapInterviewError("engine", "middleware.resolve", false, err)
	}
	runCtx.ConfigureMiddlewareChain(specs)
	return e.chain(chain, 0)(runCtx)
}

func (e *StandardEngine) Resume(ctx context.Context, runCtx *RunContext, input protocol.ResumeInput) error {
	runCtx.Request.Resume = true
	runCtx.Request.ClarifyResponse = input.Message
	return e.Run(ctx, runCtx)
}

func (e *StandardEngine) chain(middlewares []Middleware, index int) Next {
	if index >= len(middlewares) {
		return func(ctx *RunContext) error {
			return ctx.execute()
		}
	}

	return func(ctx *RunContext) error {
		mw := middlewares[index]
		spec := middlewareSpecOf(mw)
		start := time.Now()
		ctx.emitSpanStart(Span{Scope: "middleware", Name: mw.Name()})
		err := mw.Handle(ctx, e.chain(middlewares, index+1))
		duration := time.Since(start)
		if err == nil {
			ctx.MarkMiddlewareCompleted(spec.Name)
		}
		ctx.emitSpanEnd(Span{Scope: "middleware", Name: mw.Name()}, err, duration)
		if ctx.Recorder != nil {
			_ = ctx.Recorder.RecordEvent(ctx.Context, protocol.Event{
				ID:             uuid.NewString(),
				ConversationID: ctx.Run.ConversationID,
				TaskID:         ctx.Run.TaskID,
				RunID:          ctx.Run.ID,
				Type:           protocol.EventMiddlewareSummary,
				Timestamp:      time.Now(),
				Payload:        buildMiddlewareSummary(mw.Name(), ctx, duration, err),
			})
		}
		return err
	}
}

func (e *StandardEngine) resolveChain(mode protocol.InterviewMode) ([]Middleware, []MiddlewareSpec, error) {
	selected := make([]Middleware, 0, len(e.middlewares))
	specs := make([]MiddlewareSpec, 0, len(e.middlewares))
	for _, middleware := range e.middlewares {
		spec := middlewareSpecOf(middleware)
		if !middlewareEnabledForMode(spec, mode) {
			continue
		}
		selected = append(selected, middleware)
		specs = append(specs, spec)
	}
	if err := validateMiddlewareChain(specs); err != nil {
		return nil, nil, err
	}
	return selected, specs, nil
}

func validateMiddlewareChain(specs []MiddlewareSpec) error {
	if len(specs) == 0 {
		return errors.New("middleware chain is empty")
	}
	seen := make(map[string]struct{}, len(specs))
	for _, spec := range specs {
		if spec.Name == "" {
			return errors.New("middleware name is required")
		}
		if _, exists := seen[spec.Name]; exists {
			return fmt.Errorf("duplicate middleware %q", spec.Name)
		}
		for _, requirement := range spec.Requires {
			if _, exists := seen[requirement]; !exists {
				return fmt.Errorf("middleware %q requires %q to appear earlier in the chain", spec.Name, requirement)
			}
		}
		seen[spec.Name] = struct{}{}
	}
	return nil
}

func middlewareSpecOf(middleware Middleware) MiddlewareSpec {
	if aware, ok := middleware.(DependencyAwareMiddleware); ok {
		return normalizeMiddlewareSpec(aware.Spec())
	}
	return normalizeMiddlewareSpec(MiddlewareSpec{Name: middleware.Name()})
}

func normalizeMiddlewareSpec(spec MiddlewareSpec) MiddlewareSpec {
	spec.Name = normalizeMiddlewareName(spec.Name)
	spec.Requires = normalizeMiddlewareNames(spec.Requires)
	spec.Modes = normalizeMiddlewareModes(spec.Modes)
	return spec
}

func normalizeMiddlewareName(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func normalizeMiddlewareNames(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := normalizeMiddlewareName(value)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func normalizeMiddlewareModes(values []protocol.InterviewMode) []protocol.InterviewMode {
	if len(values) == 0 {
		return nil
	}
	out := make([]protocol.InterviewMode, 0, len(values))
	seen := make(map[protocol.InterviewMode]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func middlewareEnabledForMode(spec MiddlewareSpec, mode protocol.InterviewMode) bool {
	if len(spec.Modes) == 0 {
		return true
	}
	if mode == "" {
		mode = protocol.ModeStandard
	}
	for _, candidate := range spec.Modes {
		if candidate == mode {
			return true
		}
	}
	return false
}

func (c *RunContext) execute() error {
	if c.Executor == nil {
		return protocol.WrapInterviewError("executor", "execute", false, fmt.Errorf("executor is not configured"))
	}
	start := time.Now()
	c.emitSpanStart(Span{Scope: "executor", Name: "interview"})
	output, err := c.Executor.Execute(
		c.Context,
		c.Prompt,
		c.Task.Config,
		c.Task.ModelConfig,
		&c.Run,
		c.Resolved.Interview.Skill,
		c.Messages,
		c.Request.Resume,
		c.Resolved.Interview.Checkpoint,
		c.ADKCheckpoints,
		func(delta string, content string) {
			if c.Recorder == nil {
				return
			}
			if content == "" {
				return
			}
			_ = c.Recorder.RecordEvent(c.Context, protocol.Event{
				ID:             fmt.Sprintf("delta_%d", time.Now().UnixNano()),
				ConversationID: c.Run.ConversationID,
				TaskID:         c.Run.TaskID,
				RunID:          c.Run.ID,
				Type:           protocol.EventMessageDelta,
				Timestamp:      time.Now(),
				Payload: map[string]string{
					"delta":   delta,
					"content": content,
				},
			})
		},
	)
	c.emitSpanEnd(Span{Scope: "executor", Name: "interview"}, err, time.Since(start))
	if err != nil {
		return protocol.WrapInterviewError("executor", "interview.execute", false, err)
	}
	c.Result.Output = output
	return nil
}

func (c *RunContext) emitSpanStart(span Span) {
	for _, callback := range c.Callbacks {
		callback.OnSpanStart(c.Context, span)
	}
}

func (c *RunContext) emitSpanEnd(span Span, err error, duration time.Duration) {
	for _, callback := range c.Callbacks {
		callback.OnSpanEnd(c.Context, span, err, duration)
	}
}
