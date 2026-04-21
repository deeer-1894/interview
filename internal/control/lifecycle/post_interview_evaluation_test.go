package lifecycle

import (
	"testing"

	"mockinterview/internal/protocol"
)

func TestNormalizePostInterviewScorecardFiltersGarbage(t *testing.T) {
	t.Parallel()

	card := normalizePostInterviewScorecard(protocol.Scorecard{
		OverallScore: 91,
		OverallMaxScore: 100,
		Title:        "Go Agent Rubric",
		Summary:      "",
		Strengths:    []string{"系统设计扎实", "],"},
		Improvements: []string{"}", "```", "补充失败恢复策略"},
	})

	if card.OverallScore != 91 || card.OverallMaxScore != 100 {
		t.Fatalf("expected explicit overall score 91/100 to be preserved, got %#v", card)
	}
	if len(card.Strengths) != 1 || card.Strengths[0] != "系统设计扎实" {
		t.Fatalf("expected malformed strengths to be filtered, got %#v", card.Strengths)
	}
	if len(card.Improvements) != 1 || card.Improvements[0] != "补充失败恢复策略" {
		t.Fatalf("expected malformed improvements to be filtered, got %#v", card.Improvements)
	}
	if card.Summary == "" {
		t.Fatalf("expected fallback summary to be synthesized")
	}
}

func TestNormalizePostInterviewScorecardFiltersWrapupNoiseSections(t *testing.T) {
	t.Parallel()

	card := normalizePostInterviewScorecard(protocol.Scorecard{
		Strengths: []string{
			"架构分层清晰：职责边界明确",
			"画像分析",
		},
		Improvements: []string{
			"量化并发控制参数：补齐 limiter 和 breaker 的阈值推导",
			"本次面试采用了深度优先的追问策略。",
		},
	})

	if len(card.Strengths) != 1 || card.Strengths[0] != "架构分层清晰：职责边界明确" {
		t.Fatalf("expected wrapup noise strengths to be filtered, got %#v", card.Strengths)
	}
	if len(card.Improvements) != 1 || card.Improvements[0] != "量化并发控制参数：补齐 limiter 和 breaker 的阈值推导" {
		t.Fatalf("expected wrapup noise improvements to be filtered, got %#v", card.Improvements)
	}
}

func TestScorecardLooksReusableRejectsThinAssistantWrapup(t *testing.T) {
	t.Parallel()

	thin := protocol.Scorecard{
		Summary:      "建议下一步优先补一段 errgroup 并发执行代码。",
		OverallScore: 68,
		Improvements: []string{"补一段 errgroup 并发执行代码"},
	}
	if scorecardLooksReusable(thin) {
		t.Fatalf("expected thin assistant wrapup not to be reusable")
	}

	structured := protocol.Scorecard{
		Summary:      "架构分层清晰，但可观测性还可以更具体。",
		OverallScore: 85,
		Strengths:    []string{"架构分层清晰"},
		Gaps:         []string{"可观测性细节不足"},
		Improvements: []string{"补充 trace 和 metrics 设计"},
	}
	if !scorecardLooksReusable(structured) {
		t.Fatalf("expected structured assistant evaluation to remain reusable")
	}
}
