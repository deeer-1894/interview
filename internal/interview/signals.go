package interview

import (
	"sort"
	"strings"
	"unicode"

	"mockinterview/internal/protocol"
)

type AnswerSignalAnalysis struct {
	WeakSignals               []string
	StrongSignals             []string
	WeakSignalConfidence      map[string]float64
	StrongSignalConfidence    map[string]float64
	TooGeneric                bool
	HasTradeoff               bool
	HasConcreteImplementation bool
}

func AnalyzeAnswerSignals(answer string, skill protocol.SkillSpec) AnswerSignalAnalysis {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		analysis := newAnswerSignalAnalysis()
		analysis.TooGeneric = true
		addWeakSignal(&analysis, "empty_answer", 1)
		addWeakSignal(&analysis, "too_generic", 1)
		finalizeAnswerSignalAnalysis(&analysis)
		return analysis
	}

	lower := strings.ToLower(answer)
	analysis := newAnswerSignalAnalysis()
	features := inspectAnswerFeatures(answer, lower)

	tradeoffScore := scoreTradeoffSignal(lower, features)
	implementationScore := scoreImplementationSignal(lower, features)
	timeoutScore := scoreTimeoutSignal(lower, features)
	observabilityScore := scoreObservabilitySignal(lower, features)
	genericScore := scoreGenericSignal(lower, features, tradeoffScore, implementationScore)
	avoidanceScore := scoreAvoidanceSignal(lower)
	partialScore := scorePartialAnswerSignal(lower, features, implementationScore)
	conceptOnlyScore := scoreConceptOnlySignal(lower, features, implementationScore)
	evidenceGapScore := scoreEvidenceGapSignal(lower, features, tradeoffScore, implementationScore)

	genericScore = maxSignalConfidence(genericScore, conceptOnlyScore-0.06)
	genericScore = maxSignalConfidence(genericScore, evidenceGapScore-0.08)

	analysis.HasTradeoff = signalDetected(tradeoffScore)
	analysis.HasConcreteImplementation = signalDetected(implementationScore)
	analysis.TooGeneric = signalDetected(genericScore)

	if analysis.HasTradeoff {
		addStrongSignal(&analysis, "tradeoff_reasoning", tradeoffScore)
	} else {
		addWeakSignal(&analysis, "missing_tradeoff", inverseSignalConfidence(tradeoffScore))
	}
	if analysis.HasConcreteImplementation {
		addStrongSignal(&analysis, "implementation_detail", implementationScore)
	} else {
		addWeakSignal(&analysis, "missing_implementation_detail", inverseSignalConfidence(implementationScore))
	}
	if signalDetected(timeoutScore) {
		addStrongSignal(&analysis, "timeout_control", timeoutScore)
	} else {
		addWeakSignal(&analysis, "missing_timeout_detail", inverseSignalConfidence(timeoutScore))
	}
	if signalDetected(observabilityScore) {
		addStrongSignal(&analysis, "observability", observabilityScore)
	} else {
		addWeakSignal(&analysis, "missing_observability_detail", inverseSignalConfidence(observabilityScore))
	}

	if analysis.TooGeneric {
		addWeakSignal(&analysis, "too_generic", genericScore)
	}
	if signalDetected(avoidanceScore) {
		addWeakSignal(&analysis, "avoids_core_question", avoidanceScore)
	}
	if signalDetected(partialScore) {
		addWeakSignal(&analysis, "partial_answer", partialScore)
	}
	if signalDetected(conceptOnlyScore) {
		addWeakSignal(&analysis, "concept_without_plan", conceptOnlyScore)
	}
	if signalDetected(evidenceGapScore) {
		addWeakSignal(&analysis, "lacks_example_or_evidence", evidenceGapScore)
	}

	for _, area := range skill.FocusAreas {
		area = strings.TrimSpace(area)
		if area == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(area)) {
			addStrongSignal(&analysis, "focus:"+area, 0.74)
		}
	}

	finalizeAnswerSignalAnalysis(&analysis)
	return analysis
}

func (a AnswerSignalAnalysis) Snapshot() protocol.AnswerSignalSummary {
	return protocol.AnswerSignalSummary{
		WeakSignals:               append([]string(nil), a.WeakSignals...),
		StrongSignals:             append([]string(nil), a.StrongSignals...),
		WeakSignalConfidence:      cloneSignalConfidence(a.WeakSignalConfidence),
		StrongSignalConfidence:    cloneSignalConfidence(a.StrongSignalConfidence),
		TooGeneric:                a.TooGeneric,
		HasTradeoff:               a.HasTradeoff,
		HasConcreteImplementation: a.HasConcreteImplementation,
	}
}

type answerFeatures struct {
	WordCount        int
	CommaCount       int
	HasNumber        bool
	HasCode          bool
	SequenceMarkers  int
	MechanismMarkers int
	ContrastMarkers  int
	ExampleMarkers   int
}

func newAnswerSignalAnalysis() AnswerSignalAnalysis {
	return AnswerSignalAnalysis{
		WeakSignalConfidence:   make(map[string]float64),
		StrongSignalConfidence: make(map[string]float64),
	}
}

func finalizeAnswerSignalAnalysis(analysis *AnswerSignalAnalysis) {
	if analysis == nil {
		return
	}
	analysis.WeakSignals = sortSignalConfidenceKeys(analysis.WeakSignalConfidence)
	analysis.StrongSignals = sortSignalConfidenceKeys(analysis.StrongSignalConfidence)
	if len(analysis.WeakSignalConfidence) == 0 {
		analysis.WeakSignalConfidence = nil
	}
	if len(analysis.StrongSignalConfidence) == 0 {
		analysis.StrongSignalConfidence = nil
	}
}

func inspectAnswerFeatures(answer string, lower string) answerFeatures {
	return answerFeatures{
		WordCount:        estimateAnswerLength(answer),
		CommaCount:       strings.Count(answer, "，") + strings.Count(answer, ","),
		HasNumber:        containsDigit(answer),
		HasCode:          strings.Contains(answer, "`") || strings.Contains(answer, "func ") || strings.Contains(answer, "if ") || strings.Contains(answer, "=>"),
		SequenceMarkers:  countAny(lower, []string{"首先", "第一", "先", "然后", "再", "接着", "最后", "同时"}),
		MechanismMarkers: countAny(lower, []string{"落库", "异步", "补偿", "分层", "封装", "拆成", "流程", "链路", "校验", "后台任务", "消费", "回调", "状态机", "任务表", "幂等", "降级", "兜底", "批处理", "排队"}),
		ContrastMarkers:  countAny(lower, []string{"权衡", "平衡", "折中", "牺牲", "换来", "相比", "优先", "侧重", "代价", "吞吐", "稳定性", "实时性", "复杂度", "一致性"}),
		ExampleMarkers:   countAny(lower, []string{"例如", "比如", "举例", "case", "举个例子"}),
	}
}

func estimateAnswerLength(answer string) int {
	fields := strings.Fields(answer)
	hanCount := 0
	for _, r := range answer {
		if unicode.Is(unicode.Han, r) {
			hanCount++
		}
	}
	if hanCount > 0 {
		return maxInt(len(fields), hanCount/2)
	}
	return len(fields)
}

func scoreTradeoffSignal(lower string, features answerFeatures) float64 {
	score := 0.0
	if containsAny(lower, []string{"tradeoff", "取舍", "成本", "收益", "优先级", "latency", "consistency", "availability"}) {
		score = maxSignalConfidence(score, 0.96)
	}
	if features.ContrastMarkers >= 2 {
		score = maxSignalConfidence(score, 0.78)
	} else if features.ContrastMarkers >= 1 && (features.MechanismMarkers >= 1 || features.HasNumber) {
		score = maxSignalConfidence(score, 0.66)
	}
	return clampSignalConfidence(score)
}

func scoreImplementationSignal(lower string, features answerFeatures) float64 {
	score := 0.0
	if containsAny(lower, []string{"goroutine", "channel", "retry", "timeout", "circuit breaker", "context", "队列", "缓存", "熔断", "重试", "限流", "监控", "trace", "metric", "log", "fallback", "worker", "backoff", "幂等", "限速"}) {
		score = maxSignalConfidence(score, 0.95)
	}
	if features.SequenceMarkers >= 2 && features.MechanismMarkers >= 2 {
		score = maxSignalConfidence(score, 0.82)
	} else if features.MechanismMarkers >= 2 {
		score = maxSignalConfidence(score, 0.7)
	}
	if features.HasCode || features.ExampleMarkers >= 1 {
		score = maxSignalConfidence(score, 0.68)
	}
	return clampSignalConfidence(score)
}

func scoreTimeoutSignal(lower string, features answerFeatures) float64 {
	score := 0.0
	if containsAny(lower, []string{"timeout", "deadline", "context", "超时", "取消", "cancel"}) {
		score = maxSignalConfidence(score, 0.95)
	}
	if containsAny(lower, []string{"快速失败", "兜底", "截止时间", "超时重试", "失败恢复"}) {
		score = maxSignalConfidence(score, 0.68)
	}
	if features.MechanismMarkers >= 2 && containsAny(lower, []string{"失败", "恢复", "回退"}) {
		score = maxSignalConfidence(score, 0.6)
	}
	return clampSignalConfidence(score)
}

func scoreObservabilitySignal(lower string, features answerFeatures) float64 {
	score := 0.0
	if containsAny(lower, []string{"monitor", "metric", "trace", "log", "logging", "alert", "监控", "指标", "日志", "告警", "追踪"}) {
		score = maxSignalConfidence(score, 0.95)
	}
	if containsAny(lower, []string{"request id", "trace id", "采样", "埋点", "error rate", "成功率", "耗时", "告警阈值", "sli", "sla", "dashboard"}) {
		score = maxSignalConfidence(score, 0.76)
	}
	if features.ExampleMarkers >= 1 && containsAny(lower, []string{"日志", "指标", "链路"}) {
		score = maxSignalConfidence(score, 0.64)
	}
	return clampSignalConfidence(score)
}

func scoreGenericSignal(lower string, features answerFeatures, tradeoffScore float64, implementationScore float64) float64 {
	score := 0.0
	if features.WordCount < 20 {
		score += 0.45
	} else if features.WordCount < 40 {
		score += 0.18
	}
	if containsAny(lower, []string{"一般来说", "通常", "我会考虑", "best practice", "看情况", "it depends"}) {
		score += 0.35
	}
	if implementationScore < 0.55 && features.MechanismMarkers == 0 {
		score += 0.18
	}
	if tradeoffScore < 0.55 && features.ContrastMarkers == 0 {
		score += 0.12
	}
	if features.HasCode || features.HasNumber || features.MechanismMarkers >= 2 {
		score -= 0.2
	}
	if features.SequenceMarkers >= 2 {
		score -= 0.08
	}
	return clampSignalConfidence(score)
}

func scoreAvoidanceSignal(lower string) float64 {
	if containsAny(lower, []string{"先不展开", "暂时不细说", "细节略过", "不一定需要", "暂时不考虑", "之后再看", "先不讨论"}) {
		return 0.92
	}
	return 0
}

func scorePartialAnswerSignal(lower string, features answerFeatures, implementationScore float64) float64 {
	score := 0.0
	if features.CommaCount <= 1 && containsAny(lower, []string{"首先", "第一", "one", "first", "先做", "先看"}) && implementationScore < 0.55 {
		score = maxSignalConfidence(score, 0.72)
	}
	if features.WordCount < 25 && features.SequenceMarkers >= 1 && features.MechanismMarkers == 0 {
		score = maxSignalConfidence(score, 0.6)
	}
	return clampSignalConfidence(score)
}

func scoreConceptOnlySignal(lower string, features answerFeatures, implementationScore float64) float64 {
	if implementationScore >= detectedSignalThreshold {
		return 0
	}

	score := 0.0
	conceptMarkers := countAny(lower, []string{
		"思路", "原则", "方向", "关注点", "本质", "核心", "理论上", "一般会",
		"通常会", "需要考虑", "方案上", "先考虑", "核心是", "我会关注",
	})

	if conceptMarkers >= 2 && features.MechanismMarkers == 0 {
		score = maxSignalConfidence(score, 0.82)
	} else if conceptMarkers >= 1 && features.MechanismMarkers == 0 && features.ExampleMarkers == 0 {
		score = maxSignalConfidence(score, 0.68)
	}

	if features.WordCount >= 18 &&
		features.SequenceMarkers == 0 &&
		features.MechanismMarkers == 0 &&
		features.ExampleMarkers == 0 &&
		containsAny(lower, []string{"做好", "保证", "考虑清楚", "先分析", "先判断"}) {
		score = maxSignalConfidence(score, 0.72)
	}

	return clampSignalConfidence(score)
}

func scoreEvidenceGapSignal(lower string, features answerFeatures, tradeoffScore float64, implementationScore float64) float64 {
	if features.ExampleMarkers > 0 || features.HasNumber || features.HasCode {
		return 0
	}
	if tradeoffScore >= detectedSignalThreshold || implementationScore >= detectedSignalThreshold {
		return 0
	}

	score := 0.0
	if features.WordCount >= 12 &&
		features.MechanismMarkers <= 1 &&
		containsAny(lower, []string{"一般来说", "通常", "best practice", "经验上", "我会考虑", "常见做法"}) {
		score = maxSignalConfidence(score, 0.74)
	}
	if features.WordCount >= 18 && features.ExampleMarkers == 0 && features.MechanismMarkers == 0 {
		score = maxSignalConfidence(score, 0.62)
	}

	return clampSignalConfidence(score)
}

func addWeakSignal(analysis *AnswerSignalAnalysis, signal string, confidence float64) {
	if analysis == nil {
		return
	}
	confidence = clampSignalConfidence(confidence)
	if confidence < weakSignalThreshold {
		return
	}
	analysis.WeakSignalConfidence[signal] = maxSignalConfidence(analysis.WeakSignalConfidence[signal], confidence)
}

func addStrongSignal(analysis *AnswerSignalAnalysis, signal string, confidence float64) {
	if analysis == nil {
		return
	}
	confidence = clampSignalConfidence(confidence)
	if confidence < strongSignalThreshold {
		return
	}
	analysis.StrongSignalConfidence[signal] = maxSignalConfidence(analysis.StrongSignalConfidence[signal], confidence)
}

func sortSignalConfidenceKeys(values map[string]float64) []string {
	if len(values) == 0 {
		return nil
	}
	type entry struct {
		Key        string
		Confidence float64
	}
	entries := make([]entry, 0, len(values))
	for key, confidence := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		entries = append(entries, entry{Key: key, Confidence: confidence})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Confidence == entries[j].Confidence {
			return entries[i].Key < entries[j].Key
		}
		return entries[i].Confidence > entries[j].Confidence
	})
	out := make([]string, 0, len(entries))
	for _, item := range entries {
		out = append(out, item.Key)
	}
	return out
}

func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func countAny(text string, keywords []string) int {
	count := 0
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(strings.ToLower(keyword))
		if keyword == "" {
			continue
		}
		if strings.Contains(text, keyword) {
			count++
		}
	}
	return count
}

func cloneSignalConfidence(values map[string]float64) map[string]float64 {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]float64, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func CloneSignalConfidence(values map[string]float64) map[string]float64 {
	return cloneSignalConfidence(values)
}

func containsDigit(value string) bool {
	for _, r := range value {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func signalDetected(confidence float64) bool {
	return clampSignalConfidence(confidence) >= detectedSignalThreshold
}

func inverseSignalConfidence(confidence float64) float64 {
	return clampSignalConfidence(1 - clampSignalConfidence(confidence))
}

func clampSignalConfidence(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}

func maxSignalConfidence(left float64, right float64) float64 {
	if right > left {
		return right
	}
	return left
}

const (
	detectedSignalThreshold = 0.55
	weakSignalThreshold     = 0.6
	strongSignalThreshold   = 0.65
)
