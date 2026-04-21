package lifecycle

import (
	"strings"

	"mockinterview/internal/protocol"
)

type SetupInput struct {
	Task    protocol.Task
	Run     protocol.Run
	Request protocol.RunRequest
	Prompt  string
}

type SetupOutcome struct {
	Task   protocol.Task
	Run    protocol.Run
	Prompt string
}

func PrepareRunPrompt(input SetupInput) SetupOutcome {
	task := input.Task
	run := input.Run

	task.Config = task.Config.WithDefaults()
	task.Prompt = strings.TrimSpace(task.Prompt)
	run.Input = strings.TrimSpace(run.Input)

	return SetupOutcome{
		Task: task,
		Run:  run,
		Prompt: firstNonEmptyPreparedPrompt(
			input.Prompt,
			input.Request.Prompt,
			run.Input,
			task.Prompt,
		),
	}
}

type ReductionInput struct {
	Prompt   string
	Memories []protocol.MemoryRecord
	Limit    int
}

type ReductionOutcome struct {
	Prompt  string
	Summary string
}

func BuildReducedPromptContext(input ReductionInput) ReductionOutcome {
	summary := reduceMemories(input.Memories, input.Limit)
	prompt := strings.TrimSpace(input.Prompt)
	if summary != "" {
		prompt = buildReducedPrompt(prompt, summary)
	}
	return ReductionOutcome{
		Prompt:  prompt,
		Summary: summary,
	}
}

type OutputSummaryInput struct {
	Output   string
	MaxLines int
}

func BuildOutputSummary(input OutputSummaryInput) string {
	lines := splitOutputLines(input.Output)
	if len(lines) == 0 {
		return ""
	}

	maxLines := input.MaxLines
	if maxLines <= 0 {
		maxLines = 4
	}

	summary := make([]string, 0, maxLines)
	for _, line := range lines {
		item := cleanBullet(line)
		if item == "" {
			item = strings.TrimSpace(line)
		}
		if item == "" {
			continue
		}
		summary = append(summary, item)
		if len(summary) >= maxLines {
			break
		}
	}
	return strings.Join(summary, " | ")
}

func reduceMemories(memories []protocol.MemoryRecord, limit int) string {
	if len(memories) == 0 {
		return ""
	}
	if limit <= 0 || len(memories) < limit {
		limit = len(memories)
	}
	selected := memories[len(memories)-limit:]
	lines := make([]string, 0, len(selected))
	for _, record := range selected {
		content := strings.TrimSpace(record.Content)
		if content == "" {
			continue
		}
		lines = append(lines, "- "+content)
	}
	return strings.Join(lines, "\n")
}

func buildReducedPrompt(prompt, summary string) string {
	if summary == "" {
		return prompt
	}
	var b strings.Builder
	b.WriteString(strings.TrimSpace(prompt))
	b.WriteString("\n\nRecent compact memory:\n")
	b.WriteString(summary)
	b.WriteString("\nUse this only as lightweight prior context.\n")
	return b.String()
}

func firstNonEmptyPreparedPrompt(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func splitOutputLines(output string) []string {
	parts := strings.Split(output, "\n")
	lines := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		lines = append(lines, part)
	}
	return lines
}

func cleanBullet(line string) string {
	line = strings.TrimSpace(line)
	for _, prefix := range []string{"- ", "* ", "• ", "1. ", "2. ", "3. ", "4. ", "5. "} {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return line
}
