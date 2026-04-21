package middleware

import (
	"strings"
	"testing"
)

func TestAppendResumeResponsePreservesPromptContext(t *testing.T) {
	base := "请继续当前面试，不要重新开始。"
	got := appendResumeResponse(base, "我想继续回答上一个问题")

	if !strings.Contains(got, base) {
		t.Fatalf("expected base prompt to be preserved, got: %s", got)
	}
	if !strings.Contains(got, "Latest user continuation:") {
		t.Fatalf("expected continuation header, got: %s", got)
	}
	if !strings.Contains(got, "我想继续回答上一个问题") {
		t.Fatalf("expected resume response content, got: %s", got)
	}
}

func TestAppendResumeResponseDoesNotOverridePromptWhenResponseEmpty(t *testing.T) {
	base := "请继续当前面试，不要重新开始。"
	got := appendResumeResponse(base, "")

	if got != base {
		t.Fatalf("expected unchanged prompt, got: %s", got)
	}
}
