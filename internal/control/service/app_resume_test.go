package service

import (
	"strings"
	"testing"
	"time"

	"mockinterview/internal/protocol"
)

func TestBuildResumePromptIncludesRecentTranscript(t *testing.T) {
	now := time.Now()
	messages := []protocol.Message{
		{Role: "user", Content: "请模拟一场 Go agent 开发岗位的技术面试", CreatedAt: now.Add(-5 * time.Minute)},
		{Role: "assistant", Content: "第一个问题：请设计并发工具调用架构。", CreatedAt: now.Add(-4 * time.Minute)},
		{Role: "user", Content: "我会用 errgroup 和 context 控制并发和超时。", CreatedAt: now.Add(-3 * time.Minute)},
		{Role: "assistant", Content: "很好。接着说说失败隔离怎么做。", CreatedAt: now.Add(-2 * time.Minute)},
		{Role: "user", Content: "继续", CreatedAt: now.Add(-1 * time.Minute)},
	}

	prompt := buildResumePrompt("请继续当前面试。", messages, "继续")

	if !strings.Contains(prompt, "Continue the same interview instead of restarting.") {
		t.Fatalf("expected resume instruction in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "Recent transcript:") {
		t.Fatalf("expected transcript header in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "Assistant: 第一个问题：请设计并发工具调用架构。") {
		t.Fatalf("expected assistant transcript in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "User: 我会用 errgroup 和 context 控制并发和超时。") {
		t.Fatalf("expected user transcript in prompt, got: %s", prompt)
	}
}

func TestBuildCompactTranscriptLimitsRecentMessages(t *testing.T) {
	now := time.Now()
	messages := []protocol.Message{
		{Role: "user", Content: "m1", CreatedAt: now.Add(-6 * time.Minute)},
		{Role: "assistant", Content: "m2", CreatedAt: now.Add(-5 * time.Minute)},
		{Role: "user", Content: "m3", CreatedAt: now.Add(-4 * time.Minute)},
		{Role: "assistant", Content: "m4", CreatedAt: now.Add(-3 * time.Minute)},
		{Role: "user", Content: "m5", CreatedAt: now.Add(-2 * time.Minute)},
		{Role: "assistant", Content: "m6", CreatedAt: now.Add(-1 * time.Minute)},
		{Role: "user", Content: "m7", CreatedAt: now},
	}

	transcript := buildCompactTranscript(messages, 3)

	if strings.Contains(transcript, "m1") || strings.Contains(transcript, "m2") || strings.Contains(transcript, "m3") || strings.Contains(transcript, "m4") {
		t.Fatalf("expected only the last 3 messages, got: %s", transcript)
	}
	if !strings.Contains(transcript, "m5") || !strings.Contains(transcript, "m6") || !strings.Contains(transcript, "m7") {
		t.Fatalf("expected final messages in transcript, got: %s", transcript)
	}
}

func TestRunHasFinalWrapupRequiresEvaluationArtifacts(t *testing.T) {
	t.Parallel()

	messages := []protocol.Message{
		{Role: "assistant", Content: "面试到这里结束，下面是本场总结。\n\n总评：并发设计思路扎实。\n\n综合得分：88/100"},
	}

	if runHasFinalWrapup(messages, nil) {
		t.Fatalf("did not expect wrapup detection without score/review artifacts")
	}
	if !runHasFinalWrapup(messages, []protocol.Event{{Type: protocol.EventScoreGenerated}}) {
		t.Fatalf("expected score artifact plus wrapup summary to mark run as finalized")
	}
}

func TestRunHasFinalWrapupScansAnyAssistantSummary(t *testing.T) {
	t.Parallel()

	messages := []protocol.Message{
		{Role: "assistant", Content: "面试到这里结束，下面是本场总结。\n\n总评：候选人表现稳定。\n\n综合得分：82/100"},
		{Role: "assistant", Content: "请继续说明混沌测试如何落地？"},
	}

	if !runHasFinalWrapup(messages, []protocol.Event{{Type: protocol.EventReviewGenerated}}) {
		t.Fatalf("expected any historical wrapup summary plus review artifact to block same-run resume")
	}
}
