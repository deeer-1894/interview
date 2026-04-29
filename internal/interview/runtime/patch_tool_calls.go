package runtime

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/patchtoolcalls"
)

func NewPatchToolCallsMiddleware(ctx context.Context) (adk.ChatModelAgentMiddleware, error) {
	return patchtoolcalls.New(ctx, &patchtoolcalls.Config{
		PatchedContentGenerator: func(_ context.Context, toolName, toolCallID string) (string, error) {
			return fmt.Sprintf("[tool_result_patched] name=%s call_id=%s result_empty=true", toolName, toolCallID), nil
		},
	})
}
