package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const defaultToolTimeout = 8 * time.Second

type SafeToolMiddleware struct {
	adk.BaseChatModelAgentMiddleware

	Allowed map[string]struct{}
	Timeout time.Duration
}

func NewSafeToolMiddleware(allowed []string, timeout time.Duration) *SafeToolMiddleware {
	if timeout <= 0 {
		timeout = defaultToolTimeout
	}
	allowedSet := make(map[string]struct{}, len(allowed)+1)
	allowedSet[ToolClarify] = struct{}{}
	for _, name := range allowed {
		if name != "" {
			allowedSet[name] = struct{}{}
		}
	}
	return &SafeToolMiddleware{Allowed: allowedSet, Timeout: timeout}
}

func (m *SafeToolMiddleware) WrapInvokableToolCall(_ context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	if err := m.checkAllowed(tCtx); err != nil {
		return nil, err
	}
	return func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		callCtx, cancel := context.WithTimeout(ctx, m.Timeout)
		defer cancel()
		result, err := endpoint(callCtx, argumentsInJSON, opts...)
		if err != nil {
			return "", m.wrapToolError(tCtx, err)
		}
		return result, nil
	}, nil
}

func (m *SafeToolMiddleware) WrapStreamableToolCall(_ context.Context, endpoint adk.StreamableToolCallEndpoint, tCtx *adk.ToolContext) (adk.StreamableToolCallEndpoint, error) {
	if err := m.checkAllowed(tCtx); err != nil {
		return nil, err
	}
	return func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		result, err := endpoint(ctx, argumentsInJSON, opts...)
		if err != nil {
			return nil, m.wrapToolError(tCtx, err)
		}
		return result, nil
	}, nil
}

func (m *SafeToolMiddleware) WrapEnhancedInvokableToolCall(_ context.Context, endpoint adk.EnhancedInvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.EnhancedInvokableToolCallEndpoint, error) {
	if err := m.checkAllowed(tCtx); err != nil {
		return nil, err
	}
	return func(ctx context.Context, argument *schema.ToolArgument, opts ...tool.Option) (*schema.ToolResult, error) {
		callCtx, cancel := context.WithTimeout(ctx, m.Timeout)
		defer cancel()
		result, err := endpoint(callCtx, argument, opts...)
		if err != nil {
			return nil, m.wrapToolError(tCtx, err)
		}
		return result, nil
	}, nil
}

func (m *SafeToolMiddleware) WrapEnhancedStreamableToolCall(_ context.Context, endpoint adk.EnhancedStreamableToolCallEndpoint, tCtx *adk.ToolContext) (adk.EnhancedStreamableToolCallEndpoint, error) {
	if err := m.checkAllowed(tCtx); err != nil {
		return nil, err
	}
	return func(ctx context.Context, argument *schema.ToolArgument, opts ...tool.Option) (*schema.StreamReader[*schema.ToolResult], error) {
		result, err := endpoint(ctx, argument, opts...)
		if err != nil {
			return nil, m.wrapToolError(tCtx, err)
		}
		return result, nil
	}, nil
}

func (m *SafeToolMiddleware) checkAllowed(tCtx *adk.ToolContext) error {
	if tCtx == nil {
		return fmt.Errorf("tool call rejected: missing tool context")
	}
	if len(m.Allowed) == 0 {
		return nil
	}
	if _, ok := m.Allowed[tCtx.Name]; !ok {
		return fmt.Errorf("tool call rejected: %s is not allowed", tCtx.Name)
	}
	return nil
}

func (m *SafeToolMiddleware) wrapToolError(tCtx *adk.ToolContext, err error) error {
	if tCtx == nil {
		return fmt.Errorf("tool call failed: %w", err)
	}
	return fmt.Errorf("tool call %s failed: %w", tCtx.Name, err)
}
