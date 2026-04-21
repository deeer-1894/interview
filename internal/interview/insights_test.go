package interview

import (
	"testing"
	"time"

	"mockinterview/internal/protocol"
)

func TestBuildTraceTreeMarksProfileHits(t *testing.T) {
	t.Parallel()

	state := protocol.RunInterviewState{
		History: []protocol.InterviewRoundSnapshot{
			{
				Round:       1,
				Phase:       protocol.PhaseProbe,
				Reason:      string(protocol.ReasonWeakSignalObservability),
				Explanation: "系统会继续确认日志、指标、追踪和告警设计。",
				WeakSignals: []string{"missing_observability_detail"},
			},
		},
	}
	trace := BuildTraceTree(
		protocol.PersonaRigorous,
		[]protocol.Message{{ID: "u1", Role: "user", Content: "我会先讲架构。"}},
		protocol.Message{ID: "a1", RunID: "run_1", Role: "assistant", Content: "如果线上出问题，你会怎么补监控和 trace？"},
		&state,
		&protocol.CandidateProfile{RecurringGaps: []string{"observability"}},
	)

	if len(trace.Nodes) != 1 {
		t.Fatalf("expected 1 trace node, got %d", len(trace.Nodes))
	}
	if !trace.Nodes[0].ProfileHit {
		t.Fatalf("expected node to be marked as profile hit")
	}
	if len(trace.Nodes[0].FocusHits) == 0 || trace.Nodes[0].FocusHits[0] != "observability" {
		t.Fatalf("unexpected focus hits: %#v", trace.Nodes[0].FocusHits)
	}
}

func TestBuildTraceTreeSkipsWrapupSummaryMessages(t *testing.T) {
	t.Parallel()

	trace := BuildTraceTree(
		protocol.PersonaRigorous,
		[]protocol.Message{
			{ID: "a1", RunID: "run_1", Role: "assistant", Content: "请解释 errgroup 和 worker pool 的取舍。"},
			{ID: "u1", Role: "user", Content: "我会优先用 errgroup 管 fan-out，再补 timeout 和 cancel。"},
		},
		protocol.Message{
			ID:      "a2",
			RunID:   "run_1",
			Role:    "assistant",
			Content: "你的设计思路清晰，展现了高级工程师的系统设计能力，但在具体实现细节和编码实践方面还需要加强。",
		},
		&protocol.RunInterviewState{
			Phase: protocol.PhaseWrapup,
			Round: 6,
			History: []protocol.InterviewRoundSnapshot{
				{Round: 1, Phase: protocol.PhaseWarmup},
				{Round: 6, Phase: protocol.PhaseWrapup},
			},
		},
		nil,
	)

	if len(trace.Nodes) != 1 {
		t.Fatalf("expected wrapup summary to be excluded from trace, got %#v", trace.Nodes)
	}
	if trace.QuestionCount != 1 {
		t.Fatalf("expected trace question count to exclude wrapup summary, got %d", trace.QuestionCount)
	}
	if trace.Nodes[0].MessageID != "a1" {
		t.Fatalf("expected only the actual interview question to remain, got %#v", trace.Nodes)
	}
}

func TestBuildTraceTreeSkipsWrapupSummaryEvenWhenFollowedByUserReply(t *testing.T) {
	t.Parallel()

	now := time.Now()
	trace := BuildTraceTree(
		protocol.PersonaRigorous,
		[]protocol.Message{
			{ID: "a1", RunID: "run_1", Role: "assistant", Content: "请解释 errgroup 和 worker pool 的取舍。", CreatedAt: now.Add(-4 * time.Minute)},
			{ID: "u1", RunID: "run_1", Role: "user", Content: "我会优先用 errgroup 管 fan-out。", CreatedAt: now.Add(-3 * time.Minute)},
			{ID: "a2", RunID: "run_1", Role: "assistant", Content: "面试到这里结束，下面是本场总结。\n\n总评：并发设计思路扎实。\n\n下一步建议：\n- 创建一个 Grafana 仪表盘。", CreatedAt: now.Add(-2 * time.Minute)},
			{ID: "u2", RunID: "run_1", Role: "user", Content: "我想继续回答混沌测试。", CreatedAt: now.Add(-1 * time.Minute)},
		},
		protocol.Message{
			ID:        "a3",
			RunID:     "run_1",
			Role:      "assistant",
			Content:   "请具体说明你如何验证和缓解这些极端压力下的不可预测行为？",
			CreatedAt: now,
		},
		&protocol.RunInterviewState{
			Phase: protocol.PhaseWrapup,
			Round: 4,
			History: []protocol.InterviewRoundSnapshot{
				{Round: 1, Phase: protocol.PhaseWarmup},
				{Round: 4, Phase: protocol.PhaseWrapup},
			},
		},
		nil,
	)

	if len(trace.Nodes) != 2 {
		t.Fatalf("expected wrapup summary with future user reply to be excluded, got %#v", trace.Nodes)
	}
	if trace.Nodes[0].MessageID != "a1" || trace.Nodes[1].MessageID != "a3" {
		t.Fatalf("unexpected trace nodes after skipping wrapup summary: %#v", trace.Nodes)
	}
	if trace.Nodes[1].Round == 0 {
		t.Fatalf("expected unanswered follow-up to receive a non-zero round, got %#v", trace.Nodes[1])
	}
}

func TestBuildTraceTreeSkipsRogueQuestionAfterExplicitWrapupRequest(t *testing.T) {
	t.Parallel()

	trace := BuildTraceTree(
		protocol.PersonaRigorous,
		[]protocol.Message{
			{ID: "a1", RunID: "run_1", Role: "assistant", Content: "请说明你的 runtime 架构。"},
			{ID: "u1", RunID: "run_1", Role: "user", Content: "我会拆分 planner、orchestrator 和 checkpoint store。"},
			{ID: "a2", RunID: "run_1", Role: "assistant", Content: "请继续说明 timeout 和恢复策略。"},
			{ID: "u2", RunID: "run_1", Role: "user", Content: "如果按面试节奏，这轮答完后请结束本场面试，并直接给我最终评分、亮点、短板和学习建议。"},
			{ID: "a3", RunID: "run_1", Role: "assistant", Content: "3. 是否会触发人工介入，以及人工介入的触发条件和接口设计？"},
		},
		protocol.Message{
			ID:      "a4",
			RunID:   "run_1",
			Role:    "assistant",
			Content: "面试到这里结束，下面是本场总结。\n\n总评：架构和恢复设计都比较扎实。",
		},
		&protocol.RunInterviewState{
			Phase: protocol.PhaseWrapup,
			Round: 3,
			History: []protocol.InterviewRoundSnapshot{
				{Round: 1, Phase: protocol.PhaseWarmup},
				{Round: 2, Phase: protocol.PhaseProbe},
				{Round: 3, Phase: protocol.PhaseWrapup},
			},
		},
		nil,
	)

	if len(trace.Nodes) != 2 {
		t.Fatalf("expected rogue follow-up after explicit wrapup request to be excluded, got %#v", trace.Nodes)
	}
	if trace.Nodes[0].MessageID != "a1" || trace.Nodes[1].MessageID != "a2" {
		t.Fatalf("unexpected nodes after explicit wrapup request pruning: %#v", trace.Nodes)
	}
	if trace.QuestionCount != 2 {
		t.Fatalf("expected trace question count to exclude rogue follow-up, got %d", trace.QuestionCount)
	}
}

func TestMergeCandidateProfileBuildsNormalizedTrendData(t *testing.T) {
	t.Parallel()

	legacyUpdatedAt := time.Now().Add(-7 * 24 * time.Hour)
	profile := protocol.CandidateProfile{
		ID:             "global",
		InterviewCount: 2,
		UpdatedAt:      legacyUpdatedAt,
		Dimensions: []protocol.ProfileDimension{
			{
				Key:           "observability",
				Label:         "可观测性",
				Score:         2,
				EvidenceCount: 2,
			},
		},
	}
	cfg := protocol.InterviewConfig{
		Skill:   "go-agent",
		Persona: protocol.PersonaRigorous,
		Focus:   "observability",
	}
	skill := protocol.SkillSpec{
		FocusAreas: []string{"observability", "incident response"},
	}
	scorecard := protocol.Scorecard{
		Strengths:    []string{"监控、日志和 trace 设计比较完整"},
		Gaps:         []string{"incident response 流程和 runbook 还不够具体"},
		Improvements: []string{"继续补 incident response 演练方案"},
	}
	trace := protocol.InterviewTraceTree{
		QuestionCount: 2,
		Nodes: []protocol.InterviewTraceNode{
			{Question: "如果线上 incident response 升级，你会怎么组织值班和恢复？", Signal: "weak"},
			{Question: "你会如何补齐 observability 仪表盘和 trace？", Signal: "strong"},
		},
	}

	merged := MergeCandidateProfile(profile, cfg, skill, scorecard, trace)

	if len(merged.Dimensions) == 0 {
		t.Fatalf("expected dimensions to be generated")
	}
	if len(merged.Radar) == 0 {
		t.Fatalf("expected radar points to be generated")
	}
	if len(merged.GrowthCurves) == 0 {
		t.Fatalf("expected growth curves to be generated")
	}

	var observability protocol.ProfileDimension
	var incidentResponse protocol.ProfileDimension
	for _, dimension := range merged.Dimensions {
		switch dimension.Key {
		case "observability":
			observability = dimension
		case "incident_response":
			incidentResponse = dimension
		}
	}

	if observability.Key == "" {
		t.Fatalf("expected observability dimension to exist")
	}
	if observability.NormalizedScore <= 0 || observability.NormalizedScore > 100 {
		t.Fatalf("unexpected normalized score: %d", observability.NormalizedScore)
	}
	if observability.LastUpdatedAt == nil {
		t.Fatalf("expected last updated time to be populated")
	}
	if len(observability.Trend) < 2 {
		t.Fatalf("expected legacy profile to be migrated into trend history, got %#v", observability.Trend)
	}

	if incidentResponse.Key == "" {
		t.Fatalf("expected skill-level dimension registration to create incident_response dimension")
	}
	if incidentResponse.Label != "incident response" {
		t.Fatalf("unexpected dynamic dimension label: %s", incidentResponse.Label)
	}
}

func TestMergeCandidateProfileAppliesTimeDecay(t *testing.T) {
	t.Parallel()

	staleTime := time.Now().Add(-180 * 24 * time.Hour)
	profile := protocol.CandidateProfile{
		ID:        "global",
		UpdatedAt: staleTime,
		Dimensions: []protocol.ProfileDimension{
			{
				Key:           "reliability",
				Label:         "稳定性",
				Score:         6,
				EvidenceCount: 4,
				LastUpdatedAt: &staleTime,
			},
		},
	}
	cfg := protocol.InterviewConfig{
		Skill:   "backend",
		Persona: protocol.PersonaCalm,
		Focus:   "reliability",
	}

	merged := MergeCandidateProfile(profile, cfg, protocol.SkillSpec{}, protocol.Scorecard{}, protocol.InterviewTraceTree{})

	if len(merged.Dimensions) != 1 {
		t.Fatalf("expected 1 dimension after merge, got %d", len(merged.Dimensions))
	}
	if merged.Dimensions[0].Score >= 6 {
		t.Fatalf("expected stale score to decay, got %d", merged.Dimensions[0].Score)
	}
	if merged.Dimensions[0].NormalizedScore >= 100 {
		t.Fatalf("expected normalized score to reflect decay, got %d", merged.Dimensions[0].NormalizedScore)
	}
	if len(merged.Dimensions[0].Trend) == 0 {
		t.Fatalf("expected decayed dimension to keep trend history")
	}
}

func TestBuildReviewSummaryCapturesRoundsAndWeaknesses(t *testing.T) {
	t.Parallel()

	run := protocol.Run{
		InterviewState: &protocol.RunInterviewState{
			Phase: protocol.PhaseWrapup,
			LastDecision: &protocol.NextStepDecision{
				Reason:           protocol.ReasonWeakSignalObservability,
				Explanation:      "继续追问指标、日志和 trace。",
				RecommendedFocus: []string{"observability", "reliability"},
			},
		},
	}
	trace := protocol.InterviewTraceTree{
		Nodes: []protocol.InterviewTraceNode{
			{
				Round:       2,
				Adversarial: true,
				WeakSignals: []string{"missing_observability_detail"},
				FocusHits:   []string{"observability"},
			},
			{
				Round:       3,
				Pressure:    true,
				WeakSignals: []string{"missing_tradeoff", "missing_observability_detail"},
				FocusHits:   []string{"reliability", "observability"},
			},
			{
				Round: 4,
				Phase: protocol.PhaseWrapup,
			},
		},
	}
	scorecard := &protocol.Scorecard{
		Gaps:      []string{"可观测性细节不足"},
		Strengths: []string{"表达清晰"},
	}

	summary := BuildReviewSummary(
		run,
		protocol.InterviewConfig{Mode: protocol.ModeStress, Persona: protocol.PersonaRigorous},
		trace,
		scorecard,
		nil,
	)

	if summary.CurrentPhase != protocol.PhaseWrapup {
		t.Fatalf("expected current phase to be wrapup, got %q", summary.CurrentPhase)
	}
	if summary.AdversarialRound != 2 || summary.PressureRound != 3 || summary.WrapupRound != 4 {
		t.Fatalf("unexpected stage rounds: %#v", summary)
	}
	if summary.MostCommonWeakSignal != "missing_observability_detail" {
		t.Fatalf("unexpected weak signal summary: %#v", summary)
	}
	if len(summary.HistoricalWeaknessesHit) != 2 {
		t.Fatalf("expected historical weaknesses to be collected, got %#v", summary.HistoricalWeaknessesHit)
	}
	if len(summary.RecommendedFocus) != 2 {
		t.Fatalf("expected recommended focus from last decision, got %#v", summary.RecommendedFocus)
	}
	if len(summary.NewWeaknesses) != 1 || summary.NewWeaknesses[0] != "可观测性细节不足" {
		t.Fatalf("unexpected new weaknesses: %#v", summary.NewWeaknesses)
	}
	if len(summary.ResolvedWeaknesses) != 1 || summary.ResolvedWeaknesses[0] != "表达清晰" {
		t.Fatalf("unexpected resolved weaknesses: %#v", summary.ResolvedWeaknesses)
	}
}

func TestBuildReviewSummaryFallsBackToWrapupStateRound(t *testing.T) {
	t.Parallel()

	run := protocol.Run{
		InterviewState: &protocol.RunInterviewState{
			Phase: protocol.PhaseWrapup,
			Round: 3,
		},
	}
	trace := protocol.InterviewTraceTree{
		QuestionCount: 3,
		Nodes: []protocol.InterviewTraceNode{
			{Round: 1, Phase: protocol.PhaseWarmup},
			{Round: 2, Phase: protocol.PhaseProbe},
			{Round: 3, Phase: protocol.PhaseStress},
		},
	}

	summary := BuildReviewSummary(
		run,
		protocol.InterviewConfig{Mode: protocol.ModeSystemDesign, Persona: protocol.PersonaRigorous},
		trace,
		nil,
		nil,
	)

	if summary.CurrentPhase != protocol.PhaseWrapup {
		t.Fatalf("expected current phase wrapup, got %q", summary.CurrentPhase)
	}
	if summary.WrapupRound != 3 {
		t.Fatalf("expected wrapup round to fall back to state round, got %d", summary.WrapupRound)
	}
}

func TestEnsureProfileDimensionRulesPreservesLegacyDimensions(t *testing.T) {
	t.Parallel()

	rules := ensureProfileDimensionRules(
		resolveProfileDimensionRules(protocol.SkillSpec{FocusAreas: []string{"observability"}}),
		[]protocol.ProfileDimension{
			{
				Key:     "incident_response",
				Label:   "Incident Response",
				Summary: "历史维度",
			},
		},
	)

	found := false
	for _, rule := range rules {
		if rule.key != "incident_response" {
			continue
		}
		found = true
		if rule.label != "Incident Response" {
			t.Fatalf("unexpected legacy rule label: %#v", rule)
		}
		if rule.summary != "历史维度" {
			t.Fatalf("unexpected legacy rule summary: %#v", rule)
		}
		if len(rule.keywords) == 0 {
			t.Fatalf("expected legacy rule keywords to be populated: %#v", rule)
		}
	}

	if !found {
		t.Fatalf("expected legacy dimension rule to be preserved")
	}
}
