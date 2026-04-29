package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const (
	ToolInterviewContext = "interview_context"
	ToolActiveProject    = "active_project"
	ToolRecentHistory    = "recent_history"
	ToolScorecardOutline = "scorecard_outline"
)

func NewGraphTools() []tool.BaseTool {
	return []tool.BaseTool{
		readOnlyTool{
			name: ToolInterviewContext,
			desc: "读取当前面试轮次的面试官可读上下文摘要。",
			run: func(ctx context.Context) (any, error) {
				turn, ok := TurnContextFrom(ctx)
				if !ok {
					return nil, fmt.Errorf("turn context not available")
				}
				return renderInterviewContext(turn), nil
			},
		},
		readOnlyTool{
			name: ToolActiveProject,
			desc: "读取当前项目的候选人可见事实摘要。",
			run: func(ctx context.Context) (any, error) {
				turn, ok := TurnContextFrom(ctx)
				if !ok {
					return nil, fmt.Errorf("turn context not available")
				}
				return map[string]any{
					"name":      turn.ActiveProject.Name,
					"domain":    turn.ActiveProject.Domain,
					"summary":   turn.ActiveProject.Summary,
					"techStack": turn.ActiveProject.TechStack,
					"claims":    turn.ActiveProject.Claims,
				}, nil
			},
		},
		readOnlyTool{
			name: ToolRecentHistory,
			desc: "读取当前会话最近的用户和面试官对话，不包含内部 ID。",
			run: func(ctx context.Context) (any, error) {
				turn, ok := TurnContextFrom(ctx)
				if !ok {
					return nil, fmt.Errorf("turn context not available")
				}
				history := make([]map[string]string, 0, len(turn.RecentMessages))
				for _, msg := range turn.RecentMessages {
					if msg.Role != "user" && msg.Role != "assistant" {
						continue
					}
					history = append(history, map[string]string{
						"role":    string(msg.Role),
						"content": compactText(msg.Content, 600),
					})
				}
				return history, nil
			},
		},
		readOnlyTool{
			name: ToolScorecardOutline,
			desc: "读取本场面试报告应覆盖的通用评分维度。",
			run: func(ctx context.Context) (any, error) {
				return []string{"系统设计", "工程实现", "并发与稳定性", "数据与状态", "问题排查", "沟通与取舍"}, nil
			},
		},
	}
}

func GraphToolNames() []string {
	return []string{ToolInterviewContext, ToolActiveProject, ToolRecentHistory, ToolScorecardOutline}
}

type readOnlyTool struct {
	name string
	desc string
	run  func(context.Context) (any, error)
}

func (t readOnlyTool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        t.name,
		Desc:        t.desc,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t readOnlyTool) InvokableRun(ctx context.Context, _ string, _ ...tool.Option) (string, error) {
	value, err := t.run(ctx)
	if err != nil {
		return "", err
	}
	if text, ok := value.(string); ok {
		return text, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(string(data), "\\u003e", ">"), nil
}
