package interview

import (
	"strings"

	"mockinterview/internal/protocol"
)

var interviewTurnQuestionCues = []string{
	"?",
	"？",
	"**问题",
	"问题：",
	"question:",
	"follow-up",
	"你会如何",
	"你如何",
	"你会怎么",
	"你怎么",
	"请介绍",
	"请先介绍",
	"请解释",
	"请说明",
	"请描述",
	"请分析",
	"请给出",
	"请提供",
	"请展示",
	"请设计",
	"请写",
	"另外，请",
	"另外请",
	"介绍一下",
	"说明一下",
	"我想了解",
	"如果",
	"假设",
	"how ",
	"what ",
	"why ",
	"describe ",
	"explain ",
	"walk me through",
	"tell me",
}

var interviewTurnWrapupCues = []string{
	"面试到这里结束",
	"本场总结",
	"总评：",
	"综合得分：",
	"亮点：",
	"待改进：",
	"下一步建议：",
	"学习计划",
	"最终评分",
	"最终评估",
	"综合评价",
	"综合来看",
}

var explicitWrapupRequestCues = []string{
	"请结束",
	"现在结束",
	"到这里结束",
	"结束这场面试",
	"结束本场面试",
	"结束面试",
	"面试到这里结束",
	"面试到此结束",
	"不要继续追问",
	"停止追问",
	"wrap up",
	"end the interview",
	"finish the interview",
}

func IsWrapupAssistantMessage(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}

	lower := strings.ToLower(content)
	for _, cue := range interviewTurnWrapupCues {
		if strings.Contains(lower, strings.ToLower(cue)) {
			return true
		}
	}
	return false
}

func CountsAsInterviewTurn(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}

	if IsWrapupAssistantMessage(content) {
		return false
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lowerLine := strings.ToLower(line)
		for _, cue := range interviewTurnQuestionCues {
			if strings.Contains(lowerLine, strings.ToLower(cue)) {
				return true
			}
		}
	}

	return false
}

func IsExplicitWrapupRequest(content string) bool {
	content = strings.TrimSpace(strings.ToLower(content))
	if content == "" {
		return false
	}
	for _, cue := range explicitWrapupRequestCues {
		if strings.Contains(content, strings.ToLower(cue)) {
			return true
		}
	}
	return false
}

func hasFutureUserAnswer(messages []protocol.Message, assistantIndex int) bool {
	for next := assistantIndex + 1; next < len(messages); next++ {
		if strings.EqualFold(messages[next].Role, "assistant") {
			return false
		}
		if strings.EqualFold(messages[next].Role, "user") && strings.TrimSpace(messages[next].Content) != "" {
			return true
		}
	}
	return false
}
