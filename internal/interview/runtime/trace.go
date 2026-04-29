package runtime

import (
	"context"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type TraceMiddleware struct {
	adk.BaseChatModelAgentMiddleware
}

type TraceEvent struct {
	Type       string `json:"type"`
	Name       string `json:"name,omitempty"`
	CallID     string `json:"callId,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
	Error      string `json:"error,omitempty"`
}

func init() {
	schema.RegisterName[TraceEvent]("_offerbot_trace_event")
}

func NewTraceMiddleware() *TraceMiddleware {
	return &TraceMiddleware{}
}

func (m *TraceMiddleware) WrapModel(_ context.Context, base model.BaseChatModel, _ *adk.ModelContext) (model.BaseChatModel, error) {
	return tracedModel{base: base}, nil
}

func (m *TraceMiddleware) WrapInvokableToolCall(_ context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		start := time.Now()
		result, err := endpoint(ctx, argumentsInJSON, opts...)
		sendTrace(ctx, toolTraceEvent("tool", tCtx, start, err))
		return result, err
	}, nil
}

func (m *TraceMiddleware) WrapEnhancedInvokableToolCall(_ context.Context, endpoint adk.EnhancedInvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.EnhancedInvokableToolCallEndpoint, error) {
	return func(ctx context.Context, argument *schema.ToolArgument, opts ...tool.Option) (*schema.ToolResult, error) {
		start := time.Now()
		result, err := endpoint(ctx, argument, opts...)
		sendTrace(ctx, toolTraceEvent("tool", tCtx, start, err))
		return result, err
	}, nil
}

type tracedModel struct {
	base model.BaseChatModel
}

func (m tracedModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	start := time.Now()
	result, err := m.base.Generate(ctx, input, opts...)
	sendTrace(ctx, TraceEvent{
		Type:       "model_generate",
		DurationMs: time.Since(start).Milliseconds(),
		Error:      errorString(err),
	})
	return result, err
}

func (m tracedModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	start := time.Now()
	result, err := m.base.Stream(ctx, input, opts...)
	sendTrace(ctx, TraceEvent{
		Type:       "model_stream_open",
		DurationMs: time.Since(start).Milliseconds(),
		Error:      errorString(err),
	})
	return result, err
}

func toolTraceEvent(eventType string, tCtx *adk.ToolContext, start time.Time, err error) TraceEvent {
	event := TraceEvent{Type: eventType, DurationMs: time.Since(start).Milliseconds(), Error: errorString(err)}
	if tCtx != nil {
		event.Name = tCtx.Name
		event.CallID = tCtx.CallID
	}
	return event
}

func sendTrace(ctx context.Context, event TraceEvent) {
	_ = adk.SendEvent(ctx, &adk.AgentEvent{Action: &adk.AgentAction{CustomizedAction: event}})
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
