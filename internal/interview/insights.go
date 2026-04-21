package interview

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"mockinterview/internal/protocol"
)

type dimensionRule struct {
	key      string
	label    string
	summary  string
	keywords []string
}

const (
	profileDimensionHalfLifeDays = 45.0
	profileScoreClamp            = 6
	profileTrendLimit            = 8
	profileRadarLimit            = 6
)

var profileDimensionRules = []dimensionRule{
	{key: "system_design", label: "系统设计", summary: "架构、拆分、容量与边界判断", keywords: []string{"架构", "设计", "system design", "tradeoff", "边界", "伸缩", "扩展"}},
	{key: "coding", label: "实现能力", summary: "编码、实现细节、复杂度与工程落地", keywords: []string{"代码", "实现", "coding", "algorithm", "复杂度", "代码质量"}},
	{key: "debugging", label: "问题排查", summary: "定位故障、验证假设、调试与修复路径", keywords: []string{"排查", "调试", "debug", "定位", "故障", "诊断"}},
	{key: "reliability", label: "稳定性", summary: "可靠性、容错、恢复与生产风险意识", keywords: []string{"稳定性", "可靠性", "容灾", "恢复", "fallback", "重试", "高可用"}},
	{key: "observability", label: "可观测性", summary: "日志、指标、追踪与运行时可见性", keywords: []string{"监控", "指标", "日志", "trace", "tracing", "可观测", "告警"}},
	{key: "tool_use", label: "工具与 Agent 编排", summary: "工具调用、状态管理、agent workflow 组织能力", keywords: []string{"tool", "工具", "agent", "workflow", "orchestration", "memory", "checkpoint"}},
	{key: "communication", label: "表达沟通", summary: "结构化表达、澄清与结论能力", keywords: []string{"表达", "沟通", "清晰", "结构化", "总结", "communication"}},
	{key: "product_judgment", label: "业务判断", summary: "业务优先级、用户影响和工程取舍", keywords: []string{"业务", "用户", "优先级", "成本", "收益", "product"}},
}

func BuildPersonaInstruction(persona Persona) string {
	switch persona {
	case PersonaCalm:
		return "Adopt a calm, senior-engineer tone. Ask incisive questions without theatrics. Keep pressure moderate and signal respect for thoughtful answers.\n"
	case PersonaSupportive:
		return "Adopt a supportive but still rigorous coaching tone. Help the candidate recover from partial answers, but keep the technical bar real.\n"
	case PersonaManager:
		return "Adopt the tone of a hiring manager or technical lead. Focus on business impact, execution judgment, ownership, and production tradeoffs.\n"
	default:
		return "Adopt a rigorous, no-nonsense interviewer tone. Keep pressure high, challenge weak claims quickly, and demand concrete engineering detail.\n"
	}
}

func BuildTraceTree(persona protocol.InterviewPersona, existing []protocol.Message, assistant protocol.Message, state *protocol.RunInterviewState, profile *protocol.CandidateProfile) protocol.InterviewTraceTree {
	messages := make([]protocol.Message, 0, len(existing)+1)
	messages = append(messages, existing...)
	if strings.TrimSpace(assistant.Content) != "" {
		messages = append(messages, assistant)
	}
	interviewState := EnsureRunInterviewState(state)

	nodes := make([]protocol.InterviewTraceNode, 0, len(messages)/2+1)
	lastNodeIndex := -1
	pendingAnswer := ""
	previousTopic := ""
	wrapupRequested := false

	for index, message := range messages {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}

		if strings.EqualFold(message.Role, "user") {
			if pendingAnswer == "" {
				pendingAnswer = content
			} else {
				pendingAnswer += "\n" + content
			}
			continue
		}

		if lastNodeIndex >= 0 && pendingAnswer != "" {
			nodes[lastNodeIndex].AnswerSummary = summarizeAnswer(pendingAnswer)
			nodes[lastNodeIndex].Signal = inferSignal(pendingAnswer)
			if IsExplicitWrapupRequest(pendingAnswer) {
				wrapupRequested = true
			}
			pendingAnswer = ""
		}

		if IsWrapupAssistantMessage(content) {
			continue
		}
		if wrapupRequested {
			continue
		}
		if !CountsAsInterviewTurn(content) && !hasFutureUserAnswer(messages, index) {
			continue
		}

		question := extractPrimaryQuestion(content)
		topic := inferTopic(question)
		kind := "opening"
		parentID := ""
		depth := 0
		if lastNodeIndex >= 0 {
			parentID = nodes[lastNodeIndex].ID
			depth = nodes[lastNodeIndex].Depth + 1
			kind = "followup"
			if previousTopic != "" && topic != "" && topic != previousTopic {
				kind = "topic_shift"
			}
		}

		node := protocol.InterviewTraceNode{
			ID:        fmt.Sprintf("q_%d", len(nodes)+1),
			MessageID: message.ID,
			ParentID:  parentID,
			Depth:     depth,
			Kind:      kind,
			Question:  question,
			Topic:     topic,
		}
		if index := len(nodes); index < len(interviewState.History) {
			snapshot := interviewState.History[index]
			node.Round = snapshot.Round
			node.Phase = snapshot.Phase
			node.Difficulty = snapshot.Difficulty
			node.Scenario = snapshot.Scenario
			node.Adversarial = snapshot.Adversarial
			node.Pressure = snapshot.Pressure
			node.Reason = snapshot.Reason
			node.Explanation = snapshot.Explanation
			node.WeakSignals = append([]string(nil), snapshot.WeakSignals...)
			node.StrongSignals = append([]string(nil), snapshot.StrongSignals...)
		}
		if node.Round == 0 {
			node.Round = len(nodes) + 1
		}
		if node.Phase == "" {
			node.Phase = interviewState.Phase
		}
		if node.Difficulty == 0 {
			node.Difficulty = interviewState.Difficulty
		}
		node.FocusHits = traceNodeFocusHits(node, profile)
		node.ProfileHit = len(node.FocusHits) > 0
		nodes = append(nodes, node)
		lastNodeIndex = len(nodes) - 1
		if topic != "" {
			previousTopic = topic
		}
	}

	if lastNodeIndex >= 0 && pendingAnswer != "" {
		nodes[lastNodeIndex].AnswerSummary = summarizeAnswer(pendingAnswer)
		nodes[lastNodeIndex].Signal = inferSignal(pendingAnswer)
	}

	return protocol.InterviewTraceTree{
		RunID:         assistant.RunID,
		Persona:       persona,
		GeneratedAt:   time.Now(),
		QuestionCount: len(nodes),
		Nodes:         nodes,
	}
}

func traceNodeFocusHits(node protocol.InterviewTraceNode, profile *protocol.CandidateProfile) []string {
	if profile == nil {
		return nil
	}
	candidates := make([]string, 0, len(profile.RecurringGaps)+len(profile.RecommendedFocus))
	candidates = append(candidates, profile.RecurringGaps...)
	candidates = append(candidates, profile.RecommendedFocus...)
	if len(candidates) == 0 {
		return nil
	}

	searchValues := []string{
		node.Question,
		node.Topic,
		node.Reason,
		node.Explanation,
	}
	searchValues = append(searchValues, node.WeakSignals...)
	searchValues = append(searchValues, node.StrongSignals...)

	out := make([]string, 0, 3)
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		for _, value := range searchValues {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if matchesAnyFocus(value, []string{candidate}) {
				key := strings.ToLower(candidate)
				if _, ok := seen[key]; ok {
					break
				}
				seen[key] = struct{}{}
				out = append(out, candidate)
				break
			}
		}
		if len(out) >= 3 {
			break
		}
	}
	return out
}

func MergeCandidateProfile(
	profile protocol.CandidateProfile,
	cfg protocol.InterviewConfig,
	skill protocol.SkillSpec,
	scorecard protocol.Scorecard,
	trace protocol.InterviewTraceTree,
) protocol.CandidateProfile {
	now := time.Now()
	if profile.ID == "" {
		profile.ID = "global"
	}
	previousUpdatedAt := profile.UpdatedAt
	profile.InterviewCount++
	profile.LastSkill = strings.TrimSpace(cfg.Skill)
	profile.LastPersona = cfg.Persona
	profile.UpdatedAt = now
	rules := ensureProfileDimensionRules(resolveProfileDimensionRules(skill), profile.Dimensions)

	personaUsage := make(map[protocol.InterviewPersona]int, len(profile.PersonaUsage)+1)
	for _, stat := range profile.PersonaUsage {
		personaUsage[stat.Persona] = stat.Count
	}
	personaUsage[cfg.Persona]++
	profile.PersonaUsage = profile.PersonaUsage[:0]
	for persona, count := range personaUsage {
		profile.PersonaUsage = append(profile.PersonaUsage, protocol.PersonaStat{Persona: persona, Count: count})
	}
	sort.Slice(profile.PersonaUsage, func(i, j int) bool { return profile.PersonaUsage[i].Count > profile.PersonaUsage[j].Count })

	dimensions := map[string]protocol.ProfileDimension{}
	for _, dimension := range profile.Dimensions {
		dimension = migrateLegacyProfileDimension(dimension, previousUpdatedAt)
		dimension = applyProfileTimeDecay(dimension, now)
		dimensions[dimension.Key] = dimension
	}
	applyDimensionEvidence(dimensions, scorecard.Strengths, 1, rules, now)
	applyDimensionEvidence(dimensions, scorecard.Gaps, -1, rules, now)
	applyDimensionEvidence(dimensions, scorecard.Improvements, -1, rules, now)
	applyTraceEvidence(dimensions, trace, rules, now)

	profile.Dimensions = profile.Dimensions[:0]
	for _, rule := range rules {
		dimension := dimensions[rule.key]
		if dimension.Key == "" {
			continue
		}
		dimension = finalizeProfileDimension(dimension, rule, now)
		profile.Dimensions = append(profile.Dimensions, dimension)
	}
	sort.Slice(profile.Dimensions, func(i, j int) bool {
		left := profile.Dimensions[i]
		right := profile.Dimensions[j]
		if left.NormalizedScore != right.NormalizedScore {
			return left.NormalizedScore > right.NormalizedScore
		}
		if left.EvidenceCount != right.EvidenceCount {
			return left.EvidenceCount > right.EvidenceCount
		}
		return left.Label < right.Label
	})
	profile.Radar = buildProfileRadar(profile.Dimensions)
	profile.GrowthCurves = buildProfileGrowthCurves(profile.Dimensions)

	profile.StableStrengths = topBullets(uniqueStrings(append(append([]string(nil), profile.StableStrengths...), scorecard.Strengths...)), 5)
	profile.RecurringGaps = topBullets(uniqueStrings(append(append([]string(nil), profile.RecurringGaps...), scorecard.Gaps...)), 5)
	profile.RecommendedFocus = topBullets(uniqueStrings(append(append([]string(nil), profile.RecommendedFocus...), scorecard.Improvements...)), 5)

	change := fmt.Sprintf(
		"%s 模式下完成一次 %s 面试，追问 %d 轮。",
		personaLabel(cfg.Persona),
		firstNonEmptyInsightValue(strings.TrimSpace(cfg.Skill), strings.TrimSpace(cfg.Focus), "generalist"),
		trace.QuestionCount,
	)
	profile.RecentChanges = append([]string{change}, profile.RecentChanges...)
	if len(profile.RecentChanges) > 6 {
		profile.RecentChanges = profile.RecentChanges[:6]
	}

	return profile
}

func BuildReviewSummary(run protocol.Run, cfg protocol.InterviewConfig, trace protocol.InterviewTraceTree, scorecard *protocol.Scorecard, profile *protocol.CandidateProfile) protocol.ReviewSummary {
	summary := protocol.ReviewSummary{
		Mode:    NormalizeInterviewMode(string(cfg.Mode)),
		Persona: cfg.Persona,
	}
	if run.InterviewState != nil {
		state := EnsureRunInterviewState(run.InterviewState)
		summary.CurrentPhase = state.Phase
		if state.LastDecision != nil {
			summary.DecisionReason = state.LastDecision.Reason
			summary.DecisionExplanation = strings.TrimSpace(state.LastDecision.Explanation)
			summary.RecommendedFocus = append([]string(nil), state.LastDecision.RecommendedFocus...)
		}
	}
	for _, node := range trace.Nodes {
		if summary.AdversarialRound == 0 && node.Adversarial && node.Round > 0 {
			summary.AdversarialRound = node.Round
		}
		if summary.PressureRound == 0 && node.Pressure && node.Round > 0 {
			summary.PressureRound = node.Round
		}
		if summary.WrapupRound == 0 && node.Phase == protocol.PhaseWrapup && node.Round > 0 {
			summary.WrapupRound = node.Round
		}
	}
	if summary.WrapupRound == 0 && summary.CurrentPhase == protocol.PhaseWrapup {
		if run.InterviewState != nil {
			state := EnsureRunInterviewState(run.InterviewState)
			if state.Round > 0 {
				summary.WrapupRound = state.Round
			}
		}
		if summary.WrapupRound == 0 {
			summary.WrapupRound = trace.QuestionCount
		}
	}
	summary.MostCommonWeakSignal = mostCommonWeakSignal(trace)
	summary.HistoricalWeaknessesHit = historicalWeaknessHits(trace)
	if scorecard != nil {
		summary.NewWeaknesses = topBullets(uniqueStrings(scorecard.Gaps), 3)
		summary.ResolvedWeaknesses = topBullets(uniqueStrings(scorecard.Strengths), 3)
	}
	if profile != nil && len(summary.NewWeaknesses) == 0 {
		summary.NewWeaknesses = topBullets(uniqueStrings(profile.RecurringGaps), 3)
	}
	if profile != nil && len(summary.ResolvedWeaknesses) == 0 {
		summary.ResolvedWeaknesses = topBullets(uniqueStrings(profile.StableStrengths), 3)
	}
	return summary
}

func historicalWeaknessHits(trace protocol.InterviewTraceTree) []string {
	values := make([]string, 0, len(trace.Nodes))
	for _, node := range trace.Nodes {
		values = append(values, node.FocusHits...)
	}
	return topBullets(uniqueStrings(values), 4)
}

func mostCommonWeakSignal(trace protocol.InterviewTraceTree) string {
	counts := map[string]int{}
	bestKey := ""
	bestCount := 0
	for _, node := range trace.Nodes {
		for _, signal := range node.WeakSignals {
			signal = strings.TrimSpace(signal)
			if signal == "" {
				continue
			}
			counts[signal]++
			if counts[signal] > bestCount {
				bestKey = signal
				bestCount = counts[signal]
			}
		}
	}
	return bestKey
}

func applyDimensionEvidence(
	dimensions map[string]protocol.ProfileDimension,
	items []string,
	direction int,
	rules []dimensionRule,
	now time.Time,
) {
	for _, item := range items {
		text := strings.TrimSpace(item)
		if text == "" {
			continue
		}
		for _, rule := range rules {
			if !matchesKeywords(text, rule.keywords) {
				continue
			}
			dimensions[rule.key] = applyDimensionDelta(dimensions[rule.key], rule, direction, now)
		}
	}
}

func applyTraceEvidence(
	dimensions map[string]protocol.ProfileDimension,
	trace protocol.InterviewTraceTree,
	rules []dimensionRule,
	now time.Time,
) {
	for _, node := range trace.Nodes {
		for _, rule := range rules {
			if !matchesKeywords(node.Question, rule.keywords) {
				continue
			}
			delta := 0
			if node.Signal == "strong" {
				delta = 1
			}
			if node.Signal == "weak" {
				delta = -1
			}
			dimensions[rule.key] = applyDimensionDelta(dimensions[rule.key], rule, delta, now)
		}
	}
}

func applyDimensionDelta(
	dimension protocol.ProfileDimension,
	rule dimensionRule,
	delta int,
	now time.Time,
) protocol.ProfileDimension {
	dimension.Key = rule.key
	dimension.Label = firstNonEmptyInsightValue(dimension.Label, rule.label)
	dimension.Summary = firstNonEmptyInsightValue(dimension.Summary, rule.summary)
	dimension.Score += delta
	dimension.RecentDelta += delta
	dimension.EvidenceCount++
	dimension.LastUpdatedAt = timestampPointer(now)
	return dimension
}

func resolveProfileDimensionRules(skill protocol.SkillSpec) []dimensionRule {
	rules := append([]dimensionRule(nil), profileDimensionRules...)
	for _, focus := range normalizeBulletList(skill.FocusAreas) {
		if profileRuleExists(rules, focus) {
			continue
		}
		key := profileDimensionKey(focus)
		if key == "" {
			continue
		}
		rules = append(rules, dimensionRule{
			key:      key,
			label:    focus,
			summary:  fmt.Sprintf("围绕 %s 的专项能力画像。", focus),
			keywords: []string{focus},
		})
	}
	return rules
}

func ensureProfileDimensionRules(rules []dimensionRule, existing []protocol.ProfileDimension) []dimensionRule {
	for _, dimension := range existing {
		if strings.TrimSpace(dimension.Key) == "" || profileRuleKeyExists(rules, dimension.Key) {
			continue
		}
		keywords := normalizeBulletList([]string{dimension.Key, dimension.Label})
		if len(keywords) == 0 {
			keywords = []string{dimension.Key}
		}
		rules = append(rules, dimensionRule{
			key:      dimension.Key,
			label:    firstNonEmptyInsightValue(dimension.Label, dimension.Key),
			summary:  dimension.Summary,
			keywords: keywords,
		})
	}
	return rules
}

func profileRuleExists(rules []dimensionRule, focus string) bool {
	for _, rule := range rules {
		if matchesAnyFocus(focus, []string{rule.key, rule.label}) {
			return true
		}
		if matchesAnyFocus(focus, rule.keywords) {
			return true
		}
	}
	return false
}

func profileRuleKeyExists(rules []dimensionRule, key string) bool {
	key = strings.TrimSpace(key)
	for _, rule := range rules {
		if strings.EqualFold(strings.TrimSpace(rule.key), key) {
			return true
		}
	}
	return false
}

func profileDimensionKey(value string) string {
	normalized := normalizeFocusText(value)
	if normalized == "" {
		return ""
	}
	return strings.ReplaceAll(normalized, " ", "_")
}

func migrateLegacyProfileDimension(dimension protocol.ProfileDimension, fallbackUpdatedAt time.Time) protocol.ProfileDimension {
	if dimension.LastUpdatedAt == nil && !fallbackUpdatedAt.IsZero() {
		dimension.LastUpdatedAt = timestampPointer(fallbackUpdatedAt)
	}
	if len(dimension.Trend) == 0 && !fallbackUpdatedAt.IsZero() && (dimension.Score != 0 || dimension.EvidenceCount > 0) {
		dimension.Trend = append(dimension.Trend, protocol.ProfileTrendPoint{
			Timestamp:       fallbackUpdatedAt,
			Score:           dimension.Score,
			NormalizedScore: normalizeProfileScore(dimension.Score),
		})
	}
	return dimension
}

func applyProfileTimeDecay(dimension protocol.ProfileDimension, now time.Time) protocol.ProfileDimension {
	if dimension.Key == "" {
		return dimension
	}
	dimension.RecentDelta = 0
	if dimension.LastUpdatedAt == nil || dimension.LastUpdatedAt.IsZero() || !dimension.LastUpdatedAt.Before(now) {
		dimension.NormalizedScore = normalizeProfileScore(dimension.Score)
		return dimension
	}
	elapsedDays := now.Sub(*dimension.LastUpdatedAt).Hours() / 24
	if elapsedDays <= 0 {
		dimension.NormalizedScore = normalizeProfileScore(dimension.Score)
		return dimension
	}
	decayFactor := math.Pow(0.5, elapsedDays/profileDimensionHalfLifeDays)
	dimension.Score = int(math.Round(float64(dimension.Score) * decayFactor))
	dimension.NormalizedScore = normalizeProfileScore(dimension.Score)
	return dimension
}

func finalizeProfileDimension(
	dimension protocol.ProfileDimension,
	rule dimensionRule,
	now time.Time,
) protocol.ProfileDimension {
	dimension.Key = rule.key
	dimension.Label = firstNonEmptyInsightValue(dimension.Label, rule.label)
	dimension.Summary = firstNonEmptyInsightValue(dimension.Summary, rule.summary)
	dimension.NormalizedScore = normalizeProfileScore(dimension.Score)
	if dimension.LastUpdatedAt == nil || dimension.LastUpdatedAt.IsZero() {
		dimension.LastUpdatedAt = timestampPointer(now)
	}
	dimension.Trend = appendProfileTrend(dimension.Trend, protocol.ProfileTrendPoint{
		Timestamp:       now,
		Score:           dimension.Score,
		NormalizedScore: dimension.NormalizedScore,
	})
	return dimension
}

func appendProfileTrend(trend []protocol.ProfileTrendPoint, point protocol.ProfileTrendPoint) []protocol.ProfileTrendPoint {
	if point.Timestamp.IsZero() {
		return trend
	}
	if len(trend) > 0 {
		last := trend[len(trend)-1]
		if point.Timestamp.Sub(last.Timestamp) < time.Minute {
			trend[len(trend)-1] = point
		} else {
			trend = append(trend, point)
		}
	} else {
		trend = append(trend, point)
	}
	if len(trend) > profileTrendLimit {
		return append([]protocol.ProfileTrendPoint(nil), trend[len(trend)-profileTrendLimit:]...)
	}
	return trend
}

func buildProfileRadar(dimensions []protocol.ProfileDimension) []protocol.ProfileRadarPoint {
	if len(dimensions) == 0 {
		return nil
	}
	limit := minInt(len(dimensions), profileRadarLimit)
	out := make([]protocol.ProfileRadarPoint, 0, limit)
	for _, dimension := range dimensions[:limit] {
		out = append(out, protocol.ProfileRadarPoint{
			Key:             dimension.Key,
			Label:           dimension.Label,
			NormalizedScore: dimension.NormalizedScore,
		})
	}
	return out
}

func buildProfileGrowthCurves(dimensions []protocol.ProfileDimension) []protocol.ProfileGrowthCurve {
	if len(dimensions) == 0 {
		return nil
	}
	out := make([]protocol.ProfileGrowthCurve, 0, len(dimensions))
	for _, dimension := range dimensions {
		if len(dimension.Trend) == 0 {
			continue
		}
		out = append(out, protocol.ProfileGrowthCurve{
			Key:    dimension.Key,
			Label:  dimension.Label,
			Points: append([]protocol.ProfileTrendPoint(nil), dimension.Trend...),
		})
	}
	return out
}

func normalizeProfileScore(score int) int {
	clamped := score
	if clamped > profileScoreClamp {
		clamped = profileScoreClamp
	}
	if clamped < -profileScoreClamp {
		clamped = -profileScoreClamp
	}
	return int(math.Round((float64(clamped+profileScoreClamp) / float64(profileScoreClamp*2)) * 100))
}

func timestampPointer(value time.Time) *time.Time {
	copyValue := value
	return &copyValue
}

func matchesKeywords(text string, keywords []string) bool {
	lower := strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(lower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func extractPrimaryQuestion(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	parts := strings.Split(content, "\n")
	lastQuestion := ""
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "?") || strings.Contains(part, "？") {
			lastQuestion = part
		}
	}
	if lastQuestion != "" {
		return lastQuestion
	}
	if len(parts) > 0 {
		return strings.TrimSpace(parts[len(parts)-1])
	}
	return content
}

func summarizeAnswer(answer string) string {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return ""
	}
	runes := []rune(answer)
	if len(runes) <= 96 {
		return answer
	}
	return string(runes[:96]) + "..."
}

func inferSignal(answer string) string {
	answer = strings.ToLower(strings.TrimSpace(answer))
	switch {
	case answer == "":
		return ""
	case len(answer) < 24 || strings.Contains(answer, "不太确定") || strings.Contains(answer, "不清楚") || strings.Contains(answer, "不知道"):
		return "weak"
	case strings.Contains(answer, "例如") || strings.Contains(answer, "比如") || strings.Contains(answer, "具体") || strings.Contains(answer, "生产") || strings.Contains(answer, "线上"):
		return "strong"
	default:
		return "neutral"
	}
}

func inferTopic(question string) string {
	for _, rule := range profileDimensionRules {
		if matchesKeywords(question, rule.keywords) {
			return rule.label
		}
	}
	return "通用"
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func topBullets(items []string, limit int) []string {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func personaLabel(persona protocol.InterviewPersona) string {
	switch persona {
	case protocol.PersonaCalm:
		return "冷静专业型"
	case protocol.PersonaSupportive:
		return "启发引导型"
	case protocol.PersonaManager:
		return "业务负责人型"
	default:
		return "严格拷打型"
	}
}

func firstNonEmptyInsightValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
