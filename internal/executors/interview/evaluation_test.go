package interview

import (
	"strings"
	"testing"

	"mockinterview/internal/protocol"
)

func TestBuildEvaluationScorecardParsesJSON(t *testing.T) {
	t.Parallel()

	output := `{"title":"Go Agent 面试评分","summary":"系统设计思路清晰，但 tradeoff 不够具体。","dimensionScores":[{"name":"System Design","score":4,"rationale":"架构拆解清楚"},{"name":"Tradeoffs","score":2,"rationale":"缺少替代方案比较"}],"strengths":["架构拆解清晰"],"gaps":["缺少 tradeoff 比较"],"improvements":["补充候选方案取舍"]}`
	card := buildEvaluationScorecard(output, protocol.Rubric{Title: "默认评分卡", Anchors: []string{"tradeoff", "reliability"}}, protocol.OutputInterviewPlusScore)

	if card.Title != "Go Agent 面试评分" {
		t.Fatalf("expected parsed title, got %q", card.Title)
	}
	if card.Summary == "" {
		t.Fatalf("expected summary to be populated")
	}
	if len(card.DimensionScores) != 2 {
		t.Fatalf("expected 2 dimension scores, got %d", len(card.DimensionScores))
	}
	if len(card.Gaps) != 1 || card.Gaps[0] != "缺少 tradeoff 比较" {
		t.Fatalf("unexpected gaps: %#v", card.Gaps)
	}
}

func TestBuildEvaluationScorecardParsesFencedJSON(t *testing.T) {
	t.Parallel()

	output := "```json\n{\"studyPlan\":[\"补充 timeout 与重试设计\",\"练习把结论收敛到 2 句话内\"]}\n```"
	card := buildEvaluationScorecard(output, protocol.Rubric{Title: "默认评分卡"}, protocol.OutputInterviewPlusStudy)

	if len(card.StudyPlan) != 2 {
		t.Fatalf("expected study plan to be parsed from fenced json, got %#v", card.StudyPlan)
	}
}

func TestBuildEvaluationScorecardFallsBackToSections(t *testing.T) {
	t.Parallel()

	output := `
Strengths:
- 表达清晰
Gaps:
- timeout 细节不足
Improvements:
- 说明 cancel 和 fallback
`
	card := buildEvaluationScorecard(output, protocol.Rubric{Title: "默认评分卡"}, protocol.OutputInterviewPlusScore)

	if len(card.Strengths) != 1 || card.Strengths[0] != "表达清晰" {
		t.Fatalf("unexpected strengths: %#v", card.Strengths)
	}
	if len(card.Improvements) != 1 || card.Improvements[0] != "说明 cancel 和 fallback" {
		t.Fatalf("unexpected improvements: %#v", card.Improvements)
	}
}

func TestBuildEvaluationScorecardRepairsConflictingSections(t *testing.T) {
	t.Parallel()

	output := `{"title":"Go Agent 面试评分","strengths":["表达清晰","timeout 细节不足"],"gaps":["timeout 细节不足"],"improvements":["补充 cancel 细节"]}`
	card := buildEvaluationScorecard(output, protocol.Rubric{Title: "默认评分卡"}, protocol.OutputInterviewPlusScore)

	if len(card.Strengths) != 1 || card.Strengths[0] != "表达清晰" {
		t.Fatalf("expected conflicting strength to be removed, got %#v", card.Strengths)
	}
	if card.Summary == "" {
		t.Fatalf("expected summary to be synthesized when missing")
	}
}

func TestBuildEvaluationScorecardSynthesizesSummaryFromFallbackSections(t *testing.T) {
	t.Parallel()

	output := `
Gaps:
- 缺少 tradeoff 比较
Improvements:
- 补充两种方案取舍
`
	card := buildEvaluationScorecard(output, protocol.Rubric{Title: "默认评分卡"}, protocol.OutputInterviewPlusScore)

	if card.Summary == "" {
		t.Fatalf("expected summary fallback to be populated")
	}
	if len(card.Gaps) != 1 || card.Gaps[0] != "缺少 tradeoff 比较" {
		t.Fatalf("unexpected gaps: %#v", card.Gaps)
	}
}

func TestParseEvaluationResultExtractsJSONFromWrapperText(t *testing.T) {
	t.Parallel()

	result, ok := parseEvaluationResult("下面是结果：\n{\"title\":\"Wrapped\",\"gaps\":[\"timeout 不足\"]}\n请查收。")

	if !ok {
		t.Fatalf("expected wrapped json to be parsed")
	}
	if result.Title != "Wrapped" {
		t.Fatalf("expected parsed title, got %q", result.Title)
	}
	if len(result.Gaps) != 1 || result.Gaps[0] != "timeout 不足" {
		t.Fatalf("unexpected gaps: %#v", result.Gaps)
	}
}

func TestNormalizeDimensionScoresClampsAndSkipsInvalidNames(t *testing.T) {
	t.Parallel()

	scores := normalizeDimensionScores([]protocol.DimensionScore{
		{Name: "  ", Score: 4, Rationale: "ignored"},
		{Name: "System Design", Score: 0, Rationale: "  underflow  "},
		{Name: "Reliability", Score: 7, Rationale: " overflow "},
	})

	if len(scores) != 2 {
		t.Fatalf("expected 2 normalized scores, got %#v", scores)
	}
	if scores[0].Score != 1 {
		t.Fatalf("expected lower clamp to 1, got %#v", scores[0])
	}
	if scores[0].Rationale != "underflow" {
		t.Fatalf("expected rationale to be trimmed, got %#v", scores[0])
	}
	if scores[1].Score != 7 || scores[1].MaxScore != 10 {
		t.Fatalf("expected score to preserve 10-point scale, got %#v", scores[1])
	}
}

func TestBuildEvaluationScorecardParsesNarrativeOverallAndDimensionScores(t *testing.T) {
	t.Parallel()

	output := `
总评：系统设计思路比较完整，但容错细节还可以更具体。
最终评分：82/100
- **并发与容错设计**：8/10 覆盖了 errgroup、取消和失败隔离
- **可观测性**：7/10 提到了 metrics 和 trace，但告警收敛略泛
Strengths:
- 架构拆解清晰
Gaps:
- 对告警分层讲得不够细
`

	card := buildEvaluationScorecard(output, protocol.Rubric{Title: "默认评分卡"}, protocol.OutputInterviewPlusScore)

	if card.OverallScore != 82 || card.OverallMaxScore != 100 {
		t.Fatalf("expected overall score 82/100, got %#v", card)
	}
	if len(card.DimensionScores) != 2 {
		t.Fatalf("expected 2 parsed dimensions, got %#v", card.DimensionScores)
	}
	if card.DimensionScores[0].Name != "并发与容错设计" || card.DimensionScores[0].Score != 8 || card.DimensionScores[0].MaxScore != 10 {
		t.Fatalf("unexpected first dimension: %#v", card.DimensionScores[0])
	}
	if card.DimensionScores[1].Name != "可观测性" || card.DimensionScores[1].Score != 7 || card.DimensionScores[1].MaxScore != 10 {
		t.Fatalf("unexpected second dimension: %#v", card.DimensionScores[1])
	}
}

func TestBuildEvaluationScorecardUsesImprovementsAsStudyPlanFallback(t *testing.T) {
	t.Parallel()

	output := `{"title":"Study","improvements":["补 timeout 设计","练习更短结论"]}`
	card := buildEvaluationScorecard(output, protocol.Rubric{Title: "默认评分卡"}, protocol.OutputInterviewPlusStudy)

	if len(card.StudyPlan) != 2 {
		t.Fatalf("expected improvements to backfill study plan, got %#v", card.StudyPlan)
	}
	if card.StudyPlan[0] != "补 timeout 设计" {
		t.Fatalf("unexpected study plan content: %#v", card.StudyPlan)
	}
}

func TestBuildEvaluationScorecardParsesMarkdownChineseEvaluation(t *testing.T) {
	t.Parallel()

	output := `
面试到这里结束，下面是本场总结。

**综合评分：优秀 (92/100)**

## 维度评分
| 维度 | 分数 | 说明 |
| --- | --- | --- |
| 系统设计能力 | 9.5/10 | 架构拆解清楚，也能覆盖 tradeoff。 |
| 可观测性与稳定性 | 8.5/10 | 能说明 metrics、trace、日志与恢复策略。 |

## 亮点
1. 能把调度、取消和失败隔离讲清楚
2. 对恢复路径和幂等边界有明确意识

## 短板与改进建议
1. 对告警分层和噪音治理还可以更具体
2. 还可以再补充容量评估与压测闭环

## 学习计划建议
1. 练习把方案 tradeoff 压缩成 2 分钟表达
2. 复盘一次高并发任务编排系统的告警治理设计
`

	card := buildEvaluationScorecard(output, protocol.Rubric{Title: "默认评分卡"}, protocol.OutputInterviewPlusStudy)

	if card.OverallScore != 92 || card.OverallMaxScore != 100 {
		t.Fatalf("expected overall score 92/100, got %#v", card)
	}
	if len(card.DimensionScores) != 2 {
		t.Fatalf("expected 2 parsed dimension scores, got %#v", card.DimensionScores)
	}
	if card.DimensionScores[0].Name != "系统设计能力" || card.DimensionScores[0].Score != 10 || card.DimensionScores[0].MaxScore != 10 {
		t.Fatalf("unexpected first dimension: %#v", card.DimensionScores[0])
	}
	if card.DimensionScores[1].Name != "可观测性与稳定性" || card.DimensionScores[1].Score != 9 || card.DimensionScores[1].MaxScore != 10 {
		t.Fatalf("unexpected second dimension: %#v", card.DimensionScores[1])
	}
	if len(card.Strengths) != 2 || card.Strengths[0] != "能把调度、取消和失败隔离讲清楚" {
		t.Fatalf("unexpected strengths: %#v", card.Strengths)
	}
	if len(card.Gaps) != 2 || card.Gaps[0] != "对告警分层和噪音治理还可以更具体" {
		t.Fatalf("unexpected gaps: %#v", card.Gaps)
	}
	if len(card.StudyPlan) != 2 || card.StudyPlan[0] != "练习把方案 tradeoff 压缩成 2 分钟表达" {
		t.Fatalf("unexpected study plan: %#v", card.StudyPlan)
	}
}

func TestBuildEvaluationScorecardSanitizesMalformedFragments(t *testing.T) {
	t.Parallel()

	output := "Strengths:\n" +
		"- \"系统设计能力扎实，能从宏观层面规划 Agent Runtime 架构。\",\n" +
		"- \"对 Go 并发和状态管理有深入理解。\",\n" +
		"- ],\n" +
		"Improvements:\n" +
		"- }\n" +
		"- ```\n" +
		"- 补一版更明确的学习计划\n"

	card := buildEvaluationScorecard(output, protocol.Rubric{Title: "默认评分卡"}, protocol.OutputInterviewPlusStudy)

	for _, item := range append(append([]string(nil), card.Strengths...), card.Improvements...) {
		if item == "}" || item == "```" || item == "]," {
			t.Fatalf("expected malformed fragments to be filtered, got %#v", card)
		}
	}
	if !strings.Contains(strings.Join(card.Improvements, "\n"), "补一版更明确的学习计划") {
		t.Fatalf("expected meaningful improvement content to be retained, got %#v", card.Improvements)
	}
}

func TestBuildEvaluationScorecardStopsAtStrategySectionsAndStripsMarkdown(t *testing.T) {
	t.Parallel()

	output := `
## 亮点
1. **架构分层清晰**：执行层使用 goroutine，生命周期层显式建模

## 短板
1. **部分成功结果处理**：实现细节还可以更深入

## 学习计划建议
1. **量化并发控制参数**：补齐 limiter 和 breaker 的阈值推导

## 策略建议：
1. 继续强化生产系统设计经验

## 追问树总结：
执行架构 → 恢复机制 → 错误分类

## 候选人画像：
高级 Go Agent 工程师

## 阶段过程总结：
整体表现优秀，达到了高级 Go Agent 工程师的水平。
`

	card := buildEvaluationScorecard(output, protocol.Rubric{Title: "默认评分卡"}, protocol.OutputInterviewPlusStudy)

	if len(card.Strengths) != 1 || card.Strengths[0] != "架构分层清晰：执行层使用 goroutine，生命周期层显式建模" {
		t.Fatalf("unexpected strengths after markdown sanitization: %#v", card.Strengths)
	}
	if len(card.Gaps) != 1 || card.Gaps[0] != "部分成功结果处理：实现细节还可以更深入" {
		t.Fatalf("unexpected gaps after markdown sanitization: %#v", card.Gaps)
	}
	if len(card.StudyPlan) != 1 || card.StudyPlan[0] != "量化并发控制参数：补齐 limiter 和 breaker 的阈值推导" {
		t.Fatalf("unexpected study plan after section stop: %#v", card.StudyPlan)
	}
	joinedImprovements := strings.Join(card.Improvements, "\n")
	joinedStudyPlan := strings.Join(card.StudyPlan, "\n")
	for _, noise := range []string{"策略建议", "追问树总结", "候选人画像", "阶段过程总结", "整体表现优秀"} {
		if strings.Contains(joinedImprovements, noise) || strings.Contains(joinedStudyPlan, noise) {
			t.Fatalf("expected strategy/profile sections to be excluded, got %#v / %#v", card.Improvements, card.StudyPlan)
		}
	}
	if len(card.DimensionScores) != 0 {
		t.Fatalf("did not expect strategy summary headings to turn into dimensions, got %#v", card.DimensionScores)
	}
}

func TestParseDimensionScoreLineIgnoresOverallScoreHeading(t *testing.T) {
	t.Parallel()

	if score, ok := parseDimensionScoreLine("## 最终总分：4.2/5"); ok {
		t.Fatalf("expected overall score heading to be ignored, got %#v", score)
	}
	if score, ok := parseDimensionScoreLine("- **Go基础与生产实现能力：4.5/5** - 对Go并发有扎实理解"); !ok || score.Name != "Go基础与生产实现能力" {
		t.Fatalf("expected markdown dimension line to parse cleanly, got %#v ok=%v", score, ok)
	} else if score.Score != 9 || score.MaxScore != 10 {
		t.Fatalf("expected 5-point scale to normalize into 10-point scale, got %#v", score)
	}
}

func TestBuildScoringInstructionIncludesStructuredConstraints(t *testing.T) {
	t.Parallel()

	instruction := buildScoringInstruction(
		protocol.InterviewConfig{Level: "senior", Focus: "observability", Mode: protocol.ModeStress},
		protocol.SkillSpec{Name: "go-agent", Description: "Go agent engineering"},
		protocol.Rubric{Anchors: []string{"tradeoff", "reliability"}},
	)

	for _, fragment := range []string{
		"Return valid JSON only.",
		"Target level: senior",
		"Focus: observability",
		"Mode: stress",
		"Skill: go-agent",
		"Skill description: Go agent engineering",
		`"overallScore":82`,
		`"maxScore":10`,
		"- tradeoff",
		"- reliability",
	} {
		if !strings.Contains(instruction, fragment) {
			t.Fatalf("expected instruction to contain %q, got %q", fragment, instruction)
		}
	}
}

func TestStudyPlanInstructionAndPromptIncludeContext(t *testing.T) {
	t.Parallel()

	instruction := buildStudyPlanInstruction(
		protocol.InterviewConfig{Level: "staff"},
		protocol.SkillSpec{Name: "go-agent"},
		protocol.Rubric{Anchors: []string{"observability", "tradeoff"}},
	)
	for _, fragment := range []string{
		"Generate only a short learning plan",
		`{"studyPlan":["string"]}`,
		"Target level: staff",
		"Skill: go-agent",
		"- observability",
	} {
		if !strings.Contains(instruction, fragment) {
			t.Fatalf("expected study instruction to contain %q, got %q", fragment, instruction)
		}
	}

	prompt := buildStudyPlanPrompt("User: talk through the design", protocol.Scorecard{
		Gaps:         []string{"timeout 细节不足"},
		Improvements: []string{"补充熔断和降级"},
	})
	if !strings.Contains(prompt, "Interview transcript:") || !strings.Contains(prompt, "- Gap: timeout 细节不足") {
		t.Fatalf("unexpected study plan prompt: %q", prompt)
	}
	if !strings.Contains(prompt, "- Improvement: 补充熔断和降级") {
		t.Fatalf("expected improvements in study plan prompt, got %q", prompt)
	}
}

func TestBuildScoringPromptAndFallbackBullets(t *testing.T) {
	t.Parallel()

	prompt := buildScoringPrompt("User: explain the retry policy")
	if !strings.Contains(prompt, "Interview transcript:\nUser: explain the retry policy") {
		t.Fatalf("unexpected scoring prompt: %q", prompt)
	}

	lines := splitEvaluationLines(`
Summary:
- keep
not-a-bullet
* also keep
`)
	fallback := fallbackEvaluationBullets(lines, 2)
	if len(fallback) != 2 || fallback[0] != "keep" || fallback[1] != "also keep" {
		t.Fatalf("unexpected fallback bullets: %#v", fallback)
	}
}
