package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	domain "mockinterview/internal/interview"
	"mockinterview/internal/protocol"
)

func (a *App) RequestCopilotHint(ctx context.Context, runID string) (protocol.CopilotAssistResponse, error) {
	run, err := a.runs.Get(ctx, runID)
	if err != nil {
		return protocol.CopilotAssistResponse{}, fmt.Errorf("load run %s: %w", runID, err)
	}
	if run.Status == protocol.RunCancelled || run.Status == protocol.RunFailed {
		return protocol.CopilotAssistResponse{}, fmt.Errorf("run %s is not available for copilot help", runID)
	}

	task, err := a.tasks.Get(ctx, run.TaskID)
	if err != nil {
		return protocol.CopilotAssistResponse{}, fmt.Errorf("load task %s: %w", run.TaskID, err)
	}
	messages, err := a.messages.ListByRun(ctx, runID)
	if err != nil {
		return protocol.CopilotAssistResponse{}, fmt.Errorf("load messages for run %s: %w", runID, err)
	}
	if len(messages) == 0 {
		return protocol.CopilotAssistResponse{}, fmt.Errorf("run %s has no transcript for copilot help", runID)
	}

	feedback := buildCopilotFeedback(run, task.Config, messages)
	hint := buildCopilotHint(run, task.Config, messages, feedback)
	now := time.Now()
	recorded := make([]protocol.Event, 0, 2)
	for _, event := range []protocol.Event{
		{
			ID:             uuid.NewString(),
			ConversationID: run.ConversationID,
			TaskID:         run.TaskID,
			RunID:          run.ID,
			Type:           protocol.EventCopilotFeedback,
			Timestamp:      now,
			Payload:        feedback,
		},
		{
			ID:             uuid.NewString(),
			ConversationID: run.ConversationID,
			TaskID:         run.TaskID,
			RunID:          run.ID,
			Type:           protocol.EventCopilotHint,
			Timestamp:      now,
			Payload:        hint,
		},
	} {
		if err := a.RecordEvent(ctx, event); err != nil {
			return protocol.CopilotAssistResponse{}, fmt.Errorf("record copilot event %s: %w", event.Type, err)
		}
		recorded = append(recorded, event)
	}

	return protocol.CopilotAssistResponse{
		Feedback: feedback,
		Hint:     hint,
		Events:   recorded,
	}, nil
}

func buildCopilotFeedback(run protocol.Run, cfg protocol.InterviewConfig, messages []protocol.Message) protocol.CopilotFeedback {
	latestUser := latestMessageByRole(messages, "user")
	state := domain.EnsureRunInterviewState(run.InterviewState)
	triggers := collectCopilotTriggers(latestUser.Content, state)

	feedback := protocol.CopilotFeedback{
		State:      protocol.CopilotStateStable,
		Triggers:   append([]string(nil), triggers...),
		Confidence: 0.62,
	}

	switch {
	case containsAnyFold(latestUser.Content, anxietyKeywords):
		feedback.State = protocol.CopilotStateAnxious
		feedback.Summary = "你现在更需要先稳住表达节奏，而不是一次把答案讲满。"
		feedback.SuggestedMoves = []string{
			"先给一句结论，帮自己站稳主线。",
			"接着补 2 个最关键的设计点，不急着铺满细节。",
			"最后用一个风险或 tradeoff 收尾，显得更从容。",
		}
		feedback.Confidence = 0.9
	case containsAnyFold(latestUser.Content, stuckKeywords):
		feedback.State = protocol.CopilotStateStuck
		feedback.Summary = "你像是卡在“从哪里开始答”，先用结构把答案撑起来。"
		feedback.SuggestedMoves = []string{
			"先说目标和约束。",
			"再说核心方案和关键组件。",
			"最后补失败处理、监控或 tradeoff。",
		}
		feedback.Confidence = 0.86
	case needsSpecificity(state):
		feedback.State = protocol.CopilotStateNeedsSpecificity
		feedback.Summary = "面试官当前更在意具体实现、例子或权衡，而不是抽象概念。"
		feedback.SuggestedMoves = []string{
			"把答案收敛到一个你真的会落地的方案。",
			"补一个真实组件、流程或接口边界。",
			"明确讲出为什么不用另一种方案。",
		}
		feedback.Confidence = 0.8
	default:
		feedback.State = protocol.CopilotStateNeedsStructure
		feedback.Summary = "你可以继续回答，但最好先给骨架，再展开重点。"
		feedback.SuggestedMoves = []string{
			"先给结论或立场。",
			"中间按 2 到 3 个点展开。",
			"最后补一个风险、取舍或监控点。",
		}
		feedback.Confidence = 0.68
	}

	if strings.TrimSpace(feedback.Summary) == "" {
		feedback.Summary = "当前可以继续作答，但建议先结构化表达。"
	}
	return feedback
}

func buildCopilotHint(
	run protocol.Run,
	cfg protocol.InterviewConfig,
	messages []protocol.Message,
	feedback protocol.CopilotFeedback,
) protocol.CopilotHint {
	latestAssistant := latestMessageByRole(messages, "assistant")
	state := domain.EnsureRunInterviewState(run.InterviewState)
	decisionReason := strings.TrimSpace(string(firstDecisionReason(state)))

	hint := protocol.CopilotHint{
		Title:   "Copilot 提示",
		Summary: "只给你答题方向和组织方式，不直接替你作答。",
		Focus:   deriveCopilotFocus(cfg, state, decisionReason),
		Guardrails: []string{
			"不要直接背标准答案，优先讲你的判断过程。",
			"先给结论，再补实现细节和 tradeoff。",
			"如果信息不全，先声明假设边界。",
		},
	}

	switch {
	case feedback.State == protocol.CopilotStateAnxious:
		hint.Title = "先稳住节奏"
		hint.Strategy = []string{
			"第一句只回答你会怎么做，不展开背景铺垫。",
			"第二句补最关键的实现抓手，比如组件、状态或边界。",
			"第三句给一个风险和兜底方案，让表达显得更稳。",
		}
	case feedback.State == protocol.CopilotStateStuck:
		hint.Title = "给自己一个答题骨架"
		hint.Strategy = []string{
			"先说目标：你要解决什么问题、约束是什么。",
			"再说方案：核心流程怎么跑，哪些组件负责什么。",
			"最后说验证：如何观测、回滚或处理失败。",
		}
	case decisionReason == string(protocol.ReasonMissingTradeoff):
		hint.Title = "补上 tradeoff"
		hint.Focus = "tradeoff"
		hint.Strategy = []string{
			"先说你会选哪种方案。",
			"再说为什么不选另一个方案，点出成本或风险。",
			"最后说在什么场景下你会重新切换选择。",
		}
	case decisionReason == string(protocol.ReasonLackImplementationDetail):
		hint.Title = "把方案落到实现层"
		hint.Focus = "implementation_detail"
		hint.Strategy = []string{
			"给出具体组件或模块拆分。",
			"说明关键数据流、状态同步或失败处理。",
			"补一个你会如何验证正确性的例子。",
		}
	case decisionReason == string(protocol.ReasonWeakSignalTimeout):
		hint.Title = "把超时与取消讲具体"
		hint.Focus = "timeout_control"
		hint.Strategy = []string{
			"说明 timeout 设置在哪一层。",
			"讲清 context cancel 如何向下传播。",
			"补充重试边界和避免雪崩的保护措施。",
		}
	case decisionReason == string(protocol.ReasonWeakSignalObservability):
		hint.Title = "补上可观测性"
		hint.Focus = "observability"
		hint.Strategy = []string{
			"至少给出 metrics、logs、traces 三类中的两类。",
			"说清你会看哪些指标判断系统健康。",
			"补一个告警或排障入口，别只说“加监控”。",
		}
	default:
		hint.Strategy = []string{
			"围绕面试官最近这句追问，先给一条明确判断。",
			"接着补你真正会落地的实现细节，而不是抽象原则。",
			"最后用一个例子、风险或 tradeoff 收尾。",
		}
	}

	if strings.TrimSpace(latestAssistant.Content) != "" {
		hint.Summary = fmt.Sprintf("围绕面试官刚才的问题回答即可：%s", summarizeCopilotQuestion(latestAssistant.Content))
	}
	return hint
}

func deriveCopilotFocus(cfg protocol.InterviewConfig, state protocol.RunInterviewState, decisionReason string) string {
	switch decisionReason {
	case string(protocol.ReasonMissingTradeoff):
		return "tradeoff"
	case string(protocol.ReasonLackImplementationDetail):
		return "implementation_detail"
	case string(protocol.ReasonWeakSignalTimeout):
		return "timeout_control"
	case string(protocol.ReasonWeakSignalObservability):
		return "observability"
	}
	if state.LastDecision != nil && len(state.LastDecision.RecommendedFocus) > 0 {
		return strings.TrimSpace(state.LastDecision.RecommendedFocus[0])
	}
	if strings.TrimSpace(cfg.Focus) != "" {
		return strings.TrimSpace(cfg.Focus)
	}
	if strings.TrimSpace(cfg.Skill) != "" {
		return strings.TrimSpace(cfg.Skill)
	}
	return "current_question"
}

func collectCopilotTriggers(latestUser string, state protocol.RunInterviewState) []string {
	triggers := make([]string, 0, 4)
	if containsAnyFold(latestUser, anxietyKeywords) {
		triggers = append(triggers, "anxiety_keywords")
	}
	if containsAnyFold(latestUser, stuckKeywords) {
		triggers = append(triggers, "stuck_keywords")
	}
	if needsSpecificity(state) {
		triggers = append(triggers, "needs_specificity")
	}
	if state.LastDecision != nil && strings.TrimSpace(string(state.LastDecision.Reason)) != "" {
		triggers = append(triggers, string(state.LastDecision.Reason))
	}
	return uniqueStrings(triggers)
}

func needsSpecificity(state protocol.RunInterviewState) bool {
	weakSignals := append([]string(nil), state.WeakSignals...)
	if count := len(state.History); count > 0 {
		weakSignals = append(weakSignals, state.History[count-1].WeakSignals...)
	}
	for _, signal := range weakSignals {
		switch strings.TrimSpace(signal) {
		case "too_generic", "partial_answer", "concept_without_plan", "lacks_example_or_evidence", "missing_tradeoff", "missing_implementation_detail", "missing_timeout_detail", "missing_observability_detail":
			return true
		}
	}
	if state.LastDecision == nil {
		return false
	}
	switch state.LastDecision.Reason {
	case protocol.ReasonMissingTradeoff, protocol.ReasonLackImplementationDetail, protocol.ReasonWeakSignalTimeout, protocol.ReasonWeakSignalObservability:
		return true
	default:
		return false
	}
}

func firstDecisionReason(state protocol.RunInterviewState) protocol.DecisionReason {
	if state.LastDecision == nil {
		return ""
	}
	return state.LastDecision.Reason
}

func latestMessageByRole(messages []protocol.Message, role string) protocol.Message {
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.EqualFold(messages[i].Role, role) && strings.TrimSpace(messages[i].Content) != "" {
			return messages[i]
		}
	}
	return protocol.Message{}
}

func summarizeCopilotQuestion(content string) string {
	content = strings.Join(strings.Fields(strings.TrimSpace(content)), " ")
	runes := []rune(content)
	if len(runes) <= 72 {
		return content
	}
	return string(runes[:72]) + "..."
}

func containsAnyFold(value string, keywords []string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return false
	}
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		key := strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

var anxietyKeywords = []string{
	"紧张",
	"慌",
	"慌了",
	"焦虑",
	"有点怕",
	"没底",
	"不知道对不对",
	"心里没底",
	"panic",
	"nervous",
}

var stuckKeywords = []string{
	"卡壳",
	"卡住",
	"不会",
	"没思路",
	"不知道怎么答",
	"不知道怎么展开",
	"想不起来",
	"帮我提示",
	"给点提示",
	"stuck",
}
