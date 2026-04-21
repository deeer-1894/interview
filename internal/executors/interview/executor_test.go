package interview

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	domain "mockinterview/internal/interview"
	"mockinterview/internal/interview/adkapp"
	"mockinterview/internal/protocol"
)

func TestNewExecutorReturnsInstance(t *testing.T) {
	t.Parallel()

	if NewExecutor() == nil {
		t.Fatalf("expected executor instance")
	}
}

func TestShouldSkipInterviewerExecutionWhenRunAlreadyEvaluating(t *testing.T) {
	t.Parallel()

	if !shouldSkipInterviewerExecution(&protocol.Run{Phase: protocol.RunPhaseEvaluating}) {
		t.Fatalf("expected evaluating run to skip interviewer execution")
	}
	if shouldSkipInterviewerExecution(&protocol.Run{Phase: protocol.RunPhaseInterviewing}) {
		t.Fatalf("expected interviewing run not to skip interviewer execution")
	}
}

func TestBuildTranscriptForModelKeepsRecentMessagesAndSkipsBlank(t *testing.T) {
	t.Parallel()

	transcript := buildTranscriptForModel([]protocol.Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "reply"},
		{Role: "user", Content: "   "},
		{Role: "user", Content: "latest question"},
	}, 3)

	if strings.Contains(transcript, "first") {
		t.Fatalf("expected transcript to keep only recent messages, got %q", transcript)
	}
	if !strings.Contains(transcript, "Assistant: reply") || !strings.Contains(transcript, "User: latest question") {
		t.Fatalf("unexpected transcript rendering: %q", transcript)
	}
}

func TestBuildTurnPromptUsesInitialPromptAndResumeTranscript(t *testing.T) {
	t.Parallel()

	initial := buildTurnPrompt(
		"Design a resilient worker pipeline.",
		&protocol.Run{Phase: protocol.RunPhaseInitial},
		nil,
		protocol.SkillSpec{FocusAreas: []string{"observability", "timeouts", "retries", "ownership"}},
		protocol.RunInterviewState{},
	)
	if !strings.Contains(initial, "Priority focus areas:") || strings.Contains(initial, "ownership") {
		t.Fatalf("expected initial prompt to append only top 3 focuses, got %q", initial)
	}

	resumed := buildTurnPrompt(
		"ignored on resume",
		&protocol.Run{Phase: protocol.RunPhaseInterviewing},
		[]protocol.Message{
			{Role: "user", Content: "How do you limit fan-out?"},
			{Role: "assistant", Content: "I would start with queues."},
		},
		protocol.SkillSpec{FocusAreas: []string{"observability"}},
		protocol.RunInterviewState{Phase: protocol.PhaseStress},
	)
	if !strings.Contains(resumed, "Continue the same interview session.") {
		t.Fatalf("expected resume prompt framing, got %q", resumed)
	}
	if !strings.Contains(resumed, "Current interview sub-phase: stress.") {
		t.Fatalf("expected resume prompt to include sub-phase, got %q", resumed)
	}
	if !strings.Contains(resumed, "User: How do you limit fan-out?") {
		t.Fatalf("expected transcript to be embedded, got %q", resumed)
	}

	fallback := buildTurnPrompt(
		"Ask the next question.",
		&protocol.Run{Phase: protocol.RunPhaseInterviewing},
		nil,
		protocol.SkillSpec{},
		protocol.RunInterviewState{},
	)
	if fallback != "Ask the next question." {
		t.Fatalf("expected empty transcript to fall back to original prompt, got %q", fallback)
	}
}

func TestUpdateInterviewStateAppliesDecisionAndFallbackAnalysis(t *testing.T) {
	t.Parallel()

	run := &protocol.Run{
		InterviewState: &protocol.RunInterviewState{
			Phase:      protocol.PhaseProbe,
			Round:      1,
			Difficulty: 3,
			LastDecision: &protocol.NextStepDecision{
				IncreaseDifficulty: true,
				EscalatePressure:   true,
				TriggerAdversarial: true,
				Reason:             protocol.ReasonMissingTradeoff,
				Explanation:        "继续上强度检查恢复策略。",
			},
		},
	}
	messages := []protocol.Message{
		{Role: "assistant", Content: "How do you handle retries?"},
		{Role: "user", Content: "我会加 timeout、retry、trace 和 metrics，同时用队列兜底。"},
	}
	skill := protocol.SkillSpec{
		FocusAreas: []string{"observability"},
		Scenarios:  []string{"A dependency becomes flaky."},
	}

	updateInterviewState(run, messages, "How do you handle retries?", skill, protocol.InterviewConfig{
		Mode:       protocol.ModeStress,
		TimeBudget: "15 minutes",
	})

	if run.InterviewState == nil {
		t.Fatalf("expected updated interview state")
	}
	if run.InterviewState.Round != 2 || run.InterviewState.Difficulty != 5 {
		t.Fatalf("expected round increment and capped difficulty increase, got %#v", run.InterviewState)
	}
	if run.InterviewState.Phase != protocol.PhaseAdversarial {
		t.Fatalf("expected stress mode to reach adversarial at round 2, got %#v", run.InterviewState)
	}
	if run.InterviewState.LastScenario != "A dependency becomes flaky." {
		t.Fatalf("expected scenario to be selected, got %#v", run.InterviewState)
	}
	if len(run.InterviewState.History) != 1 || !run.InterviewState.History[0].Adversarial {
		t.Fatalf("expected history snapshot with adversarial flag, got %#v", run.InterviewState.History)
	}
	if !containsString(run.InterviewState.StrongSignals, "observability") {
		t.Fatalf("expected strong observability signal, got %#v", run.InterviewState.StrongSignals)
	}

	fallbackRun := &protocol.Run{
		InterviewState: &protocol.RunInterviewState{
			Phase:      protocol.PhaseProbe,
			Round:      1,
			Difficulty: 3,
		},
	}
	updateInterviewState(fallbackRun, []protocol.Message{{Role: "user", Content: "这个需要看情况。"}}, "请说明你的取舍。", protocol.SkillSpec{}, protocol.InterviewConfig{
		Mode:       protocol.ModeStandard,
		TimeBudget: "25 minutes",
	})

	if fallbackRun.InterviewState.LastDecision == nil {
		t.Fatalf("expected fallback decision to be synthesized")
	}
	if fallbackRun.InterviewState.Difficulty > 3 {
		t.Fatalf("expected generic fallback path not to keep increasing difficulty, got %#v", fallbackRun.InterviewState)
	}
	if !containsString(fallbackRun.InterviewState.WeakSignals, "too_generic") {
		t.Fatalf("expected too_generic weak signal, got %#v", fallbackRun.InterviewState.WeakSignals)
	}
}

func TestUpdateInterviewStateSkipsWrapupSummaryOutputs(t *testing.T) {
	t.Parallel()

	run := &protocol.Run{
		InterviewState: &protocol.RunInterviewState{
			Phase:      protocol.PhaseWrapup,
			Round:      6,
			Difficulty: 5,
		},
	}

	updateInterviewState(
		run,
		[]protocol.Message{{Role: "user", Content: "请现在结束这场面试并给我最终总结。"}},
		"你的设计思路清晰，展现了高级工程师的系统设计能力，但在具体实现细节和编码实践方面还需要加强。",
		protocol.SkillSpec{},
		protocol.InterviewConfig{Mode: protocol.ModeStandard, TimeBudget: "45 minutes"},
	)

	if run.InterviewState.Round != 6 {
		t.Fatalf("expected wrapup summary not to create a new round, got %#v", run.InterviewState)
	}
	if len(run.InterviewState.History) != 0 {
		t.Fatalf("expected wrapup summary not to append history, got %#v", run.InterviewState.History)
	}
}

func TestExecutorUtilityHelpersCoverTimeBudgetAndRetryLogic(t *testing.T) {
	t.Parallel()

	if latestUserAnswer([]protocol.Message{{Role: "assistant", Content: "a"}, {Role: "user", Content: " final "}}) != "final" {
		t.Fatalf("expected latest user answer to be trimmed")
	}

	merged := uniqueMergedSignals([]string{"observability"}, []string{"Observability", "timeouts", "retry", "trace"}, 3)
	if len(merged) != 3 || merged[1] != "timeouts" {
		t.Fatalf("unexpected merged signals: %#v", merged)
	}

	if minutes := domain.ParseTimeBudgetMinutes("about 45 Minutes"); minutes != 45 {
		t.Fatalf("expected parsed minutes, got %d", minutes)
	}
	if minutes := domain.ParseTimeBudgetMinutes("not sure"); minutes != 0 {
		t.Fatalf("expected invalid budget to parse as 0, got %d", minutes)
	}
	if limit := domain.DeriveInterviewTurnLimit("45 minutes"); limit != 12 {
		t.Fatalf("expected 45 minutes to map to 12 turns, got %d", limit)
	}
	if limit := domain.DeriveInterviewTurnLimit(""); limit != 5 {
		t.Fatalf("expected empty budget fallback to 5 turns, got %d", limit)
	}

	retryPrompt := slimPromptForRetry("Intro\nRecent compact memory:\n  note\n\nRelevant workspace artifacts:\n  file\n\nKeep this line", domain.InterviewConfig{})
	if strings.Contains(retryPrompt, "Recent compact memory") || strings.Contains(retryPrompt, "workspace artifacts") {
		t.Fatalf("expected retry prompt to strip bulky sections, got %q", retryPrompt)
	}
	if !strings.Contains(retryPrompt, "exactly one strong interview question") {
		t.Fatalf("expected retry instruction suffix, got %q", retryPrompt)
	}

	if !isRetryableModelError(assertError("client.timeout while waiting")) {
		t.Fatalf("expected retryable model error match")
	}
	if isRetryableModelError(assertError("validation failed")) {
		t.Fatalf("expected non-retryable error to be ignored")
	}

	if prompt := buildProfileAwarePrompt("plain prompt", protocol.SkillSpec{}); prompt != "plain prompt" {
		t.Fatalf("expected no-focus prompt to stay unchanged, got %q", prompt)
	}
	if answer := latestUserAnswer([]protocol.Message{{Role: "assistant", Content: "no user"}}); answer != "" {
		t.Fatalf("expected empty answer when no user message exists, got %q", answer)
	}
}

func TestExecutorConfigAndEnvHelpersApplyDefaults(t *testing.T) {
	t.Setenv("STREAM_DEBUG", "true")
	if !streamDebugEnabled() {
		t.Fatalf("expected STREAM_DEBUG=true to enable debug")
	}
	t.Setenv("STREAM_DEBUG", "0")
	if streamDebugEnabled() {
		t.Fatalf("expected STREAM_DEBUG=0 to disable debug")
	}

	if value := firstNonEmptyString(" ", "fallback"); value != "fallback" {
		t.Fatalf("unexpected first non-empty value: %q", value)
	}
	if minInt(4, 2) != 2 {
		t.Fatalf("expected minInt to return smaller value")
	}

	interviewCfg := toInterviewConfig(protocol.InterviewConfig{
		Skill:        "go-agent",
		SkillFocuses: []string{"observability", "observability"},
		Mode:         protocol.ModeStress,
		OutputStyle:  protocol.OutputInterviewPlusStudy,
	})
	if interviewCfg.Persona != domain.PersonaRigorous || interviewCfg.TimeBudget != "25 minutes" {
		t.Fatalf("expected interview config defaults, got %#v", interviewCfg)
	}
	if len(interviewCfg.SkillFocuses) != 1 || interviewCfg.SkillFocuses[0] != "observability" {
		t.Fatalf("expected focus normalization, got %#v", interviewCfg.SkillFocuses)
	}

	modelCfg := toModelConfig(protocol.ModelConfig{Model: "gpt-test"})
	if modelCfg.Provider != domain.ProviderOpenAI || modelCfg.Timeout == 0 {
		t.Fatalf("expected model config defaults, got %#v", modelCfg)
	}
}

func TestCollectAssistantOutputConcatenatesAssistantChunks(t *testing.T) {
	t.Parallel()

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(nil)
	gen.Send(adk.EventFromMessage(schema.UserMessage("ignore me"), nil, schema.User, ""))
	gen.Send(adk.EventFromMessage(schema.AssistantMessage("Hello", nil), nil, schema.Assistant, ""))
	gen.Send(adk.EventFromMessage(schema.AssistantMessage(" world", nil), nil, schema.Assistant, ""))
	gen.Close()

	var deltas []string
	var snapshots []string
	output, err := collectAssistantOutput(iter, func(delta string, content string) {
		deltas = append(deltas, delta)
		snapshots = append(snapshots, content)
	}, "run_1", protocol.ModelConfig{Provider: "openai", Model: "gpt-test"})
	if err != nil {
		t.Fatalf("collectAssistantOutput returned error: %v", err)
	}
	if output != "Hello world" {
		t.Fatalf("unexpected output: %q", output)
	}
	if len(deltas) != 2 || deltas[0] != "Hello" || snapshots[1] != "Hello world" {
		t.Fatalf("unexpected callback snapshots: deltas=%#v snapshots=%#v", deltas, snapshots)
	}
}

func TestCollectAssistantOutputReturnsStreamError(t *testing.T) {
	t.Parallel()

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(&adk.AgentEvent{Err: simpleError("stream failed")})
	gen.Close()

	output, err := collectAssistantOutput(iter, nil, "run_2", protocol.ModelConfig{})
	if err == nil || err.Error() != "stream failed" {
		t.Fatalf("expected propagated stream error, got output=%q err=%v", output, err)
	}
}

func TestCollectAssistantOutputDebugModeSkipsEmptyChunks(t *testing.T) {
	t.Setenv("STREAM_DEBUG", "true")

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(adk.EventFromMessage(schema.AssistantMessage("", nil), nil, schema.Assistant, ""))
	gen.Send(adk.EventFromMessage(schema.AssistantMessage("answer", nil), nil, schema.Assistant, ""))
	gen.Close()

	output, err := collectAssistantOutput(iter, nil, "run_debug", protocol.ModelConfig{Provider: "openai", Model: "gpt-test"})
	if err != nil {
		t.Fatalf("collectAssistantOutput returned error: %v", err)
	}
	if output != "answer" {
		t.Fatalf("expected empty chunks to be skipped, got %q", output)
	}
}

func TestCollectAssistantOutputStripsInterviewSkillMetaPreamble(t *testing.T) {
	t.Parallel()

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(adk.EventFromMessage(schema.AssistantMessage("我将开始这场 Go agent 开发岗位的技术面试。首先调用面试技能工具。\n\n", nil), nil, schema.Assistant, ""))
	gen.Send(adk.EventFromMessage(schema.AssistantMessage("请先介绍你的并发模型。", nil), nil, schema.Assistant, ""))
	gen.Close()

	var snapshots []string
	output, err := collectAssistantOutput(iter, func(_ string, content string) {
		snapshots = append(snapshots, content)
	}, "run_clean", protocol.ModelConfig{})
	if err != nil {
		t.Fatalf("collectAssistantOutput returned error: %v", err)
	}
	if strings.Contains(output, "首先调用面试技能工具") {
		t.Fatalf("expected meta preamble to be stripped, got %q", output)
	}
	if output != "请先介绍你的并发模型。" {
		t.Fatalf("expected substantive interview content to remain, got %q", output)
	}
	if len(snapshots) == 0 || strings.Contains(snapshots[len(snapshots)-1], "首先调用面试技能工具") {
		t.Fatalf("expected streamed snapshots to stay sanitized, got %#v", snapshots)
	}
}

func TestCollectAssistantOutputStripsGenericInterviewIntroParagraph(t *testing.T) {
	t.Parallel()

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(adk.EventFromMessage(schema.AssistantMessage("好的，我们开始这场 Go agent 开发岗位的技术面试。我会保持严谨的面试风格，重点关注工程实现细节和系统设计能力。\n\n", nil), nil, schema.Assistant, ""))
	gen.Send(adk.EventFromMessage(schema.AssistantMessage("场景设定：你需要设计一个 Go agent 系统，该系统需要调用多个外部工具来完成复杂任务。第一个问题：在 Go 中，你会如何设计这个 agent 系统的并发控制机制？", nil), nil, schema.Assistant, ""))
	gen.Close()

	var snapshots []string
	output, err := collectAssistantOutput(iter, func(_ string, content string) {
		snapshots = append(snapshots, content)
	}, "run_intro_clean", protocol.ModelConfig{})
	if err != nil {
		t.Fatalf("collectAssistantOutput returned error: %v", err)
	}
	for _, fragment := range []string{"开始这场 Go agent 开发岗位的技术面试", "保持严谨的面试风格", "工程实现细节"} {
		if strings.Contains(output, fragment) {
			t.Fatalf("expected generic intro fragment %q to be stripped, got %q", fragment, output)
		}
	}
	if !strings.HasPrefix(output, "场景设定：") {
		t.Fatalf("expected scenario question to remain, got %q", output)
	}
	if len(snapshots) == 0 || strings.Contains(snapshots[len(snapshots)-1], "保持严谨的面试风格") {
		t.Fatalf("expected streamed snapshots to stay sanitized, got %#v", snapshots)
	}
}

func TestCollectAssistantOutputStripsInterviewerFrameworkPreamble(t *testing.T) {
	t.Parallel()

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(adk.EventFromMessage(schema.AssistantMessage("我将作为你的面试官，开始这场系统设计面试。我会保持严谨的态度，逐步深入技术细节。首先，让我加载面试框架。\n\n", nil), nil, schema.Assistant, ""))
	gen.Send(adk.EventFromMessage(schema.AssistantMessage("先说说你会如何设计一个支持工具调用、记忆和评估闭环的 agent runtime。", nil), nil, schema.Assistant, ""))
	gen.Close()

	var snapshots []string
	output, err := collectAssistantOutput(iter, func(_ string, content string) {
		snapshots = append(snapshots, content)
	}, "run_framework_clean", protocol.ModelConfig{})
	if err != nil {
		t.Fatalf("collectAssistantOutput returned error: %v", err)
	}
	for _, fragment := range []string{"作为你的面试官", "加载面试框架", "系统设计面试"} {
		if strings.Contains(output, fragment) {
			t.Fatalf("expected interviewer meta fragment %q to be stripped, got %q", fragment, output)
		}
	}
	if output != "先说说你会如何设计一个支持工具调用、记忆和评估闭环的 agent runtime。" {
		t.Fatalf("expected substantive interview question to remain, got %q", output)
	}
	if len(snapshots) == 0 || strings.Contains(snapshots[len(snapshots)-1], "加载面试框架") {
		t.Fatalf("expected streamed snapshots to stay sanitized, got %#v", snapshots)
	}
}

func TestCollectAssistantOutputStripsToolProtocolFragments(t *testing.T) {
	t.Parallel()

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(adk.EventFromMessage(schema.AssistantMessage("请解释你的统一可观测性方案。\nskill\n<arg_key>skill</arg_key>\n<arg_value>agent-interview-sim</arg_value>\n</tool_call>", nil), nil, schema.Assistant, ""))
	gen.Close()

	output, err := collectAssistantOutput(iter, nil, "run_tool_proto", protocol.ModelConfig{})
	if err != nil {
		t.Fatalf("collectAssistantOutput returned error: %v", err)
	}
	for _, fragment := range []string{"<arg_key>", "<arg_value>", "</tool_call>", "\nskill\n"} {
		if strings.Contains(output, fragment) {
			t.Fatalf("expected tool protocol fragment %q to be stripped, got %q", fragment, output)
		}
	}
	if output != "请解释你的统一可观测性方案。" {
		t.Fatalf("expected substantive question to remain, got %q", output)
	}
}

func TestRunQueryFallsBackToQueryWhenResumeCannotLoadCheckpoint(t *testing.T) {
	t.Parallel()

	runner := adk.NewRunner(context.Background(), adk.RunnerConfig{
		Agent: fakeAgent{response: "resumed-by-query"},
	})
	iter, err := runQuery(context.Background(), &adkapp.App{Runner: runner}, "continue interview", "run_3", true, protocol.CheckpointSnapshot{
		RawState: []byte("checkpoint exists"),
	})
	if err != nil {
		t.Fatalf("runQuery returned error: %v", err)
	}

	output, err := collectAssistantOutput(iter, nil, "run_3", protocol.ModelConfig{})
	if err != nil {
		t.Fatalf("collectAssistantOutput returned error: %v", err)
	}
	if output != "resumed-by-query" {
		t.Fatalf("expected query fallback output, got %q", output)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}

func assertError(text string) error {
	return simpleError(text)
}

type simpleError string

func (e simpleError) Error() string {
	return string(e)
}

type fakeAgent struct {
	response string
}

func (a fakeAgent) Name(context.Context) string {
	return "fake"
}

func (a fakeAgent) Description(context.Context) string {
	return "fake"
}

func (a fakeAgent) Run(ctx context.Context, input *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(adk.EventFromMessage(schema.AssistantMessage(a.response, nil), nil, schema.Assistant, ""))
	gen.Close()
	return iter
}
