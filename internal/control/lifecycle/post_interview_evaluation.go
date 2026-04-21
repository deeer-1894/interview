package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	runtimepkg "mockinterview/internal/control/runtime"
	interviewexec "mockinterview/internal/executors/interview"
	domain "mockinterview/internal/interview"
	"mockinterview/internal/protocol"
)

func CompletePostInterviewEvaluation(
	ctx context.Context,
	recorder runtimepkg.EventRecorder,
	runCtx *runtimepkg.RunContext,
	run protocol.Run,
	messages []protocol.Message,
	events []protocol.Event,
	generateScorecard func(context.Context, string, protocol.InterviewConfig, protocol.ModelConfig, protocol.SkillSpec, protocol.Rubric) (protocol.Scorecard, error),
	completedAt time.Time,
) error {
	if runCtx == nil {
		return nil
	}
	runCtx.Run = run

	transcript := BuildEvaluationTranscript(messages, protocol.Message{})
	traceTree := ResolveEvaluationTrace(runCtx.Task.Config, run, messages)
	assistantMessage := LatestAssistantMessage(messages)
	if assistantMessage.CreatedAt.IsZero() {
		assistantMessage.CreatedAt = completedAt
	}

	reviewSnapshot := BuildInterviewerReviewSnapshot(
		runCtx,
		transcript,
		messages,
		assistantMessage,
		traceTree,
		completedAt,
	)

	evaluatorStartedAt := time.Now()
	evaluatorShared := interviewexec.BuildSharedSessionContext(
		run.ID,
		"",
		protocol.RunPhaseEvaluating,
		domain.EnsureRunInterviewState(run.InterviewState),
		runCtx.Resolved.Interview.Skill,
		runCtx.Resolved.Interview.Rubric,
		transcript,
	)

	if generateScorecard == nil {
		generateScorecard = interviewexec.GenerateScorecard
	}

	scorecard, reusedExisting := resolvePostInterviewScorecard(
		assistantMessage.Content,
		runCtx.Task.Config,
		runCtx.Resolved.Interview.Rubric,
	)
	scoreErr := error(nil)
	if !reusedExisting {
		scoreCtx, cancel := context.WithTimeout(ctx, postInterviewEvaluationTimeout())
		scorecard, scoreErr = generateScorecard(
			scoreCtx,
			transcript,
			runCtx.Task.Config,
			runCtx.Task.ModelConfig,
			runCtx.Resolved.Interview.Skill,
			runCtx.Resolved.Interview.Rubric,
		)
		cancel()
		if scoreErr != nil && isEvaluationTimeoutError(scoreErr) {
			scorecard = buildFallbackScorecard(traceTree, runCtx.Task.Config, runCtx.Resolved.Interview.Rubric)
			scoreErr = nil
		}
	}
	if scoreErr != nil {
		if errors.Is(scoreErr, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return context.Canceled
		}
		reviewSnapshot.Agents = append(reviewSnapshot.Agents, interviewexec.BuildAgentExecution(
			protocol.AgentRoleEvaluator,
			protocol.RunFailed,
			evaluatorShared,
			evaluatorStartedAt,
			time.Now(),
			transcript,
			"",
			scoreErr,
		))
	} else if scorecard.Title != "" || scorecard.OverallScore > 0 || len(scorecard.DimensionScores) > 0 || len(scorecard.Gaps) > 0 || len(scorecard.Strengths) > 0 {
		scorecard = normalizePostInterviewScorecard(scorecard)
		reviewSnapshot.Scorecard = &scorecard
		evaluatorOutput := firstNonEmptyString(strings.TrimSpace(scorecard.Summary), strings.TrimSpace(scorecard.Title))
		if reusedExisting {
			evaluatorOutput = firstNonEmptyString(evaluatorOutput, "reused final assistant evaluation output")
		}
		reviewSnapshot.Agents = append(reviewSnapshot.Agents, interviewexec.BuildAgentExecution(
			protocol.AgentRoleEvaluator,
			protocol.RunCompleted,
			evaluatorShared,
			evaluatorStartedAt,
			time.Now(),
			transcript,
			evaluatorOutput,
			nil,
		))
		if err := recordScoreGenerated(ctx, recorder, run, scorecard, time.Now()); err != nil {
			return err
		}
		if scopedProfile := buildScopedCandidateProfile(runCtx, traceTree, scorecard); scopedProfile != nil {
			reviewSnapshot.Profile = scopedProfile
		}
		if _, err := mergeCandidateProfile(ctx, recorder, runCtx, traceTree, scorecard); err != nil {
			return err
		}
	}

	if traceTree != nil {
		summary := domain.BuildReviewSummary(run, runCtx.Task.Config, *traceTree, reviewSnapshot.Scorecard, reviewSnapshot.Profile)
		reviewSnapshot.Summary = &summary
	}
	finalizedAt := completedAt

	wrapupSource := firstNonEmptyString(strings.TrimSpace(assistantMessage.Content), strings.TrimSpace(run.Output))
	if finalOutput := buildPostInterviewWrapup(wrapupSource, runCtx.Task.Config, reviewSnapshot); strings.TrimSpace(finalOutput) != "" {
		finalizedAt = nextWrapupTimestamp(completedAt, LatestAssistantMessage(messages).CreatedAt)
		runCtx.Run.Output = finalOutput
		if err := recordPostInterviewWrapup(ctx, recorder, runCtx.Run, finalOutput, finalizedAt); err != nil {
			return err
		}
	}

	reviewSnapshot.GeneratedAt = finalizedAt
	return FinalizeRunCompleted(ctx, recorder, runCtx, finalizedAt, &reviewSnapshot)
}

func ResolveEvaluationTrace(cfg protocol.InterviewConfig, run protocol.Run, messages []protocol.Message) *protocol.InterviewTraceTree {
	if run.TraceTree != nil {
		trace := *run.TraceTree
		return &trace
	}

	assistant := LatestAssistantMessage(messages)
	if assistant.ID == "" {
		return nil
	}

	assistantIndex := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].ID == assistant.ID {
			assistantIndex = i
			break
		}
	}
	if assistantIndex < 0 {
		return nil
	}

	trace := domain.BuildTraceTree(cfg.Persona, messages[:assistantIndex], assistant, run.InterviewState, nil)
	return &trace
}

func LatestAssistantMessage(messages []protocol.Message) protocol.Message {
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.EqualFold(messages[i].Role, "assistant") && strings.TrimSpace(messages[i].Content) != "" {
			return messages[i]
		}
	}
	return protocol.Message{}
}

func recordScoreGenerated(
	ctx context.Context,
	recorder runtimepkg.EventRecorder,
	run protocol.Run,
	scorecard protocol.Scorecard,
	timestamp time.Time,
) error {
	if recorder == nil {
		return nil
	}
	if err := recorder.RecordEvent(ctx, protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: run.ConversationID,
		TaskID:         run.TaskID,
		RunID:          run.ID,
		Type:           protocol.EventScoreGenerated,
		Timestamp:      timestamp,
		Payload:        scorecard,
	}); err != nil {
		return fmt.Errorf("record score event: %w", err)
	}
	return nil
}

func mergeCandidateProfile(
	ctx context.Context,
	recorder runtimepkg.EventRecorder,
	runCtx *runtimepkg.RunContext,
	traceTree *protocol.InterviewTraceTree,
	scorecard protocol.Scorecard,
) (*protocol.CandidateProfile, error) {
	if recorder == nil || runCtx == nil || traceTree == nil {
		return nil, nil
	}
	profile, err := recorder.GetCandidateProfile(ctx)
	if err != nil {
		return nil, nil
	}
	profile = domain.MergeCandidateProfile(profile, runCtx.Task.Config, runCtx.Resolved.Interview.Skill, scorecard, *traceTree)
	savedProfile, err := recorder.SaveCandidateProfile(ctx, profile)
	if err != nil {
		return nil, nil
	}
	if err := recorder.RecordEvent(ctx, protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: runCtx.Run.ConversationID,
		TaskID:         runCtx.Run.TaskID,
		RunID:          runCtx.Run.ID,
		Type:           protocol.EventProfileUpdated,
		Timestamp:      time.Now(),
		Payload:        savedProfile,
	}); err != nil {
		return nil, fmt.Errorf("record profile event: %w", err)
	}
	return &savedProfile, nil
}

func buildScopedCandidateProfile(
	runCtx *runtimepkg.RunContext,
	traceTree *protocol.InterviewTraceTree,
	scorecard protocol.Scorecard,
) *protocol.CandidateProfile {
	if runCtx == nil || traceTree == nil {
		return nil
	}
	profile := domain.MergeCandidateProfile(
		protocol.CandidateProfile{ID: "run:" + runCtx.Run.ID},
		runCtx.Task.Config,
		runCtx.Resolved.Interview.Skill,
		scorecard,
		*traceTree,
	)
	return &profile
}

func postInterviewEvaluationTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("EVALUATION_TIMEOUT_SECONDS"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("MODEL_TIMEOUT_SECONDS"))
	}
	if raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			timeout := time.Duration(seconds) * time.Second
			if timeout > 45*time.Second {
				return 45 * time.Second
			}
			return timeout
		}
	}
	return 45 * time.Second
}

func isEvaluationTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "deadline exceeded")
}

func buildFallbackScorecard(
	traceTree *protocol.InterviewTraceTree,
	cfg protocol.InterviewConfig,
	rubric protocol.Rubric,
) protocol.Scorecard {
	card := protocol.Scorecard{
		Title:           firstNonEmptyString(strings.TrimSpace(rubric.Title), "Interview Scorecard"),
		OverallMaxScore: 100,
		Anchors:         append([]string(nil), rubric.Anchors...),
	}
	if traceTree == nil {
		card.OverallScore = 60
		card.Gaps = []string{"评分模型超时，未能生成更细的结构化评分。"}
		card.Improvements = []string{"缩短本轮面试上下文或降低评分模型延迟。"}
		card.Summary = buildFallbackScorecardSummary(card)
		return card
	}

	weakCounts := map[string]int{}
	strongCounts := map[string]int{}
	for _, node := range traceTree.Nodes {
		for _, signal := range node.WeakSignals {
			weakCounts[strings.ToLower(strings.TrimSpace(signal))]++
		}
		for _, signal := range node.StrongSignals {
			strongCounts[strings.ToLower(strings.TrimSpace(signal))]++
		}
	}

	systemDesignScore := clampDimensionScore(8 - weakCounts["missing_tradeoff"] - weakCounts["too_generic"])
	implementationScore := clampDimensionScore(8 - weakCounts["missing_implementation_detail"])
	reliabilityScore := clampDimensionScore(7 + strongCounts["timeout_control"] - weakCounts["missing_timeout_detail"])
	observabilityScore := clampDimensionScore(7 + strongCounts["observability"] - weakCounts["missing_observability_detail"])
	if cfg.Mode == protocol.ModeSystemDesign && systemDesignScore < 7 {
		systemDesignScore++
	}

	card.DimensionScores = []protocol.DimensionScore{
		{Name: "System Design", Score: systemDesignScore, MaxScore: 10, Rationale: "基于本场追问里的架构拆分、tradeoff 与回答深度信号生成。"},
		{Name: "Implementation Detail", Score: implementationScore, MaxScore: 10, Rationale: "基于实现细节、代码级说明和工程落地信号生成。"},
		{Name: "Reliability", Score: reliabilityScore, MaxScore: 10, Rationale: "基于 timeout、cancel、失败恢复与稳定性信号生成。"},
		{Name: "Observability", Score: observabilityScore, MaxScore: 10, Rationale: "基于日志、指标、trace 与运行时可见性信号生成。"},
	}

	if strongCounts["timeout_control"] > 0 {
		card.Strengths = append(card.Strengths, "对 timeout / cancel / failure isolation 的核心方向有基本把握。")
	}
	if strongCounts["observability"] > 0 {
		card.Strengths = append(card.Strengths, "能够主动提到指标、日志或 trace 这类可观测性面。")
	}
	if len(card.Strengths) == 0 && traceTree.QuestionCount > 0 {
		card.Strengths = append(card.Strengths, "能够跟随追问持续围绕同一系统主题作答。")
	}

	appendGap := func(condition bool, gap string) {
		if condition {
			card.Gaps = append(card.Gaps, gap)
		}
	}
	appendGap(weakCounts["missing_tradeoff"] > 0, "tradeoff 表达不够明确，缺少方案比较与取舍理由。")
	appendGap(weakCounts["missing_implementation_detail"] > 0, "实现细节不足，关键组件缺少足够具体的 Go 级设计说明。")
	appendGap(weakCounts["missing_timeout_detail"] > 0, "timeout / cancel / retry 细节不够完整。")
	appendGap(weakCounts["missing_observability_detail"] > 0, "可观测性方案不够具体，指标与 trace 设计仍偏抽象。")
	appendGap(weakCounts["too_generic"] > 0 && len(card.Gaps) < 3, "部分回答仍然偏抽象，缺少可落地的工程例子。")
	if len(card.Gaps) == 0 {
		card.Gaps = append(card.Gaps, "评分模型超时，本次结构化评分由本地规则回退生成，细节粒度有限。")
	}

	card.Improvements = append(card.Improvements,
		"后续回答优先补齐关键组件的 Go 代码骨架、并发模型和错误传播路径。",
		"在每个设计点明确说明 tradeoff、超时预算和 observability 埋点策略。",
	)

	total := 0
	for _, dimension := range card.DimensionScores {
		total += dimension.Score
	}
	if len(card.DimensionScores) > 0 {
		card.OverallScore = total * 10 / len(card.DimensionScores)
	}
	card = normalizePostInterviewScorecard(card)
	return card
}

func clampDimensionScore(score int) int {
	if score < 1 {
		return 1
	}
	if score > 10 {
		return 10
	}
	return score
}

func normalizePostInterviewScorecard(card protocol.Scorecard) protocol.Scorecard {
	card.Strengths = filterMeaningfulEvaluationItems(card.Strengths)
	card.Gaps = filterMeaningfulEvaluationItems(card.Gaps)
	card.Improvements = filterMeaningfulEvaluationItems(card.Improvements)
	card.StudyPlan = filterMeaningfulEvaluationItems(card.StudyPlan)

	if card.OverallScore > 0 && card.OverallMaxScore <= 0 {
		card.OverallMaxScore = 100
	}
	if strings.TrimSpace(card.Summary) == "" {
		card.Summary = buildFallbackScorecardSummary(card)
	}
	return card
}

func filterMeaningfulEvaluationItems(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || !containsMeaningfulWrapupText(value) || isWrapupNoiseItem(value) {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func containsMeaningfulWrapupText(value string) bool {
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r > 127 {
			return true
		}
	}
	return false
}

func isWrapupNoiseItem(value string) bool {
	normalized := strings.ToLower(strings.Trim(strings.TrimSpace(value), "：:"))
	switch normalized {
	case "面试策略总结", "策略建议", "追问树分析", "追问树总结", "画像分析", "候选人画像", "阶段推进评估", "阶段过程总结", "建议后续阶段":
		return true
	}
	noiseFragments := []string{
		"本次面试采用了",
		"候选人展现出",
		"阶段运行良好",
		"建议后续阶段",
		"整体表现优秀",
	}
	for _, fragment := range noiseFragments {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
}

func buildFallbackScorecardSummary(card protocol.Scorecard) string {
	switch {
	case len(card.Strengths) > 0 && len(card.Gaps) > 0:
		return fmt.Sprintf("优势是%s，当前最需要补齐的是%s。", card.Strengths[0], card.Gaps[0])
	case len(card.Strengths) > 0:
		return fmt.Sprintf("本场表现较稳的部分是%s。", card.Strengths[0])
	case len(card.Gaps) > 0:
		return fmt.Sprintf("本场最明显的短板是%s。", card.Gaps[0])
	case len(card.Improvements) > 0:
		return fmt.Sprintf("建议下一步优先%s。", card.Improvements[0])
	default:
		return ""
	}
}

func resolvePostInterviewScorecard(
	assistantOutput string,
	cfg protocol.InterviewConfig,
	rubric protocol.Rubric,
) (protocol.Scorecard, bool) {
	card := interviewexec.BuildEvaluationScorecardFromOutput(assistantOutput, rubric, cfg.OutputStyle)
	if !scorecardLooksReusable(card) {
		return protocol.Scorecard{}, false
	}
	return card, true
}

func scorecardLooksReusable(card protocol.Scorecard) bool {
	if len(card.DimensionScores) > 0 {
		return true
	}
	sectionCount := 0
	if len(card.Strengths) > 0 {
		sectionCount++
	}
	if len(card.Gaps) > 0 {
		sectionCount++
	}
	if len(card.Improvements) > 0 {
		sectionCount++
	}
	if len(card.StudyPlan) > 0 {
		sectionCount++
	}
	if card.OverallScore > 0 && sectionCount >= 2 && (len(card.Strengths) > 0 || len(card.Gaps) > 0) {
		return true
	}
	if strings.TrimSpace(card.Summary) != "" && sectionCount >= 2 && (len(card.Strengths) > 0 || len(card.Gaps) > 0) {
		return true
	}
	return false
}

func buildPostInterviewWrapup(previousOutput string, cfg protocol.InterviewConfig, snapshot protocol.ReviewSnapshot) string {
	if shouldPreserveStructuredEvaluationOutput(previousOutput) {
		return strings.TrimSpace(previousOutput)
	}

	var sections []string

	summary := ""
	if snapshot.Scorecard != nil {
		summary = strings.TrimSpace(snapshot.Scorecard.Summary)
	}
	if summary == "" && snapshot.Summary != nil {
		summary = strings.TrimSpace(snapshot.Summary.DecisionExplanation)
	}
	if summary == "" && !looksLikeQuestion(previousOutput) {
		summary = strings.TrimSpace(previousOutput)
	}
	if summary != "" {
		sections = append(sections, "面试到这里结束，下面是本场总结。", "总评："+summary)
	} else {
		sections = append(sections, "面试到这里结束，下面是本场总结。")
	}

	if snapshot.Scorecard != nil {
		if overall := formatWrapupOverallScore(*snapshot.Scorecard); overall != "" {
			sections = append(sections, overall)
		}
		if strengths := formatWrapupBullets("亮点", snapshot.Scorecard.Strengths, 2); strengths != "" {
			sections = append(sections, strengths)
		}
		if gaps := formatWrapupBullets("待改进", snapshot.Scorecard.Gaps, 2); gaps != "" {
			sections = append(sections, gaps)
		}
		nextSteps := snapshot.Scorecard.Improvements
		if cfg.OutputStyle == protocol.OutputInterviewPlusStudy && len(snapshot.Scorecard.StudyPlan) > 0 {
			nextSteps = snapshot.Scorecard.StudyPlan
		}
		if plan := formatWrapupBullets("下一步建议", nextSteps, 3); plan != "" {
			sections = append(sections, plan)
		}
	}

	if len(sections) == 1 && strings.TrimSpace(sections[0]) == "面试到这里结束，下面是本场总结。" {
		return ""
	}
	return strings.Join(sections, "\n\n")
}

func shouldPreserveStructuredEvaluationOutput(output string) bool {
	output = strings.TrimSpace(output)
	if output == "" || looksLikeQuestion(output) {
		return false
	}

	scoreLabels := []string{"综合评分", "综合得分", "最终评分", "总分", "overall score"}
	hasScore := false
	lower := strings.ToLower(output)
	for _, label := range scoreLabels {
		if strings.Contains(lower, strings.ToLower(label)) {
			hasScore = true
			break
		}
	}
	if !hasScore {
		return false
	}

	structureSignals := 0
	for _, marker := range []string{
		"维度评分", "亮点", "短板", "待改进", "下一步建议", "学习计划", "strengths", "gaps", "improvements", "study plan",
	} {
		if strings.Contains(lower, strings.ToLower(marker)) {
			structureSignals++
		}
	}
	if strings.Contains(output, "|") {
		structureSignals++
	}
	return structureSignals >= 2
}

func formatWrapupOverallScore(card protocol.Scorecard) string {
	if card.OverallScore <= 0 {
		return ""
	}
	maxScore := card.OverallMaxScore
	if maxScore <= 0 {
		maxScore = 100
	}
	return fmt.Sprintf("综合得分：%d/%d", card.OverallScore, maxScore)
}

func formatWrapupBullets(title string, items []string, limit int) string {
	items = topNonEmptyWrapupItems(items, limit)
	if len(items) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("：")
	for _, item := range items {
		b.WriteString("\n- ")
		b.WriteString(item)
	}
	return b.String()
}

func topNonEmptyWrapupItems(items []string, limit int) []string {
	if limit <= 0 {
		limit = len(items)
	}
	out := make([]string, 0, min(limit, len(items)))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func looksLikeQuestion(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	if strings.Contains(content, "?") || strings.Contains(content, "？") {
		return true
	}
	questionPrefixes := []string{
		"那你",
		"你会怎么",
		"你如何",
		"请你",
		"能不能",
		"可不可以",
		"what",
		"how",
		"why",
	}
	lower := strings.ToLower(content)
	for _, prefix := range questionPrefixes {
		if strings.HasPrefix(lower, strings.ToLower(prefix)) {
			return true
		}
	}
	return false
}

func recordPostInterviewWrapup(
	ctx context.Context,
	recorder runtimepkg.EventRecorder,
	run protocol.Run,
	content string,
	timestamp time.Time,
) error {
	if recorder == nil || strings.TrimSpace(content) == "" {
		return nil
	}

	if err := recorder.RecordMessage(ctx, protocol.Message{
		ID:             uuid.NewString(),
		ConversationID: run.ConversationID,
		TaskID:         run.TaskID,
		RunID:          run.ID,
		Role:           "assistant",
		Content:        content,
		CreatedAt:      timestamp,
	}); err != nil {
		return fmt.Errorf("record post interview wrapup message: %w", err)
	}

	if err := recorder.RecordEvent(ctx, protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: run.ConversationID,
		TaskID:         run.TaskID,
		RunID:          run.ID,
		Type:           protocol.EventMessageCompleted,
		Timestamp:      timestamp,
		Payload: map[string]string{
			"content": content,
		},
	}); err != nil {
		return fmt.Errorf("record post interview wrapup event: %w", err)
	}

	return nil
}

func nextWrapupTimestamp(base time.Time, latestAssistantAt time.Time) time.Time {
	now := time.Now()
	candidate := base
	if candidate.Before(now) {
		candidate = now
	}
	if !latestAssistantAt.IsZero() && !candidate.After(latestAssistantAt) {
		candidate = latestAssistantAt.Add(time.Nanosecond)
	}
	return candidate
}
