package interview

import (
	"fmt"
	"strings"

	"mockinterview/internal/protocol"
)

type NextStepDecision = protocol.NextStepDecision
type DecisionReason = protocol.DecisionReason

func DecideNextStep(
	profile protocol.CandidateProfile,
	state protocol.RunInterviewState,
	analysis AnswerSignalAnalysis,
	cfg protocol.InterviewConfig,
	skill protocol.SkillSpec,
) protocol.NextStepDecision {
	state = EnsureRunInterviewState(&state)
	cfg = cfg.WithDefaults()
	mode := NormalizeInterviewMode(string(cfg.Mode))
	strategy := ResolveStrategyConfig(cfg, skill)
	policy := strategy.Policy

	decision := protocol.NextStepDecision{
		KeepTopic:        true,
		RecommendedFocus: mergeDecisionFocus(TopWeakAreas(profile), policy.RecommendedFocus),
		Reason:           protocol.ReasonConfidenceConfirm,
	}

	missingTradeoffConfidence := weakSignalConfidence(analysis, "missing_tradeoff")
	missingImplementationConfidence := weakSignalConfidence(analysis, "missing_implementation_detail")
	tooGenericConfidence := weakSignalConfidence(analysis, "too_generic")
	timeoutConfidence := maxSignalConfidence(signalConfidenceFromAnalysis(analysis, "missing_timeout_detail", "timeout_control"), signalConfidenceFromState(state, "missing_timeout_detail", "timeout_control"))
	observabilityConfidence := maxSignalConfidence(signalConfidenceFromAnalysis(analysis, "missing_observability_detail", "observability"), signalConfidenceFromState(state, "missing_observability_detail", "observability"))
	tradeoffStrengthConfidence := strongSignalConfidence(analysis, "tradeoff_reasoning")
	implementationStrengthConfidence := strongSignalConfidence(analysis, "implementation_detail")

	reasonScore := 0.0
	assignReason := func(reason protocol.DecisionReason, score float64) {
		if score <= reasonScore {
			return
		}
		decision.Reason = reason
		reasonScore = score
	}

	if missingTradeoffConfidence >= policy.Thresholds.MissingTradeoffReason {
		assignReason(protocol.ReasonMissingTradeoff, missingTradeoffConfidence+0.08)
		decision.EscalatePressure = missingTradeoffConfidence >= policy.Thresholds.MissingTradeoffEscalate
		if missingTradeoffConfidence >= policy.Thresholds.MissingTradeoffAdversarial && shouldAllowAdversarial(state.Phase, policy) {
			decision.TriggerAdversarial = true
		}
	}
	if missingImplementationConfidence >= policy.Thresholds.MissingImplementationReason {
		assignReason(protocol.ReasonLackImplementationDetail, missingImplementationConfidence)
		decision.KeepTopic = true
	}
	if tooGenericConfidence >= policy.Thresholds.TooGenericReason {
		assignReason(protocol.ReasonLackImplementationDetail, tooGenericConfidence-0.04)
		decision.KeepTopic = true
		if tooGenericConfidence >= policy.Thresholds.TooGenericEscalate || state.Round >= policy.EscalatePressureFromRound {
			decision.EscalatePressure = true
		}
		if tooGenericConfidence >= policy.Thresholds.TooGenericAdversarial && shouldAllowAdversarial(state.Phase, policy) {
			decision.TriggerAdversarial = true
		}
	}
	if timeoutConfidence >= policy.Thresholds.TimeoutReason {
		assignReason(protocol.ReasonWeakSignalTimeout, timeoutConfidence)
	}
	if observabilityConfidence >= policy.Thresholds.ObservabilityReason {
		assignReason(protocol.ReasonWeakSignalObservability, observabilityConfidence)
	}
	if state.Phase == protocol.PhaseStress || state.Phase == protocol.PhaseWrapup {
		decision.EscalatePressure = true
		assignReason(protocol.ReasonPressureTest, 1.1)
	}
	if shouldPreferBudgetWrapup(state, cfg) {
		decision.SwitchTopic = false
		decision.KeepTopic = false
		assignReason(protocol.ReasonWrapupDueToBudget, 1.2)
	}

	switch mode {
	case protocol.ModeStress:
		decision.EscalatePressure = true
		if shouldAllowAdversarial(state.Phase, policy) {
			decision.TriggerAdversarial = true
		}
		assignReason(protocol.ReasonPressureTest, reasonScore+0.01)
	case protocol.ModeWeaknessFocused:
		decision.KeepTopic = true
		decision.SwitchTopic = false
		if policy.PreferWeaknessFocus && len(decision.RecommendedFocus) > 0 && decision.Reason == protocol.ReasonConfidenceConfirm {
			assignReason(protocol.ReasonProfileWeaknessFocus, 0.7)
		}
	case protocol.ModeSystemDesign:
		decision.RecommendedFocus = mergeDecisionFocus(decision.RecommendedFocus, []string{"system design", "reliability", "observability"})
		if policy.PreferTopicSwitch &&
			state.Round >= policy.TopicSwitchFromRound &&
			tradeoffStrengthConfidence >= policy.Thresholds.IncreaseDifficultyTradeoff &&
			implementationStrengthConfidence >= policy.Thresholds.IncreaseDifficultyImplementation &&
			tooGenericConfidence < policy.Thresholds.TooGenericReason {
			decision.SwitchTopic = true
			decision.KeepTopic = false
			assignReason(protocol.ReasonTopicSwitch, maxSignalConfidence(tradeoffStrengthConfidence, implementationStrengthConfidence)+0.05)
		}
	case protocol.ModeResumeDeepDive:
		decision.RecommendedFocus = mergeDecisionFocus(decision.RecommendedFocus, []string{"resume evidence", "ownership", "decision tradeoffs"})
		decision.KeepTopic = true
	}

	if len(decision.RecommendedFocus) > 0 && decision.Reason == protocol.ReasonConfidenceConfirm {
		assignReason(protocol.ReasonProfileWeaknessFocus, 0.66)
	}
	if tradeoffStrengthConfidence >= policy.Thresholds.IncreaseDifficultyTradeoff &&
		implementationStrengthConfidence >= policy.Thresholds.IncreaseDifficultyImplementation &&
		tooGenericConfidence < policy.Thresholds.TooGenericReason {
		decision.IncreaseDifficulty = true
	}
	decision.Explanation = buildDecisionExplanation(decision, analysis, state, mode, strategy)

	return decision
}

func shouldPreferBudgetWrapup(state protocol.RunInterviewState, cfg protocol.InterviewConfig) bool {
	turnLimit := DeriveInterviewTurnLimit(cfg.TimeBudget)
	if turnLimit <= 0 {
		return state.Phase == protocol.PhaseWrapup && state.Round >= 4
	}
	return state.Phase == protocol.PhaseWrapup && state.Round >= maxInt(4, turnLimit-1)
}

func shouldAllowAdversarial(phase protocol.InterviewPhase, policy DecisionPolicy) bool {
	order := map[protocol.InterviewPhase]int{
		protocol.PhaseWarmup:      0,
		protocol.PhaseProbe:       1,
		protocol.PhaseAdversarial: 2,
		protocol.PhaseStress:      3,
		protocol.PhaseWrapup:      4,
	}
	return order[phase] >= order[policy.AdversarialFromPhase]
}

func buildDecisionExplanation(
	decision protocol.NextStepDecision,
	analysis AnswerSignalAnalysis,
	state protocol.RunInterviewState,
	mode protocol.InterviewMode,
	strategy StrategyConfig,
) string {
	mode = NormalizeInterviewMode(string(mode))
	focus := strings.Join(decision.RecommendedFocus, ", ")
	strategyNote := strategyExplanationNote(strategy)

	withStrategyNote := func(text string) string {
		if strategyNote == "" {
			return text
		}
		return text + " 当前策略：" + strategyNote + "。"
	}

	switch decision.Reason {
	case protocol.ReasonMissingTradeoff:
		return withStrategyNote("上一轮回答缺少明确取舍比较，下一问会继续留在当前主题，并要求给出更直接的工程取舍。")
	case protocol.ReasonLackImplementationDetail:
		if analysis.TooGeneric {
			return withStrategyNote("上一轮回答偏泛且实现细节不足，下一问会继续深挖同一主题，要求给出具体机制、流程或组件。")
		}
		return withStrategyNote("上一轮回答缺少可落地实现细节，下一问会继续追问具体机制、异常路径和工程边界。")
	case protocol.ReasonWeakSignalTimeout:
		return withStrategyNote("系统检测到超时控制相关弱项，后续会优先确认 timeout、cancel 和失败恢复策略。")
	case protocol.ReasonWeakSignalObservability:
		return withStrategyNote("系统检测到可观测性相关弱项，后续会优先追问日志、指标、追踪和告警设计。")
	case protocol.ReasonPressureTest:
		return withStrategyNote("当前已进入更高压力阶段，系统会缩短回答空间，优先验证你的决策速度和结论表达。")
	case protocol.ReasonTopicSwitch:
		return withStrategyNote("当前主题已经覆盖到 tradeoff 和实现细节，系统会切到下一个主题验证广度。")
	case protocol.ReasonWrapupRequested:
		return withStrategyNote("用户已经明确要求结束面试，系统会停止继续追问并直接进入总结与评分。")
	case protocol.ReasonWrapupDueToBudget:
		return withStrategyNote("时间预算接近收尾，系统会减少分支追问，优先确认最终建议和核心判断。")
	case protocol.ReasonProfileWeaknessFocus:
		if focus == "" {
			return withStrategyNote("系统检测到历史弱项命中，下一问会继续围绕薄弱点校验是否真正补齐。")
		}
		return withStrategyNote(fmt.Sprintf("系统命中了历史弱项，下一问会优先围绕这些重点继续校验：%s。", focus))
	default:
		if state.Phase == protocol.PhaseStress || mode == protocol.ModeStress {
			return withStrategyNote("系统会保持高压力节奏，优先验证在时间限制下的判断与表达质量。")
		}
		if focus != "" {
			return withStrategyNote(fmt.Sprintf("系统会继续围绕当前重点保持追问，优先覆盖这些方向：%s。", focus))
		}
		return withStrategyNote("系统会继续确认当前回答是否具备足够的工程细节、取舍意识和落地可行性。")
	}
}

func strategyExplanationNote(strategy StrategyConfig) string {
	if len(strategy.Notes) > 0 {
		return strategy.Notes[0]
	}
	if strings.TrimSpace(strategy.Description) != "" && strategy.Name != StrategyDefault {
		return strings.TrimSpace(strategy.Description)
	}
	return ""
}

func weakSignalConfidence(analysis AnswerSignalAnalysis, signal string) float64 {
	return signalConfidenceFromMap(analysis.WeakSignalConfidence, analysis.WeakSignals, signal, 0.72)
}

func strongSignalConfidence(analysis AnswerSignalAnalysis, signal string) float64 {
	return signalConfidenceFromMap(analysis.StrongSignalConfidence, analysis.StrongSignals, signal, 0.72)
}

func signalConfidenceFromAnalysis(analysis AnswerSignalAnalysis, weakSignal string, strongSignal string) float64 {
	return maxSignalConfidence(weakSignalConfidence(analysis, weakSignal), strongSignalConfidence(analysis, strongSignal))
}

func signalConfidenceFromState(state protocol.RunInterviewState, weakSignal string, strongSignal string) float64 {
	return maxSignalConfidence(
		signalConfidenceFromMap(nil, state.WeakSignals, weakSignal, 0.64),
		signalConfidenceFromMap(nil, state.StrongSignals, strongSignal, 0.64),
	)
}

func signalConfidenceFromMap(confidence map[string]float64, values []string, signal string, fallback float64) float64 {
	signal = strings.TrimSpace(signal)
	if signal == "" {
		return 0
	}
	if confidence != nil {
		if value, ok := confidence[signal]; ok {
			return clampSignalConfidence(value)
		}
	}
	if containsSignal(values, signal) {
		return clampSignalConfidence(fallback)
	}
	return 0
}

func hasSkillFocus(focusAreas []string, values ...string) bool {
	for _, value := range values {
		if matchesAnyFocus(value, focusAreas) {
			return true
		}
	}
	return false
}

func mergeDecisionFocus(base []string, extra []string) []string {
	out := append([]string(nil), base...)
	seen := make(map[string]struct{}, len(out))
	for _, item := range out {
		seen[strings.ToLower(strings.TrimSpace(item))] = struct{}{}
	}
	for _, item := range extra {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func containsAnySignal(values []string, targets []string) bool {
	for _, target := range targets {
		if containsSignal(values, target) {
			return true
		}
	}
	return false
}
