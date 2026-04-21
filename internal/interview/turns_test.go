package interview

import "testing"

func TestCountsAsInterviewTurnRecognizesCommonQuestionPhrasings(t *testing.T) {
	t.Parallel()

	cases := []string{
		"请先介绍一个你设计过的 agent runtime 系统。",
		"我想了解你在设计时如何处理工具调用和状态管理。",
		"请展示一下你提到的工具级并发实现。",
	}

	for _, content := range cases {
		if !CountsAsInterviewTurn(content) {
			t.Fatalf("expected content to count as interview turn: %q", content)
		}
	}
}

func TestCountsAsInterviewTurnSkipsWrapupSummary(t *testing.T) {
	t.Parallel()

	content := "面试到这里结束，下面是本场总结。\n\n总评：系统设计思路扎实。\n\n综合得分：88/100"
	if CountsAsInterviewTurn(content) {
		t.Fatalf("did not expect wrapup summary to count as interview turn")
	}
	if !IsWrapupAssistantMessage(content) {
		t.Fatalf("expected wrapup summary to be recognized")
	}
}
