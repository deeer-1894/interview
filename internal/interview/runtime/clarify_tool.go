package runtime

import (
	"context"
	"encoding/json"
	"fmt"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const ToolClarify = "clarify"

type ClarifyInfo struct {
	Question string `json:"question"`
	Reason   string `json:"reason,omitempty"`
}

type ClarifyState struct {
	Question string `json:"question"`
	Reason   string `json:"reason,omitempty"`
}

type clarifyArgs struct {
	Question string `json:"question"`
	Reason   string `json:"reason,omitempty"`
}

func init() {
	schema.Register[ClarifyInfo]()
	schema.Register[ClarifyState]()
}

func NewClarifyTool() einotool.BaseTool {
	return clarifyTool{}
}

func ControlToolNames() []string {
	return []string{ToolClarify}
}

type clarifyTool struct{}

func (t clarifyTool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: ToolClarify,
		Desc: "当缺少必要事实或候选人回答无法继续判断时，中断当前轮次并向候选人澄清一个问题。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"question": {Type: schema.String, Desc: "需要候选人澄清的问题", Required: true},
			"reason":   {Type: schema.String, Desc: "为什么需要澄清"},
		}),
	}, nil
}

func (t clarifyTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...einotool.Option) (string, error) {
	wasInterrupted, hasState, state := einotool.GetInterruptState[ClarifyState](ctx)
	if !wasInterrupted {
		args, err := parseClarifyArgs(argumentsInJSON)
		if err != nil {
			return "", err
		}
		info := ClarifyInfo{Question: args.Question, Reason: args.Reason}
		return "", einotool.StatefulInterrupt(ctx, info, ClarifyState(info))
	}

	isResumeTarget, hasData, data := einotool.GetResumeContext[string](ctx)
	if !isResumeTarget {
		if !hasState {
			return "", einotool.StatefulInterrupt(ctx, ClarifyInfo{}, ClarifyState{})
		}
		return "", einotool.StatefulInterrupt(ctx, ClarifyInfo{Question: state.Question, Reason: state.Reason}, state)
	}
	if hasData && data != "" {
		return data, nil
	}
	return "候选人已确认继续，但没有提供额外澄清。", nil
}

func parseClarifyArgs(argumentsInJSON string) (clarifyArgs, error) {
	var args clarifyArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return clarifyArgs{}, err
	}
	if args.Question == "" {
		return clarifyArgs{}, fmt.Errorf("clarify question is required")
	}
	return args, nil
}
