package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"mockinterview/internal/interview/session"
)

const baseInterviewInstruction = `你是 OfferBot 的技术面试官。根据结构化上下文进行真实面试；不要编造简历事实；一次只输出一个自然的面试官回应。`

func BuildModelInput(ctx context.Context, instruction string, input *adk.AgentInput) ([]adk.Message, error) {
	messages := make([]adk.Message, 0, len(input.Messages)+4)
	messages = append(messages, systemMessage(joinInstruction(baseInterviewInstruction, instruction)))

	if turn, ok := TurnContextFrom(ctx); ok {
		messages = append(messages, systemMessage(renderInterviewContext(turn)))
		for _, msg := range turn.RecentMessages {
			if converted, ok := convertStoredMessage(msg); ok {
				messages = append(messages, converted)
			}
		}
	}

	messages = append(messages, input.Messages...)
	return messages, nil
}

func joinInstruction(base string, extra string) string {
	if extra == "" {
		return base
	}
	return base + "\n" + extra
}

func renderInterviewContext(turn session.TurnContext) string {
	var b strings.Builder
	b.WriteString("当前面试上下文，仅供你决定下一句面试官回应，不要向候选人复述上下文本身。\n")
	writeLine(&b, "面试职位", turn.Role)
	writeLine(&b, "候选级别", turn.Level)
	writeLine(&b, "面试模式", turn.Mode)
	if turn.Round > 0 {
		writeLine(&b, "当前进度", fmt.Sprintf("第 %d 轮", turn.Round))
	}
	if len(turn.ResumeSkills) > 0 {
		writeLine(&b, "候选人技术栈", strings.Join(turn.ResumeSkills, " / "))
	}
	if strings.TrimSpace(turn.ResumeSummary) != "" {
		writeLine(&b, "候选人概览", turn.ResumeSummary)
	}
	if strings.TrimSpace(turn.ResumeRawText) != "" {
		writeLine(&b, "候选人简历", compactText(turn.ResumeRawText, 1800))
	}
	if strings.TrimSpace(turn.ActiveProject.Name) != "" {
		writeLine(&b, "当前聚焦项目", turn.ActiveProject.Name)
		writeLine(&b, "项目领域", turn.ActiveProject.Domain)
		writeLine(&b, "项目简介", turn.ActiveProject.Summary)
		if len(turn.ActiveProject.TechStack) > 0 {
			writeLine(&b, "项目技术栈", strings.Join(turn.ActiveProject.TechStack, " / "))
		}
		if len(turn.ActiveProject.Claims) > 0 {
			writeLine(&b, "候选人声称负责", strings.Join(turn.ActiveProject.Claims, "；"))
		}
	}
	if len(turn.OtherProjects) > 0 {
		briefs := make([]string, 0, len(turn.OtherProjects))
		for _, project := range turn.OtherProjects {
			brief := project.Name
			if project.Domain != "" {
				brief += "（" + project.Domain + "）"
			}
			briefs = append(briefs, brief)
		}
		writeLine(&b, "其他可切换项目", strings.Join(briefs, "；"))
	}
	if len(turn.CoveredTopics) > 0 {
		writeLine(&b, "已覆盖方向", strings.Join(turn.CoveredTopics, " / "))
	}
	if strings.TrimSpace(turn.LastAnswer) != "" {
		writeLine(&b, "候选人上一轮回答", compactText(turn.LastAnswer, 700))
	}
	b.WriteString("\n面试要求：开场要自然，不要说“读取上下文”；不要输出 JSON、字段名、ID、邮箱、系统内部信息；一次只问一个主问题，可带一个简短追问点。\n")
	return b.String()
}

func writeLine(b *strings.Builder, label string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	b.WriteString(label)
	b.WriteString("：")
	b.WriteString(value)
	b.WriteByte('\n')
}

func compactText(text string, limit int) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	text = strings.Join(strings.Fields(text), " ")
	if limit > 0 && len([]rune(text)) > limit {
		runes := []rune(text)
		return string(runes[:limit]) + "..."
	}
	return text
}

func convertStoredMessage(msg session.Message) (adk.Message, bool) {
	switch msg.Role {
	case session.RoleUser:
		return schema.UserMessage(msg.Content), true
	case session.RoleAssistant:
		return schema.AssistantMessage(msg.Content, nil), true
	default:
		return nil, false
	}
}
