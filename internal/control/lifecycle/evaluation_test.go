package lifecycle

import (
	"testing"

	"mockinterview/internal/protocol"
)

func TestShouldGenerateEvaluationRecognizesExplicitEndRequests(t *testing.T) {
	t.Parallel()

	cases := []string{
		"请结束这场面试并给我评分总结",
		"请现在结束这场面试，不要继续追问，直接给我最终评分。",
		"请模拟一场很短的 Go 面试，只问一个问题并结束。",
		"那你怎么不结束",
		"别继续了，结束面试吧",
		"wrap up the interview and give me final feedback",
	}

	for _, input := range cases {
		if !shouldGenerateEvaluation(input) {
			t.Fatalf("expected explicit end request to trigger evaluation: %q", input)
		}
	}
}

func TestShouldGenerateEvaluationIgnoresTechnicalReviewVocabulary(t *testing.T) {
	t.Parallel()

	cases := []string{
		"review snapshot 负责事后复盘，trace tree 负责问题链路分析。",
		"observability 我会拆成 logs、metrics、traces 三层，review summary 用于事后审计。",
		"decision audit 和 profile merge 可以异步执行，不阻塞主问答。",
		"memory.append 会带 run_id 和 content_hash，避免恢复时重复提交。",
		"不要把 interviewer 的追问输出和 evaluator 的最终评分混在一个生命周期里，否则很容易出现还在追问就被误标成完成。",
	}

	for _, input := range cases {
		if shouldGenerateEvaluation(input) {
			t.Fatalf("expected technical vocabulary not to trigger evaluation: %q", input)
		}
	}
}

func TestShouldGenerateEvaluationIgnoresLongTechnicalAnswerWithEvaluationVocabulary(t *testing.T) {
	t.Parallel()

	input := "我的设计会把它拆成 6 个核心组件：API Gateway、Planner、Workflow Orchestrator、Tool Router、State Store、Recovery & Evaluation Loop。\n\n" +
		"1. 核心组件\n" +
		"- Planner 负责把任务拆成 DAG。\n" +
		"- State Store 持久化 checkpoint 和 artifact。\n\n" +
		"```go\n" +
		"type Step struct {\n" +
		"    ID string\n" +
		"}\n\n" +
		"func ExecuteRun(runID string) error {\n" +
		"    return nil\n" +
		"}\n" +
		"```\n\n" +
		"如果要继续落地，我下一层会重点讲两件事：一是 planner 输出的数据结构怎么设计，二是如何保证工具调用的幂等性和一致性。"

	if shouldGenerateEvaluation(input) {
		t.Fatalf("expected long technical answer not to trigger evaluation: %q", input)
	}
}

func TestShouldGenerateEvaluationRecognizesDirectScoreRequestWithoutExplicitEndWord(t *testing.T) {
	t.Parallel()

	cases := []string{
		"请给我这场面试的最终评分、亮点、短板和学习计划。",
		"please give me final feedback and a score for this interview",
	}

	for _, input := range cases {
		if !shouldGenerateEvaluation(input) {
			t.Fatalf("expected direct scoring request to trigger evaluation: %q", input)
		}
	}
}

func TestShouldDeferEvaluationRespectsExplicitEndWithoutRubric(t *testing.T) {
	t.Parallel()

	shouldDefer := ShouldDeferEvaluation(
		protocol.InterviewConfig{
			TimeBudget:  "45 分钟",
			OutputStyle: protocol.OutputInterviewPlusScore,
		},
		"结束面试，并给我最终评分和学习建议。",
		[]protocol.Message{
			{Role: "assistant", Content: "请先介绍你最近做过的 agent runtime。"},
			{Role: "user", Content: "我会先讲调度和工具路由。"},
		},
		&protocol.RunInterviewState{
			Phase: protocol.PhaseProbe,
			Round: 1,
		},
		protocol.Rubric{},
	)

	if !shouldDefer {
		t.Fatalf("expected explicit end request to defer evaluation even when rubric is empty")
	}
}

func TestShouldGenerateEvaluationIgnoresFutureScoringExpectationInInitialPrompt(t *testing.T) {
	t.Parallel()

	cases := []string{
		"请模拟一场 Go agent 面试，并在最后给出结构化评分。",
		"请开始一场系统设计面试，最后再给我总结和学习计划。",
		"start a mock interview and give me a score at the end",
	}

	for _, input := range cases {
		if shouldGenerateEvaluation(input) {
			t.Fatalf("expected deferred end-of-interview scoring request not to trigger immediate evaluation: %q", input)
		}
	}
}

func TestShouldDeferEvaluationDoesNotEndLongInterviewTooEarly(t *testing.T) {
	t.Parallel()

	shouldDefer := ShouldDeferEvaluation(
		protocol.InterviewConfig{
			TimeBudget:  "45 分钟",
			OutputStyle: protocol.OutputInterviewPlusScore,
		},
		"继续回答这个问题。",
		[]protocol.Message{
			{Role: "assistant", Content: "请先介绍你最近设计过的 runtime。"},
			{Role: "assistant", Content: "请具体说明 timeout 和 cancel 机制。"},
			{Role: "assistant", Content: "请具体说明 observability 设计。"},
		},
		&protocol.RunInterviewState{
			Phase: protocol.PhaseAdversarial,
			Round: 3,
		},
		protocol.Rubric{Anchors: []string{"并发", "可观测性"}},
	)

	if shouldDefer {
		t.Fatalf("expected long-budget interview not to defer evaluation at round 3")
	}
}
