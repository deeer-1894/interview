package middleware

import (
	"strings"

	"mockinterview/internal/protocol"
)

func buildScorecard(output string, rubric protocol.Rubric, style protocol.OutputStyle) protocol.Scorecard {
	card := protocol.Scorecard{
		Title:   rubric.Title,
		Anchors: append([]string(nil), rubric.Anchors...),
	}

	lines := splitOutputLines(output)
	sections := sectionBuckets{
		strengths:    collectSection(lines, []string{"strengths", "优点", "strength", "做得好"}),
		gaps:         collectSection(lines, []string{"gaps", "shortcomings", "不足", "薄弱点"}),
		improvements: collectSection(lines, []string{"improvements", "next steps", "建议", "改进建议"}),
		studyPlan:    collectSection(lines, []string{"study plan", "learning plan", "学习计划"}),
	}

	card.Strengths = sections.strengths
	card.Gaps = sections.gaps
	card.Improvements = sections.improvements

	if style == protocol.OutputInterviewPlusStudy {
		card.StudyPlan = sections.studyPlan
		if len(card.StudyPlan) == 0 {
			card.StudyPlan = append([]string(nil), sections.improvements...)
		}
	}

	if len(card.Improvements) == 0 {
		card.Improvements = fallbackBulletItems(lines, 5)
	}

	return card
}

type sectionBuckets struct {
	strengths    []string
	gaps         []string
	improvements []string
	studyPlan    []string
}

func splitOutputLines(output string) []string {
	raw := strings.Split(output, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

func collectSection(lines []string, headers []string) []string {
	if len(lines) == 0 {
		return nil
	}

	active := false
	items := make([]string, 0, 6)
	for _, line := range lines {
		lower := strings.ToLower(line)
		if matchesHeader(lower, headers) {
			active = true
			continue
		}
		if active && looksLikeHeader(lower) {
			break
		}
		if !active {
			continue
		}
		item := cleanBullet(line)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func matchesHeader(line string, headers []string) bool {
	for _, header := range headers {
		if strings.Contains(line, header) {
			return true
		}
	}
	return false
}

func looksLikeHeader(line string) bool {
	headers := []string{
		"strength", "strengths", "gap", "gaps", "improvement", "improvements",
		"study plan", "learning plan", "score", "scorecard",
		"优点", "不足", "改进", "学习计划", "评分",
	}
	return matchesHeader(line, headers)
}

func cleanBullet(line string) string {
	cleaned := strings.TrimSpace(line)
	cleaned = strings.TrimLeft(cleaned, "-*•0123456789. ")
	return strings.TrimSpace(cleaned)
}

func fallbackBulletItems(lines []string, limit int) []string {
	items := make([]string, 0, limit)
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "-") && !strings.HasPrefix(strings.TrimSpace(line), "*") {
			continue
		}
		item := cleanBullet(line)
		if item == "" {
			continue
		}
		items = append(items, item)
		if len(items) >= limit {
			break
		}
	}
	return items
}
