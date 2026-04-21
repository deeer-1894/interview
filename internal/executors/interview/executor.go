package interview

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	domain "mockinterview/internal/interview"
	"mockinterview/internal/interview/adkapp"
	"mockinterview/internal/protocol"
)

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) Execute(
	ctx context.Context,
	prompt string,
	cfg protocol.InterviewConfig,
	modelCfg protocol.ModelConfig,
	run *protocol.Run,
	skill protocol.SkillSpec,
	messages []protocol.Message,
	resume bool,
	checkpoint protocol.CheckpointSnapshot,
	checkPointStore adk.CheckPointStore,
	onAssistantDelta func(delta string, content string),
) (string, error) {
	if shouldSkipInterviewerExecution(run) {
		return "", nil
	}
	interviewCfg := toInterviewConfig(cfg)
	runPhase := protocol.RunPhaseInitial
	if run != nil && run.Phase != "" {
		runPhase = run.Phase
	}
	state := domain.DefaultRunInterviewState()
	if run != nil {
		state = domain.EnsureRunInterviewState(run.InterviewState)
	}
	if checkpoint.InterviewState != nil && resume {
		state = domain.EnsureRunInterviewState(checkpoint.InterviewState)
	}
	turnPrompt := buildTurnPrompt(prompt, run, messages, skill, state)
	runID := ""
	if run != nil {
		runID = run.ID
	}
	shared := BuildSharedSessionContext(
		runID,
		domain.InterviewPromptVersion,
		runPhase,
		state,
		skill,
		protocol.Rubric{},
		buildTranscriptForModel(messages, 10),
	)
	app, err := adkapp.NewInterviewerApp(ctx, interviewCfg, toModelConfig(modelCfg), adkapp.Options{
		CheckPointStore: checkPointStore,
		SharedContext:   shared,
	})
	if err != nil {
		return "", protocol.WrapModelError("interviewer", "create_interview_app", false, fmt.Errorf("create interview app: %w", err))
	}

	iter, err := runQuery(ctx, app, turnPrompt, run.ID, resume, checkpoint)
	if err != nil {
		return "", protocol.WrapModelError("interviewer", "run_query", isRetryableModelError(err), err)
	}
	result, err := collectAssistantOutput(iter, onAssistantDelta, run.ID, modelCfg)
	if err == nil && strings.TrimSpace(result) != "" {
		updateInterviewState(run, messages, result, skill, cfg)
		return result, nil
	}
	if err == nil {
		return "", protocol.WrapModelError("interviewer", "collect_output", false, fmt.Errorf("interview agent returned no assistant content"))
	}
	if !isRetryableModelError(err) || resume {
		return "", protocol.WrapModelError("interviewer", "collect_output", isRetryableModelError(err), err)
	}

	retryPrompt := slimPromptForRetry(turnPrompt, interviewCfg)
	retryApp, retryErr := adkapp.NewInterviewerApp(ctx, interviewCfg, toModelConfig(modelCfg), adkapp.Options{
		CheckPointStore: checkPointStore,
		SharedContext:   shared,
	})
	if retryErr != nil {
		return "", protocol.WrapModelError("interviewer", "create_retry_interview_app", false, fmt.Errorf("create retry interview app: %w", retryErr))
	}
	retryIter, retryErr := runQuery(ctx, retryApp, retryPrompt, run.ID, false, protocol.CheckpointSnapshot{})
	if retryErr != nil {
		return "", protocol.WrapModelError("interviewer", "retry_run_query", isRetryableModelError(retryErr), retryErr)
	}
	retryResult, retryErr := collectAssistantOutput(retryIter, onAssistantDelta, run.ID, modelCfg)
	if retryErr != nil {
		return "", protocol.WrapModelError("interviewer", "retry_collect_output", isRetryableModelError(retryErr), retryErr)
	}
	if strings.TrimSpace(retryResult) == "" {
		return "", protocol.WrapModelError("interviewer", "retry_collect_output", false, fmt.Errorf("interview agent returned no assistant content after retry"))
	}
	updateInterviewState(run, messages, retryResult, skill, cfg)
	return retryResult, nil
}

func shouldSkipInterviewerExecution(run *protocol.Run) bool {
	return run != nil && run.Phase == protocol.RunPhaseEvaluating
}

func buildTurnPrompt(prompt string, run *protocol.Run, messages []protocol.Message, skill protocol.SkillSpec, state protocol.RunInterviewState) string {
	phase := protocol.RunPhaseInitial
	if run != nil {
		phase = run.Phase
	}
	if phase == "" {
		phase = protocol.RunPhaseInitial
	}
	if phase == protocol.RunPhaseInitial {
		return buildProfileAwarePrompt(strings.TrimSpace(prompt), skill)
	}

	transcript := buildTranscriptForModel(messages, 10)
	if transcript == "" {
		return buildProfileAwarePrompt(strings.TrimSpace(prompt), skill)
	}

	var b strings.Builder
	b.WriteString("Continue the same interview session.\n")
	b.WriteString("Use the transcript as the active conversation history.\n")
	b.WriteString("Do not restart the interview or repeat the first question unless explicitly requested.\n\n")
	if state.Phase != "" {
		b.WriteString("Current interview sub-phase: ")
		b.WriteString(string(state.Phase))
		b.WriteString(".\n")
	}
	b.WriteString("Conversation transcript:\n")
	b.WriteString(transcript)
	return buildProfileAwarePrompt(strings.TrimSpace(b.String()), skill)
}

func buildTranscriptForModel(messages []protocol.Message, limit int) string {
	if len(messages) == 0 {
		return ""
	}
	if limit <= 0 || len(messages) < limit {
		limit = len(messages)
	}
	selected := messages[len(messages)-limit:]
	var b strings.Builder
	for _, message := range selected {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		role := "Assistant"
		if strings.EqualFold(message.Role, "user") {
			role = "User"
		}
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(content)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func updateInterviewState(run *protocol.Run, messages []protocol.Message, assistantOutput string, skill protocol.SkillSpec, cfg protocol.InterviewConfig) {
	if run == nil {
		return
	}
	state := domain.EnsureRunInterviewState(run.InterviewState)
	if !domain.CountsAsInterviewTurn(assistantOutput) {
		run.InterviewState = &state
		return
	}
	maxRounds := domain.DeriveInterviewTurnLimit(cfg.TimeBudget)
	state.Round++

	answer := latestUserAnswer(messages)
	analysis := domain.AnalyzeAnswerSignals(answer, skill)
	decision := state.LastDecision
	if decision == nil {
		fallback := domain.DecideNextStep(protocol.CandidateProfile{}, state, analysis, cfg, skill)
		decision = &fallback
	}
	state.WeakSignals = uniqueMergedSignals(state.WeakSignals, analysis.WeakSignals, 6)
	state.StrongSignals = uniqueMergedSignals(state.StrongSignals, analysis.StrongSignals, 6)
	switch {
	case decision.IncreaseDifficulty:
		if state.Difficulty < 5 {
			state.Difficulty++
		}
	case analysis.TooGeneric:
		if state.Difficulty > 1 {
			state.Difficulty--
		}
	}
	if decision.EscalatePressure && state.Difficulty < 5 {
		state.Difficulty++
	}
	domain.AdvancePhaseForMode(&state, maxRounds, cfg.Mode)
	scenario := domain.SelectScenario(skill, state)
	snapshot := protocol.InterviewRoundSnapshot{
		Round:                  state.Round,
		Phase:                  state.Phase,
		Difficulty:             state.Difficulty,
		Scenario:               scenario,
		Adversarial:            decision.TriggerAdversarial || domain.ShouldTriggerAdversarial(state, analysis),
		Pressure:               state.Phase == protocol.PhaseStress || state.Phase == protocol.PhaseWrapup,
		Reason:                 string(decision.Reason),
		Explanation:            decision.Explanation,
		WeakSignals:            append([]string(nil), analysis.WeakSignals...),
		StrongSignals:          append([]string(nil), analysis.StrongSignals...),
		WeakSignalConfidence:   domain.CloneSignalConfidence(analysis.WeakSignalConfidence),
		StrongSignalConfidence: domain.CloneSignalConfidence(analysis.StrongSignalConfidence),
	}
	if scenario != "" {
		state.LastScenario = scenario
	}
	state.LastDecision = decision
	domain.AppendRoundSnapshot(&state, snapshot)
	run.InterviewState = &state
}

func latestUserAnswer(messages []protocol.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.EqualFold(messages[i].Role, "user") {
			return strings.TrimSpace(messages[i].Content)
		}
	}
	return ""
}

func uniqueMergedSignals(base []string, extra []string, limit int) []string {
	out := append([]string(nil), base...)
	seen := map[string]struct{}{}
	for _, item := range out {
		seen[strings.ToLower(strings.TrimSpace(item))] = struct{}{}
	}
	for _, item := range extra {
		key := strings.ToLower(strings.TrimSpace(item))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, strings.TrimSpace(item))
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func buildProfileAwarePrompt(prompt string, skill protocol.SkillSpec) string {
	prompt = strings.TrimSpace(prompt)
	if len(skill.FocusAreas) == 0 {
		return prompt
	}
	var b strings.Builder
	b.WriteString(prompt)
	b.WriteString("\n\nPriority focus areas:\n")
	for _, area := range skill.FocusAreas[:minInt(len(skill.FocusAreas), 3)] {
		b.WriteString("- ")
		b.WriteString(area)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func runQuery(ctx context.Context, app *adkapp.App, prompt, runID string, resume bool, checkpoint protocol.CheckpointSnapshot) (*adk.AsyncIterator[*adk.AgentEvent], error) {
	if resume && len(checkpoint.RawState) > 0 {
		iter, err := app.Runner.Resume(ctx, runID)
		if err == nil {
			return iter, nil
		}
	}
	return app.Runner.Query(ctx, prompt, adk.WithCheckPointID(runID)), nil
}

func toInterviewConfig(cfg protocol.InterviewConfig) domain.InterviewConfig {
	return domain.InterviewConfig{
		Skill:        cfg.Skill,
		SkillFocuses: append([]string(nil), cfg.SkillFocuses...),
		Persona:      domain.Persona(cfg.Persona),
		Level:        cfg.Level,
		Focus:        cfg.Focus,
		Mode:         domain.Mode(cfg.Mode),
		TimeBudget:   cfg.TimeBudget,
		OutputStyle:  domain.OutputStyle(cfg.OutputStyle),
	}.WithDefaults()
}

func toModelConfig(cfg protocol.ModelConfig) domain.ModelConfig {
	return domain.ModelConfig{
		Provider: domain.ModelProvider(cfg.Provider),
		Model:    cfg.Model,
		APIKey:   cfg.APIKey,
		BaseURL:  cfg.BaseURL,
	}.WithDefaults()
}

func collectAssistantOutput(iter *adk.AsyncIterator[*adk.AgentEvent], onAssistantDelta func(delta string, content string), runID string, modelCfg protocol.ModelConfig) (string, error) {
	var out strings.Builder
	previousSanitized := ""
	debugStream := streamDebugEnabled()
	startedAt := time.Now()
	lastChunkAt := startedAt
	chunkCount := 0
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		if event.Err != nil {
			return "", event.Err
		}

		msg, _, err := adk.GetMessage(event)
		if err != nil || msg == nil || msg.Role != schema.Assistant {
			continue
		}

		chunk := msg.Content
		if chunk == "" {
			continue
		}
		out.WriteString(chunk)
		chunkCount++
		if debugStream {
			now := time.Now()
			slog.Info(
				"stream_debug",
				"run_id", runID,
				"provider", firstNonEmptyString(modelCfg.Provider, "default"),
				"model", firstNonEmptyString(modelCfg.Model, "default"),
				"chunk", chunkCount,
				"bytes", len([]byte(chunk)),
				"total_bytes", out.Len(),
				"first_latency_ms", now.Sub(startedAt).Round(time.Millisecond).Milliseconds(),
				"delta_interval_ms", now.Sub(lastChunkAt).Round(time.Millisecond).Milliseconds(),
			)
			lastChunkAt = now
		}
		if onAssistantDelta != nil {
			sanitized := sanitizeInterviewAssistantOutput(out.String())
			if sanitized != previousSanitized {
				delta := sanitized
				if strings.HasPrefix(sanitized, previousSanitized) {
					delta = strings.TrimPrefix(sanitized, previousSanitized)
				}
				previousSanitized = sanitized
				if delta != "" || sanitized != "" {
					onAssistantDelta(delta, sanitized)
				}
			}
		}
	}
	if debugStream {
		slog.Info(
			"stream_debug_completed",
			"run_id", runID,
			"provider", firstNonEmptyString(modelCfg.Provider, "default"),
			"model", firstNonEmptyString(modelCfg.Model, "default"),
			"chunks", chunkCount,
			"total_bytes", out.Len(),
			"duration_ms", time.Since(startedAt).Round(time.Millisecond).Milliseconds(),
		)
	}
	return sanitizeInterviewAssistantOutput(out.String()), nil
}

func sanitizeInterviewAssistantOutput(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	content = stripInterviewToolProtocol(content)
	if content == "" {
		return ""
	}

	paragraphs := strings.Split(content, "\n\n")
	start := 0
	for start < len(paragraphs) {
		if !isInterviewMetaParagraph(paragraphs[start]) {
			break
		}
		start++
	}
	return strings.TrimSpace(strings.Join(paragraphs[start:], "\n\n"))
}

var interviewToolProtocolTagPattern = regexp.MustCompile(`(?is)</?tool_call>\s*|<arg_key>.*?</arg_key>\s*|<arg_value>.*?</arg_value>\s*`)
var interviewMetaNoisePattern = regexp.MustCompile(`[[:punct:]\p{P}]+`)

func stripInterviewToolProtocol(content string) string {
	content = interviewToolProtocolTagPattern.ReplaceAllString(content, "")
	lines := strings.Split(content, "\n")
	cleaned := make([]string, 0, len(lines))
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			cleaned = append(cleaned, "")
			continue
		}
		if trimmed == "skill" && (hasAdjacentProtocolResidue(lines, index) || isDanglingProtocolLabel(lines, index)) {
			continue
		}
		if strings.Contains(trimmed, "<arg_key>") || strings.Contains(trimmed, "<arg_value>") || strings.Contains(trimmed, "tool_call") {
			continue
		}
		cleaned = append(cleaned, line)
	}
	return strings.TrimSpace(strings.Join(collapseBlankLineRuns(cleaned), "\n"))
}

func hasAdjacentProtocolResidue(lines []string, index int) bool {
	for _, neighbor := range []int{index - 1, index + 1} {
		if neighbor < 0 || neighbor >= len(lines) {
			continue
		}
		trimmed := strings.TrimSpace(lines[neighbor])
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, "<arg_key>") || strings.Contains(trimmed, "<arg_value>") || strings.Contains(trimmed, "tool_call") {
			return true
		}
	}
	return false
}

func isDanglingProtocolLabel(lines []string, index int) bool {
	if strings.TrimSpace(lines[index]) != "skill" {
		return false
	}
	for _, line := range lines[index+1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		return false
	}
	return true
}

func collapseBlankLineRuns(lines []string) []string {
	if len(lines) == 0 {
		return nil
	}
	out := make([]string, 0, len(lines))
	previousBlank := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			if previousBlank {
				continue
			}
			previousBlank = true
			out = append(out, "")
			continue
		}
		previousBlank = false
		out = append(out, line)
	}
	return out
}

func isInterviewMetaParagraph(paragraph string) bool {
	normalized := normalizeInterviewMetaText(paragraph)
	if normalized == "" {
		return false
	}

	metaPatterns := []string{
		"好的 我们开始这场",
		"我们开始这场",
		"开始这场 go agent 开发岗位的技术面试",
		"开始这场系统设计面试",
		"开始这场技术面试",
		"我会保持严谨的面试风格",
		"重点关注工程实现细节和系统设计能力",
		"首先调用面试技能工具",
		"首先加载面试技能配置",
		"加载面试技能配置",
		"调用面试技能工具",
		"加载技能配置",
		"先调用技能工具",
		"先加载技能配置",
		"首先让我加载面试框架",
		"首先让我们加载面试框架",
		"让我加载面试框架",
		"让我们加载面试框架",
		"开始本场系统设计面试",
		"开始本场技术面试",
		"我将作为你的面试官",
		"我会作为你的面试官",
		"i will be your interviewer",
		"let me load the interview framework",
		"first let me load the interview framework",
		"start this system design interview",
		"start this technical interview",
		"first load the interview skill",
		"first load the interview skills",
		"load the interview skill config",
		"load the interview skill configuration",
		"call the interview skill tool",
	}
	for _, pattern := range metaPatterns {
		if strings.Contains(normalized, normalizeInterviewMetaText(pattern)) {
			return true
		}
	}

	return strings.Contains(normalized, "技能") &&
		strings.Contains(normalized, "面试") &&
		(strings.Contains(normalized, "调用") || strings.Contains(normalized, "加载"))
}

func normalizeInterviewMetaText(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = interviewMetaNoisePattern.ReplaceAllString(normalized, " ")
	fields := strings.Fields(normalized)
	return strings.Join(fields, " ")
}

func streamDebugEnabled() bool {
	raw := strings.TrimSpace(os.Getenv("STREAM_DEBUG"))
	return strings.EqualFold(raw, "1") || strings.EqualFold(raw, "true") || strings.EqualFold(raw, "yes")
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isRetryableModelError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "context deadline exceeded") ||
		strings.Contains(text, "client.timeout") ||
		strings.Contains(text, "failed to receive stream chunk") ||
		strings.Contains(text, "timed out")
}

func slimPromptForRetry(prompt string, cfg domain.InterviewConfig) string {
	lines := strings.Split(prompt, "\n")
	filtered := make([]string, 0, len(lines))
	skipIndented := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case "Recent compact memory:", "Relevant workspace artifacts:", "Recent web references:":
			skipIndented = true
			continue
		}
		if skipIndented {
			if trimmed == "" {
				skipIndented = false
			}
			continue
		}
		filtered = append(filtered, line)
	}

	var b strings.Builder
	b.WriteString(strings.TrimSpace(strings.Join(filtered, "\n")))
	b.WriteString("\n\nRetry instruction:\n")
	b.WriteString("- Keep this turn concise.\n")
	b.WriteString("- Return one short framing sentence and exactly one strong interview question.\n")
	if cfg.OutputStyle != domain.OutputInterviewOnly {
		b.WriteString("- Do not generate scoring or a study plan in this turn unless the user explicitly asks to end the interview.\n")
	}
	return b.String()
}
