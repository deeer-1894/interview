package interview

import (
	"testing"

	"mockinterview/internal/protocol"
)

func TestAnalyzeAnswerSignalsFlagsEmptyAnswer(t *testing.T) {
	t.Parallel()

	analysis := AnalyzeAnswerSignals("   ", protocol.SkillSpec{})

	if !analysis.TooGeneric {
		t.Fatalf("expected empty answer to be flagged as too generic")
	}
	if !containsSignal(analysis.WeakSignals, "empty_answer") {
		t.Fatalf("expected empty_answer weak signal, got %#v", analysis.WeakSignals)
	}
	if analysis.WeakSignalConfidence["empty_answer"] != 1 {
		t.Fatalf("expected empty answer confidence to be 1, got %#v", analysis.WeakSignalConfidence)
	}
}

func TestAnalyzeAnswerSignalsFlagsGenericAnswer(t *testing.T) {
	t.Parallel()

	analysis := AnalyzeAnswerSignals("一般来说我会先看情况，然后按 best practice 推进。", protocol.SkillSpec{})

	if !analysis.TooGeneric {
		t.Fatalf("expected generic answer to be flagged")
	}
	if !containsSignal(analysis.WeakSignals, "too_generic") {
		t.Fatalf("expected too_generic weak signal, got %#v", analysis.WeakSignals)
	}
	if analysis.WeakSignalConfidence["too_generic"] < 0.6 {
		t.Fatalf("expected too_generic confidence, got %#v", analysis.WeakSignalConfidence)
	}
}

func TestAnalyzeAnswerSignalsDetectsTradeoffAndImplementation(t *testing.T) {
	t.Parallel()

	answer := "我会用 goroutine worker pool 控制并发，请求链路加 timeout 和 context cancel，同时比较 latency 与 consistency 的 tradeoff，再补 metric、trace 和日志告警。"
	analysis := AnalyzeAnswerSignals(answer, protocol.SkillSpec{FocusAreas: []string{"reliability"}})

	if !analysis.HasTradeoff {
		t.Fatalf("expected tradeoff detection")
	}
	if !analysis.HasConcreteImplementation {
		t.Fatalf("expected implementation detection")
	}
	if containsSignal(analysis.WeakSignals, "missing_tradeoff") {
		t.Fatalf("did not expect missing_tradeoff signal: %#v", analysis.WeakSignals)
	}
	if !containsSignal(analysis.StrongSignals, "observability") {
		t.Fatalf("expected observability strong signal: %#v", analysis.StrongSignals)
	}
	if analysis.StrongSignalConfidence["tradeoff_reasoning"] < 0.7 {
		t.Fatalf("expected tradeoff confidence to be populated, got %#v", analysis.StrongSignalConfidence)
	}
}

func TestAnalyzeAnswerSignalsUsesSemanticHeuristics(t *testing.T) {
	t.Parallel()

	answer := "我会先校验请求并落库，再由后台任务异步执行；如果下游变慢，我更偏向吞吐优先，接受结果稍晚返回，失败时走补偿流程。"
	analysis := AnalyzeAnswerSignals(answer, protocol.SkillSpec{})

	if !analysis.HasTradeoff {
		t.Fatalf("expected semantic tradeoff detection, got %#v", analysis)
	}
	if !analysis.HasConcreteImplementation {
		t.Fatalf("expected semantic implementation detection, got %#v", analysis)
	}
	if analysis.StrongSignalConfidence["implementation_detail"] < 0.65 {
		t.Fatalf("expected semantic implementation confidence, got %#v", analysis.StrongSignalConfidence)
	}
	if analysis.StrongSignalConfidence["tradeoff_reasoning"] < 0.6 {
		t.Fatalf("expected semantic tradeoff confidence, got %#v", analysis.StrongSignalConfidence)
	}
}

func TestAnalyzeAnswerSignalsFlagsConceptWithoutPlan(t *testing.T) {
	t.Parallel()

	answer := "我会先看整体思路和核心原则，重点关注稳定性、扩展性和边界，本质上先把方向判断清楚，再根据情况推进。"
	analysis := AnalyzeAnswerSignals(answer, protocol.SkillSpec{})

	if !containsSignal(analysis.WeakSignals, "concept_without_plan") {
		t.Fatalf("expected concept_without_plan weak signal, got %#v", analysis.WeakSignals)
	}
	if analysis.WeakSignalConfidence["concept_without_plan"] < 0.65 {
		t.Fatalf("expected concept_without_plan confidence, got %#v", analysis.WeakSignalConfidence)
	}
	if !analysis.TooGeneric {
		t.Fatalf("expected concept-only answer to still be considered too generic")
	}
}

func TestAnalyzeAnswerSignalsFlagsLackOfEvidence(t *testing.T) {
	t.Parallel()

	answer := "一般来说我会考虑稳定性、性能和可维护性，通常按 best practice 处理，经验上先把方案设计好再持续优化。"
	analysis := AnalyzeAnswerSignals(answer, protocol.SkillSpec{})

	if !containsSignal(analysis.WeakSignals, "lacks_example_or_evidence") {
		t.Fatalf("expected lacks_example_or_evidence weak signal, got %#v", analysis.WeakSignals)
	}
	if analysis.WeakSignalConfidence["lacks_example_or_evidence"] < 0.7 {
		t.Fatalf("expected evidence gap confidence, got %#v", analysis.WeakSignalConfidence)
	}
}

func TestAnalyzeAnswerSignalsDetectsFocusTimeoutAndObservability(t *testing.T) {
	t.Parallel()

	answer := "在 observability 这块，我会给核心请求链路配置 timeout 和 context cancel，并补 metrics、trace、日志和告警阈值。"
	analysis := AnalyzeAnswerSignals(answer, protocol.SkillSpec{FocusAreas: []string{"observability"}})

	if !containsSignal(analysis.StrongSignals, "focus:observability") {
		t.Fatalf("expected focus hit signal, got %#v", analysis.StrongSignals)
	}
	if !containsSignal(analysis.StrongSignals, "timeout_control") {
		t.Fatalf("expected timeout strong signal, got %#v", analysis.StrongSignals)
	}
	if !containsSignal(analysis.StrongSignals, "observability") {
		t.Fatalf("expected observability strong signal, got %#v", analysis.StrongSignals)
	}
}

func TestAnswerSignalAnalysisSnapshotClonesCollections(t *testing.T) {
	t.Parallel()

	analysis := AnalyzeAnswerSignals(
		"我会先用 worker 控制并发，再加 timeout、日志和 trace。",
		protocol.SkillSpec{FocusAreas: []string{"trace"}},
	)

	snapshot := analysis.Snapshot()
	if len(snapshot.StrongSignals) == 0 {
		t.Fatalf("expected strong signals to exist")
	}

	snapshot.StrongSignals[0] = "mutated"
	if snapshot.StrongSignalConfidence == nil {
		t.Fatalf("expected strong signal confidence to exist")
	}
	snapshot.StrongSignalConfidence["observability"] = 0.1

	if analysis.StrongSignals[0] == "mutated" {
		t.Fatalf("expected snapshot strong signals to be cloned")
	}
	if analysis.StrongSignalConfidence["observability"] == 0.1 {
		t.Fatalf("expected snapshot confidence map to be cloned")
	}
}
