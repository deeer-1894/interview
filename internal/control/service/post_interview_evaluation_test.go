package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	runtimepkg "mockinterview/internal/control/runtime"
	"mockinterview/internal/protocol"
)

func TestCompletePostInterviewEvaluationArchivesDualAgents(t *testing.T) {
	t.Parallel()

	now := time.Now()
	task := protocol.Task{
		ID:             "task_1",
		ConversationID: "conv_1",
		Prompt:         "请继续这场 Go agent 面试。",
		Config: protocol.InterviewConfig{
			Skill:       "go-agent",
			Persona:     protocol.PersonaRigorous,
			Mode:        protocol.ModeStandard,
			OutputStyle: protocol.OutputInterviewPlusScore,
			TimeBudget:  "25 minutes",
		}.WithDefaults(),
	}
	run := protocol.Run{
		ID:             "run_1",
		ConversationID: "conv_1",
		TaskID:         task.ID,
		Status:         protocol.RunRunning,
		Phase:          protocol.RunPhaseEvaluating,
		Input:          "请结束并评分",
		Output:         "今天的面试先到这里，下面我来总结你的表现。",
		CreatedAt:      now.Add(-3 * time.Minute),
		UpdatedAt:      now.Add(-30 * time.Second),
		InterviewState: &protocol.RunInterviewState{
			Phase: protocol.PhaseWrapup,
			LastDecision: &protocol.NextStepDecision{
				Reason:           protocol.ReasonWrapupDueToBudget,
				Explanation:      "时间预算已到，进入总结。",
				RecommendedFocus: []string{"observability"},
			},
		},
		TraceTree: &protocol.InterviewTraceTree{
			QuestionCount: 3,
			Nodes: []protocol.InterviewTraceNode{
				{Round: 1, Phase: protocol.PhaseWarmup},
				{Round: 2, Phase: protocol.PhaseProbe, WeakSignals: []string{"missing_tradeoff"}},
				{Round: 3, Phase: protocol.PhaseWrapup},
			},
		},
	}
	messages := []protocol.Message{
		{
			ID:             "msg_assistant_1",
			ConversationID: run.ConversationID,
			TaskID:         run.TaskID,
			RunID:          run.ID,
			Role:           "assistant",
			Content:        "请解释你会如何用 errgroup 控制并发，并说明失败隔离和 context 取消策略。",
			CreatedAt:      now.Add(-2 * time.Minute),
		},
		{
			ID:             "msg_user_1",
			ConversationID: run.ConversationID,
			TaskID:         run.TaskID,
			RunID:          run.ID,
			Role:           "user",
			Content:        "我会用 errgroup 控制并发，并用 context 透传取消。",
			CreatedAt:      now.Add(-15 * time.Second),
		},
	}
	profile := protocol.CandidateProfile{
		ID:            "global",
		RecurringGaps: []string{"tradeoff_expression"},
	}

	app := &App{
		tasks: &memoryTaskRepo{
			items: map[string]protocol.Task{task.ID: task},
		},
		runs: &memoryRunRepo{
			items: map[string]protocol.Run{run.ID: run},
		},
		messages: &memoryMessageRepo{
			byRun: map[string][]protocol.Message{run.ID: messages},
		},
		events: &memoryEventRepo{
			byRun: make(map[string][]protocol.Event),
		},
		profiles: &memoryProfileRepo{
			items: map[string]protocol.CandidateProfile{profile.ID: profile},
		},
		broker: NewEventBroker(),
		scorecards: func(
			ctx context.Context,
			transcript string,
			cfg protocol.InterviewConfig,
			modelCfg protocol.ModelConfig,
			skill protocol.SkillSpec,
			rubric protocol.Rubric,
		) (protocol.Scorecard, error) {
			return protocol.Scorecard{
				Title:           "Go Agent Rubric",
				Summary:         "候选人对并发控制掌握扎实，但 tradeoff 表达还需要更具体。",
				OverallScore:    82,
				OverallMaxScore: 100,
				Anchors:         append([]string(nil), rubric.Anchors...),
				DimensionScores: []protocol.DimensionScore{
					{Name: "并发控制", Score: 8, MaxScore: 10, Rationale: "能够解释 errgroup 与 context 的组合方式"},
					{Name: "可观测性", Score: 7, MaxScore: 10, Rationale: "观测面覆盖到 metrics 与 trace"},
				},
				Strengths:    []string{"能够解释 errgroup 与 context 的组合方式"},
				Gaps:         []string{"tradeoff 表达不够具体"},
				Improvements: []string{"补充并发失败隔离和观测指标的权衡"},
			}, nil
		},
	}

	runCtx := &runtimepkg.RunContext{
		Context:       context.Background(),
		Prompt:        task.Prompt,
		PromptVersion: "interviewer.v2",
		Task:          task,
		Run:           run,
		Result: runtimepkg.RunResultContext{
			Output: run.Output,
		},
		Resolved: runtimepkg.RunResolvedContext{
			Interview: runtimepkg.RunInterviewContext{
				Skill: protocol.SkillSpec{
					Name:        "Go Agent",
					Description: "Go-based agent engineering interviews",
				},
				Rubric: protocol.Rubric{
					Title:   "Go Agent Rubric",
					Anchors: []string{"并发控制", "可观测性"},
				},
				Profile: profile,
			},
		},
	}

	if err := app.completePostInterviewEvaluation(context.Background(), runCtx); err != nil {
		t.Fatalf("completePostInterviewEvaluation returned error: %v", err)
	}

	updatedRun, persistedMessages, events, err := app.GetRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("GetRun returned error: %v", err)
	}
	if updatedRun.Status != protocol.RunCompleted {
		t.Fatalf("expected completed run, got %s", updatedRun.Status)
	}
	if updatedRun.Phase != protocol.RunPhaseCompleted {
		t.Fatalf("expected completed phase, got %s", updatedRun.Phase)
	}
	if updatedRun.CompletedAt == nil || updatedRun.CompletedAt.IsZero() {
		t.Fatalf("expected completedAt to be set")
	}
	if updatedRun.Output == run.Output {
		t.Fatalf("expected final run output to be replaced with wrapup summary, got %q", updatedRun.Output)
	}
	if !strings.Contains(updatedRun.Output, "总评：候选人对并发控制掌握扎实") {
		t.Fatalf("expected final run output to contain score summary, got %q", updatedRun.Output)
	}
	if !strings.Contains(updatedRun.Output, "综合得分：82/100") {
		t.Fatalf("expected final run output to contain overall score, got %q", updatedRun.Output)
	}
	if len(persistedMessages) != 3 {
		t.Fatalf("expected 3 persisted messages including final wrapup, got %d", len(persistedMessages))
	}
	if persistedMessages[len(persistedMessages)-1].Content != updatedRun.Output {
		t.Fatalf("expected final persisted assistant message to match run output")
	}
	if !persistedMessages[len(persistedMessages)-1].CreatedAt.After(persistedMessages[len(persistedMessages)-2].CreatedAt) {
		t.Fatalf("expected final wrapup message to be recorded after previous assistant output")
	}

	review, err := app.GetReviewSnapshot(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("GetReviewSnapshot returned error: %v", err)
	}
	if review.Scorecard == nil {
		t.Fatalf("expected scorecard in review snapshot")
	}
	if review.Scorecard.OverallScore != 82 || review.Scorecard.OverallMaxScore != 100 {
		t.Fatalf("expected persisted overall score, got %#v", review.Scorecard)
	}
	if review.Profile == nil {
		t.Fatalf("expected profile in review snapshot")
	}
	if review.Profile.ID != "run:"+run.ID {
		t.Fatalf("expected review profile to be run-scoped, got %#v", review.Profile)
	}
	if review.Profile.InterviewCount != 1 {
		t.Fatalf("expected run-scoped profile interview count to be isolated, got %#v", review.Profile)
	}
	savedGlobal, err := app.profiles.(*memoryProfileRepo).Get(context.Background(), "global")
	if err != nil {
		t.Fatalf("expected global profile to be persisted: %v", err)
	}
	if savedGlobal.ID != "global" || savedGlobal.InterviewCount != 1 {
		t.Fatalf("expected global profile to be updated separately, got %#v", savedGlobal)
	}
	if len(review.Agents) != 2 {
		t.Fatalf("expected 2 agent executions, got %d", len(review.Agents))
	}
	if review.Agents[0].Role != protocol.AgentRoleInterviewer || review.Agents[1].Role != protocol.AgentRoleEvaluator {
		t.Fatalf("unexpected agent roles: %#v", review.Agents)
	}
	if review.Agents[0].PromptVersion != "interviewer.v2" {
		t.Fatalf("expected interviewer prompt version, got %q", review.Agents[0].PromptVersion)
	}
	if review.Agents[1].Status != protocol.RunCompleted {
		t.Fatalf("expected evaluator status completed, got %s", review.Agents[1].Status)
	}
	if review.Trace == nil || review.Trace.QuestionCount != 3 {
		t.Fatalf("expected archived trace tree, got %#v", review.Trace)
	}

	assertEventRecorded(t, events, protocol.EventScoreGenerated)
	assertEventRecorded(t, events, protocol.EventProfileUpdated)
	assertEventRecorded(t, events, protocol.EventReviewGenerated)
	assertEventRecorded(t, events, protocol.EventRunCompleted)
}

func TestShouldCompletePostInterviewEvaluationWithoutInterviewerOutputForExplicitWrapup(t *testing.T) {
	t.Parallel()

	runCtx := &runtimepkg.RunContext{
		Run: protocol.Run{
			Phase: protocol.RunPhaseEvaluating,
			Input: "请结束这场面试并给我最终评分。",
		},
		Task: protocol.Task{
			Config: protocol.InterviewConfig{
				OutputStyle: protocol.OutputInterviewPlusScore,
			}.WithDefaults(),
		},
		Resolved: runtimepkg.RunResolvedContext{
			Interview: runtimepkg.RunInterviewContext{
				Rubric: protocol.Rubric{Anchors: []string{"并发控制"}},
			},
		},
	}

	if !shouldCompletePostInterviewEvaluation(runCtx) {
		t.Fatalf("expected explicit wrapup request in evaluating phase to complete post interview evaluation")
	}
}

func TestCompletePostInterviewEvaluationReusesExistingAssistantScorecard(t *testing.T) {
	t.Parallel()

	now := time.Now()
	task := protocol.Task{
		ID:             "task_reuse",
		ConversationID: "conv_reuse",
		Prompt:         "请继续这场 Go agent 面试。",
		Config: protocol.InterviewConfig{
			Skill:       "go-agent",
			Persona:     protocol.PersonaRigorous,
			Mode:        protocol.ModeStandard,
			OutputStyle: protocol.OutputInterviewPlusStudy,
			TimeBudget:  "25 minutes",
		}.WithDefaults(),
	}
	run := protocol.Run{
		ID:             "run_reuse",
		ConversationID: "conv_reuse",
		TaskID:         task.ID,
		Status:         protocol.RunRunning,
		Phase:          protocol.RunPhaseEvaluating,
		Input:          "请结束并评分",
		Output:         "先前的 run output 会被最终 wrapup 覆盖。",
		CreatedAt:      now.Add(-4 * time.Minute),
		UpdatedAt:      now.Add(-10 * time.Second),
		InterviewState: &protocol.RunInterviewState{
			Phase: protocol.PhaseWrapup,
			LastDecision: &protocol.NextStepDecision{
				Reason:      protocol.ReasonWrapupDueToBudget,
				Explanation: "用户明确要求结束。",
			},
		},
	}
	messages := []protocol.Message{
		{
			ID:             "msg_user_reuse",
			ConversationID: run.ConversationID,
			TaskID:         run.TaskID,
			RunID:          run.ID,
			Role:           "user",
			Content:        "我会用 errgroup 管理 fan-out，用 context 统一取消，并把重试封在幂等边界内。",
			CreatedAt:      now.Add(-2 * time.Minute),
		},
		{
			ID:             "msg_assistant_reuse",
			ConversationID: run.ConversationID,
			TaskID:         run.TaskID,
			RunID:          run.ID,
			Role:           "assistant",
			Content: `面试到这里结束，下面是本场总结。

**综合评分：优秀 (84/100)**

## 维度评分
| 维度 | 分数 | 说明 |
| --- | --- | --- |
| 并发控制 | 9/10 | 能把 errgroup、取消和幂等边界串起来 |
| 可观测性 | 8/10 | 能说明 metrics、trace 和日志的配合 |

## 亮点
1. 能把调度、取消和失败隔离联系起来

## 短板与改进建议
1. 告警收敛策略不够具体
2. 补充错误预算和降级分层的取舍

## 学习计划建议
1. 复盘一次生产任务编排系统的告警降噪设计
2. 练习把 tradeoff 压缩成 2 分钟口头表达`,
			CreatedAt: now.Add(-30 * time.Second),
		},
	}

	scorecardCalls := 0
	app := &App{
		conversations: &memoryConversationRepo{
			items: map[string]protocol.Conversation{
				run.ConversationID: {
					ID:        run.ConversationID,
					Title:     "Timeout Fallback",
					Status:    "active",
					CreatedAt: now.Add(-4 * time.Minute),
					UpdatedAt: now.Add(-time.Minute),
				},
			},
		},
		tasks: &memoryTaskRepo{items: map[string]protocol.Task{task.ID: task}},
		runs:  &memoryRunRepo{items: map[string]protocol.Run{run.ID: run}},
		messages: &memoryMessageRepo{
			byRun: map[string][]protocol.Message{run.ID: messages},
		},
		events: &memoryEventRepo{byRun: make(map[string][]protocol.Event)},
		profiles: &memoryProfileRepo{
			items: map[string]protocol.CandidateProfile{
				"global": {ID: "global"},
			},
		},
		broker: NewEventBroker(),
		scorecards: func(
			ctx context.Context,
			transcript string,
			cfg protocol.InterviewConfig,
			modelCfg protocol.ModelConfig,
			skill protocol.SkillSpec,
			rubric protocol.Rubric,
		) (protocol.Scorecard, error) {
			scorecardCalls++
			return protocol.Scorecard{}, nil
		},
	}

	runCtx := &runtimepkg.RunContext{
		Context: context.Background(),
		Task:    task,
		Run:     run,
		Result: runtimepkg.RunResultContext{
			Output: run.Output,
		},
		Resolved: runtimepkg.RunResolvedContext{
			Interview: runtimepkg.RunInterviewContext{
				Skill: protocol.SkillSpec{Name: "Go Agent"},
				Rubric: protocol.Rubric{
					Title:   "Go Agent Rubric",
					Anchors: []string{"并发控制", "可观测性"},
				},
			},
		},
	}

	if err := app.completePostInterviewEvaluation(context.Background(), runCtx); err != nil {
		t.Fatalf("completePostInterviewEvaluation returned error: %v", err)
	}
	if scorecardCalls != 0 {
		t.Fatalf("expected reusable assistant scorecard to skip generator, got %d calls", scorecardCalls)
	}

	updatedRun, _, _, err := app.GetRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("GetRun returned error: %v", err)
	}
	if updatedRun.Phase != protocol.RunPhaseCompleted {
		t.Fatalf("expected completed phase, got %s", updatedRun.Phase)
	}
	if updatedRun.Output != strings.TrimSpace(messages[1].Content) {
		t.Fatalf("expected final output to preserve structured assistant evaluation, got %q", updatedRun.Output)
	}

	review, err := app.GetReviewSnapshot(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("GetReviewSnapshot returned error: %v", err)
	}
	if review.Scorecard == nil || review.Scorecard.OverallScore != 84 || review.Scorecard.OverallMaxScore != 100 {
		t.Fatalf("expected reusable scorecard to be persisted, got %#v", review.Scorecard)
	}
	if len(review.Scorecard.DimensionScores) != 2 {
		t.Fatalf("expected markdown dimensions to be parsed, got %#v", review.Scorecard.DimensionScores)
	}
	if len(review.Scorecard.StudyPlan) != 2 {
		t.Fatalf("expected study plan to be parsed from markdown sections, got %#v", review.Scorecard.StudyPlan)
	}
}

func TestCompletePostInterviewEvaluationFallsBackWhenScorecardTimesOut(t *testing.T) {
	t.Setenv("EVALUATION_TIMEOUT_SECONDS", "1")

	now := time.Now()
	task := protocol.Task{
		ID:             "task_timeout",
		ConversationID: "conv_timeout",
		Prompt:         "请继续这场 Go agent 面试。",
		Config: protocol.InterviewConfig{
			Skill:       "go-agent",
			Persona:     protocol.PersonaRigorous,
			Mode:        protocol.ModeSystemDesign,
			OutputStyle: protocol.OutputInterviewPlusScore,
			TimeBudget:  "45 分钟",
		}.WithDefaults(),
	}
	run := protocol.Run{
		ID:             "run_timeout",
		ConversationID: "conv_timeout",
		TaskID:         task.ID,
		Status:         protocol.RunRunning,
		Phase:          protocol.RunPhaseEvaluating,
		Input:          "请现在结束这场面试并给我评分。",
		Output:         "准备进入总结阶段。",
		CreatedAt:      now.Add(-3 * time.Minute),
		UpdatedAt:      now.Add(-15 * time.Second),
		InterviewState: &protocol.RunInterviewState{
			Phase: protocol.PhaseWrapup,
		},
		TraceTree: &protocol.InterviewTraceTree{
			QuestionCount: 2,
			Nodes: []protocol.InterviewTraceNode{
				{Round: 1, Phase: protocol.PhaseProbe, WeakSignals: []string{"missing_tradeoff"}},
				{Round: 2, Phase: protocol.PhaseWrapup, WeakSignals: []string{"missing_implementation_detail", "missing_observability_detail"}},
			},
		},
	}
	messages := []protocol.Message{
		{
			ID:             "msg_user_timeout",
			ConversationID: run.ConversationID,
			TaskID:         run.TaskID,
			RunID:          run.ID,
			Role:           "user",
			Content:        "我会先拆成 planner、executor、worker，但代码细节先略过。",
			CreatedAt:      now.Add(-2 * time.Minute),
		},
	}

	app := &App{
		conversations: &memoryConversationRepo{
			items: map[string]protocol.Conversation{
				run.ConversationID: {
					ID:        run.ConversationID,
					Title:     "Timeout Fallback",
					Status:    "active",
					CreatedAt: now.Add(-4 * time.Minute),
					UpdatedAt: now.Add(-time.Minute),
				},
			},
		},
		tasks: &memoryTaskRepo{items: map[string]protocol.Task{task.ID: task}},
		runs:  &memoryRunRepo{items: map[string]protocol.Run{run.ID: run}},
		messages: &memoryMessageRepo{
			byRun: map[string][]protocol.Message{run.ID: messages},
		},
		events: &memoryEventRepo{byRun: make(map[string][]protocol.Event)},
		profiles: &memoryProfileRepo{
			items: map[string]protocol.CandidateProfile{"global": {ID: "global"}},
		},
		broker: NewEventBroker(),
		scorecards: func(
			ctx context.Context,
			transcript string,
			cfg protocol.InterviewConfig,
			modelCfg protocol.ModelConfig,
			skill protocol.SkillSpec,
			rubric protocol.Rubric,
		) (protocol.Scorecard, error) {
			<-ctx.Done()
			return protocol.Scorecard{}, ctx.Err()
		},
	}

	runCtx := &runtimepkg.RunContext{
		Context: context.Background(),
		Task:    task,
		Run:     run,
		Result: runtimepkg.RunResultContext{
			Output: run.Output,
		},
		Resolved: runtimepkg.RunResolvedContext{
			Interview: runtimepkg.RunInterviewContext{
				Skill:  protocol.SkillSpec{Name: "Go Agent"},
				Rubric: protocol.Rubric{Title: "Go Agent Rubric", Anchors: []string{"system design", "observability"}},
			},
		},
	}

	if err := app.completePostInterviewEvaluation(context.Background(), runCtx); err != nil {
		t.Fatalf("completePostInterviewEvaluation returned error: %v", err)
	}

	updatedRun, _, events, err := app.GetRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("GetRun returned error: %v", err)
	}
	if updatedRun.Status != protocol.RunCompleted || updatedRun.Phase != protocol.RunPhaseCompleted {
		t.Fatalf("expected timeout fallback to complete the run, got %#v", updatedRun)
	}

	review, err := app.GetReviewSnapshot(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("GetReviewSnapshot returned error: %v", err)
	}
	if review.Scorecard == nil || review.Scorecard.OverallScore <= 0 {
		t.Fatalf("expected fallback scorecard to be persisted, got %#v", review.Scorecard)
	}
	if len(review.Scorecard.DimensionScores) == 0 {
		t.Fatalf("expected fallback scorecard dimensions, got %#v", review.Scorecard)
	}
	assertEventRecorded(t, events, protocol.EventScoreGenerated)
	assertEventRecorded(t, events, protocol.EventRunCompleted)
}

func TestGetRunSortsMessagesAndEventsChronologically(t *testing.T) {
	t.Parallel()

	now := time.Now()
	run := protocol.Run{
		ID:             "run_sort",
		ConversationID: "conv_sort",
		TaskID:         "task_sort",
	}
	app := &App{
		runs: &memoryRunRepo{
			items: map[string]protocol.Run{run.ID: run},
		},
		messages: &memoryMessageRepo{
			byRun: map[string][]protocol.Message{
				run.ID: {
					{ID: "m2", RunID: run.ID, Role: "assistant", Content: "second", CreatedAt: now.Add(2 * time.Second)},
					{ID: "m1", RunID: run.ID, Role: "user", Content: "first", CreatedAt: now},
					{ID: "m3", RunID: run.ID, Role: "assistant", Content: "third", CreatedAt: now.Add(3 * time.Second)},
				},
			},
		},
		events: &memoryEventRepo{
			byRun: map[string][]protocol.Event{
				run.ID: {
					{ID: "e2", RunID: run.ID, Type: protocol.EventMessageCompleted, Timestamp: now.Add(2 * time.Second)},
					{ID: "e1", RunID: run.ID, Type: protocol.EventRunStarted, Timestamp: now},
					{ID: "e3", RunID: run.ID, Type: protocol.EventRunCompleted, Timestamp: now.Add(3 * time.Second)},
				},
			},
		},
	}

	_, messages, events, err := app.GetRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("GetRun returned error: %v", err)
	}
	if len(messages) != 3 || messages[0].ID != "m1" || messages[2].ID != "m3" {
		t.Fatalf("expected messages to be sorted chronologically, got %#v", messages)
	}
	if len(events) != 3 || events[0].ID != "e1" || events[2].ID != "e3" {
		t.Fatalf("expected events to be sorted chronologically, got %#v", events)
	}
}

func assertEventRecorded(t *testing.T, events []protocol.Event, eventType protocol.EventType) {
	t.Helper()

	for _, event := range events {
		if event.Type == eventType {
			return
		}
	}
	t.Fatalf("expected event %s, got %#v", eventType, events)
}

type memoryTaskRepo struct {
	items map[string]protocol.Task
}

func (r *memoryTaskRepo) Create(ctx context.Context, task protocol.Task) (protocol.Task, error) {
	_ = ctx
	if r.items == nil {
		r.items = make(map[string]protocol.Task)
	}
	r.items[task.ID] = task
	return task, nil
}

func (r *memoryTaskRepo) Get(ctx context.Context, id string) (protocol.Task, error) {
	_ = ctx
	task, ok := r.items[id]
	if !ok {
		return protocol.Task{}, fmt.Errorf("task %s not found", id)
	}
	return task, nil
}

func (r *memoryTaskRepo) ListByConversation(ctx context.Context, conversationID string) ([]protocol.Task, error) {
	_ = ctx
	tasks := make([]protocol.Task, 0, len(r.items))
	for _, task := range r.items {
		if task.ConversationID == conversationID {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

type memoryRunRepo struct {
	items map[string]protocol.Run
}

func (r *memoryRunRepo) Create(ctx context.Context, run protocol.Run) (protocol.Run, error) {
	_ = ctx
	if r.items == nil {
		r.items = make(map[string]protocol.Run)
	}
	r.items[run.ID] = run
	return run, nil
}

func (r *memoryRunRepo) Update(ctx context.Context, run protocol.Run) (protocol.Run, error) {
	_ = ctx
	if r.items == nil {
		r.items = make(map[string]protocol.Run)
	}
	r.items[run.ID] = run
	return run, nil
}

func (r *memoryRunRepo) Get(ctx context.Context, id string) (protocol.Run, error) {
	_ = ctx
	run, ok := r.items[id]
	if !ok {
		return protocol.Run{}, fmt.Errorf("run %s not found", id)
	}
	return run, nil
}

func (r *memoryRunRepo) ListByConversation(ctx context.Context, conversationID string) ([]protocol.Run, error) {
	_ = ctx
	runs := make([]protocol.Run, 0, len(r.items))
	for _, run := range r.items {
		if run.ConversationID == conversationID {
			runs = append(runs, run)
		}
	}
	return runs, nil
}

type memoryMessageRepo struct {
	byRun map[string][]protocol.Message
}

func (r *memoryMessageRepo) Create(ctx context.Context, message protocol.Message) (protocol.Message, error) {
	_ = ctx
	if r.byRun == nil {
		r.byRun = make(map[string][]protocol.Message)
	}
	r.byRun[message.RunID] = append(r.byRun[message.RunID], message)
	return message, nil
}

func (r *memoryMessageRepo) ListByRun(ctx context.Context, runID string) ([]protocol.Message, error) {
	_ = ctx
	items := append([]protocol.Message(nil), r.byRun[runID]...)
	return items, nil
}

type memoryEventRepo struct {
	byRun map[string][]protocol.Event
}

func (r *memoryEventRepo) Create(ctx context.Context, event protocol.Event) (protocol.Event, error) {
	_ = ctx
	if r.byRun == nil {
		r.byRun = make(map[string][]protocol.Event)
	}
	r.byRun[event.RunID] = append(r.byRun[event.RunID], event)
	return event, nil
}

func (r *memoryEventRepo) ListByRun(ctx context.Context, runID string) ([]protocol.Event, error) {
	_ = ctx
	items := append([]protocol.Event(nil), r.byRun[runID]...)
	return items, nil
}

type memoryProfileRepo struct {
	items map[string]protocol.CandidateProfile
}

func (r *memoryProfileRepo) Get(ctx context.Context, id string) (protocol.CandidateProfile, error) {
	_ = ctx
	profile, ok := r.items[id]
	if !ok {
		return protocol.CandidateProfile{}, fmt.Errorf("profile %s not found", id)
	}
	return profile, nil
}

func (r *memoryProfileRepo) Save(ctx context.Context, profile protocol.CandidateProfile) (protocol.CandidateProfile, error) {
	_ = ctx
	if r.items == nil {
		r.items = make(map[string]protocol.CandidateProfile)
	}
	r.items[profile.ID] = profile
	return profile, nil
}
