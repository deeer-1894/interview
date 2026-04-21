package interview

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"mockinterview/internal/interview/adkapp"
	"mockinterview/internal/protocol"
)

type EvaluationResult struct {
	Title           string                    `json:"title"`
	Summary         string                    `json:"summary"`
	OverallScore    int                       `json:"overallScore,omitempty"`
	OverallMaxScore int                       `json:"overallMaxScore,omitempty"`
	DimensionScores []protocol.DimensionScore `json:"dimensionScores"`
	Strengths       []string                  `json:"strengths"`
	Gaps            []string                  `json:"gaps"`
	Improvements    []string                  `json:"improvements"`
	StudyPlan       []string                  `json:"studyPlan,omitempty"`
}

func GenerateScorecard(
	ctx context.Context,
	transcript string,
	cfg protocol.InterviewConfig,
	modelCfg protocol.ModelConfig,
	skill protocol.SkillSpec,
	rubric protocol.Rubric,
) (protocol.Scorecard, error) {
	cfg = cfg.WithDefaults()

	scoreText, err := generateEvaluationText(
		ctx,
		modelCfg,
		buildScoringInstruction(cfg, skill, rubric),
		BuildSharedSessionContext("", "", protocol.RunPhaseEvaluating, protocol.RunInterviewState{}, skill, rubric, transcript),
		buildScoringPrompt(transcript),
	)
	if err != nil {
		return protocol.Scorecard{}, err
	}

	scorecard := buildEvaluationScorecard(scoreText, rubric, protocol.OutputInterviewPlusScore)
	if scorecard.Title == "" {
		scorecard.Title = rubric.Title
	}
	if len(scorecard.Anchors) == 0 {
		scorecard.Anchors = append([]string(nil), rubric.Anchors...)
	}

	if cfg.OutputStyle != protocol.OutputInterviewPlusStudy {
		return scorecard, nil
	}

	studyText, err := generateEvaluationText(
		ctx,
		modelCfg,
		buildStudyPlanInstruction(cfg, skill, rubric),
		BuildSharedSessionContext("", "", protocol.RunPhaseStudyPlan, protocol.RunInterviewState{}, skill, rubric, transcript),
		buildStudyPlanPrompt(transcript, scorecard),
	)
	if err != nil {
		return scorecard, nil
	}

	studyOnly := buildEvaluationScorecard(studyText, rubric, protocol.OutputInterviewPlusStudy)
	if len(studyOnly.StudyPlan) > 0 {
		scorecard.StudyPlan = studyOnly.StudyPlan
	}
	return scorecard, nil
}

func generateEvaluationText(
	ctx context.Context,
	modelCfg protocol.ModelConfig,
	instruction string,
	shared adkapp.SharedSessionContext,
	prompt string,
) (string, error) {
	app, err := adkapp.NewEvaluatorApp(ctx, toModelConfig(modelCfg), instruction, adkapp.Options{
		SharedContext: shared,
	})
	if err != nil {
		return "", protocol.WrapModelError("evaluation", "create_evaluation_agent", false, fmt.Errorf("create evaluation agent: %w", err))
	}

	iter := app.Runner.Query(ctx, prompt)
	output, err := collectAssistantOutput(iter, nil, "evaluation", modelCfg)
	if err != nil {
		return "", protocol.WrapModelError("evaluation", "collect_evaluation_output", isRetryableModelError(err), err)
	}
	return output, nil
}

func buildScoringInstruction(cfg protocol.InterviewConfig, skill protocol.SkillSpec, rubric protocol.Rubric) string {
	var b strings.Builder
	b.WriteString("You are an interview evaluator.\n")
	b.WriteString("Evaluate the interview transcript and produce only a structured scorecard.\n")
	b.WriteString("Be concise, specific, and evidence-based.\n")
	b.WriteString("Do not continue the interview.\n")
	b.WriteString("Do not ask follow-up questions.\n")
	b.WriteString("Return valid JSON only. Do not wrap the JSON in markdown fences.\n")
	b.WriteString("Use a consistent scoring scale: overallScore must be 0-100, and each dimension score should use maxScore=10 unless the rubric explicitly says otherwise.\n")
	b.WriteString("Use this schema exactly:\n")
	b.WriteString(`{"title":"string","summary":"string","overallScore":82,"overallMaxScore":100,"dimensionScores":[{"name":"string","score":8,"maxScore":10,"rationale":"string"}],"strengths":["string"],"gaps":["string"],"improvements":["string"]}` + "\n\n")
	b.WriteString(fmt.Sprintf("Target level: %s\n", cfg.Level))
	b.WriteString(fmt.Sprintf("Focus: %s\n", cfg.Focus))
	b.WriteString(fmt.Sprintf("Mode: %s\n", cfg.Mode))
	if strings.TrimSpace(skill.Name) != "" {
		b.WriteString(fmt.Sprintf("Skill: %s\n", skill.Name))
	}
	if strings.TrimSpace(skill.Description) != "" {
		b.WriteString(fmt.Sprintf("Skill description: %s\n", skill.Description))
	}
	if len(rubric.Anchors) > 0 {
		b.WriteString("Rubric anchors:\n")
		for _, anchor := range rubric.Anchors {
			b.WriteString("- " + strings.TrimSpace(anchor) + "\n")
		}
	}
	return b.String()
}

func buildScoringPrompt(transcript string) string {
	return fmt.Sprintf("Interview transcript:\n%s\n", strings.TrimSpace(transcript))
}

func buildStudyPlanInstruction(cfg protocol.InterviewConfig, skill protocol.SkillSpec, rubric protocol.Rubric) string {
	var b strings.Builder
	b.WriteString("You are an interview coach.\n")
	b.WriteString("Generate only a short learning plan from the provided evaluation.\n")
	b.WriteString("Do not repeat strengths or gaps.\n")
	b.WriteString("Do not continue the interview.\n")
	b.WriteString("Return valid JSON only. Do not wrap the JSON in markdown fences.\n")
	b.WriteString(`Use this schema exactly: {"studyPlan":["string"]}` + "\n\n")
	b.WriteString(fmt.Sprintf("Target level: %s\n", cfg.Level))
	if strings.TrimSpace(skill.Name) != "" {
		b.WriteString(fmt.Sprintf("Skill: %s\n", skill.Name))
	}
	if len(rubric.Anchors) > 0 {
		b.WriteString("Rubric anchors:\n")
		for _, anchor := range rubric.Anchors {
			b.WriteString("- " + strings.TrimSpace(anchor) + "\n")
		}
	}
	return b.String()
}

func buildStudyPlanPrompt(transcript string, scorecard protocol.Scorecard) string {
	var b strings.Builder
	b.WriteString("Interview transcript:\n")
	b.WriteString(strings.TrimSpace(transcript))
	b.WriteString("\n\nCurrent evaluation:\n")
	for _, item := range scorecard.Gaps {
		b.WriteString("- Gap: " + item + "\n")
	}
	for _, item := range scorecard.Improvements {
		b.WriteString("- Improvement: " + item + "\n")
	}
	return b.String()
}

func buildEvaluationScorecard(output string, rubric protocol.Rubric, style protocol.OutputStyle) protocol.Scorecard {
	card := protocol.Scorecard{
		Title:   rubric.Title,
		Anchors: append([]string(nil), rubric.Anchors...),
	}
	if parsed, ok := parseEvaluationResult(output); ok {
		card = enrichScorecardFromText(normalizeEvaluationScorecard(parsed, rubric, style), output)
		if len(card.StudyPlan) == 0 && style == protocol.OutputInterviewPlusStudy && len(card.Improvements) > 0 {
			card.StudyPlan = append([]string(nil), card.Improvements...)
		}
		return card
	}

	lines := splitEvaluationLines(output)
	strengths := collectEvaluationSection(lines, []string{"strengths", "优点", "strength", "做得好", "亮点"})
	gaps := collectEvaluationSection(lines, []string{"gaps", "shortcomings", "不足", "薄弱点", "短板", "待改进"})
	improvements := collectEvaluationSection(lines, []string{"improvements", "next steps", "建议", "改进建议", "短板与改进建议", "下一步建议"})
	studyPlan := collectEvaluationSection(lines, []string{"study plan", "learning plan", "学习计划", "学习计划建议"})

	card.Strengths = strengths
	card.Gaps = gaps
	card.Improvements = improvements

	if style == protocol.OutputInterviewPlusStudy {
		card.StudyPlan = studyPlan
		if len(card.StudyPlan) == 0 {
			card.StudyPlan = append([]string(nil), improvements...)
		}
	}

	if len(card.Improvements) == 0 {
		card.Improvements = fallbackEvaluationBullets(lines, 5)
	}

	return normalizeEvaluationConsistency(enrichScorecardFromText(card, output))
}

func BuildEvaluationScorecardFromOutput(output string, rubric protocol.Rubric, style protocol.OutputStyle) protocol.Scorecard {
	return buildEvaluationScorecard(output, rubric, style)
}

func parseEvaluationResult(output string) (EvaluationResult, bool) {
	candidate := strings.TrimSpace(output)
	if candidate == "" {
		return EvaluationResult{}, false
	}
	candidate = strings.TrimPrefix(candidate, "```json")
	candidate = strings.TrimPrefix(candidate, "```JSON")
	candidate = strings.TrimPrefix(candidate, "```")
	candidate = strings.TrimSuffix(candidate, "```")
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return EvaluationResult{}, false
	}
	if strings.Index(candidate, "{") > 0 || strings.LastIndex(candidate, "}") < len(candidate)-1 {
		start := strings.Index(candidate, "{")
		end := strings.LastIndex(candidate, "}")
		if start >= 0 && end > start {
			candidate = candidate[start : end+1]
		}
	}

	var result EvaluationResult
	if err := json.Unmarshal([]byte(candidate), &result); err != nil {
		return EvaluationResult{}, false
	}
	return result, true
}

func normalizeEvaluationScorecard(result EvaluationResult, rubric protocol.Rubric, style protocol.OutputStyle) protocol.Scorecard {
	card := protocol.Scorecard{
		Title:           firstNonEmptyEvaluationValue(result.Title, rubric.Title),
		Summary:         sanitizeEvaluationText(result.Summary),
		OverallScore:    clampEvaluationScore(result.OverallScore, 0, fallbackMaxScore(result.OverallMaxScore, 100)),
		OverallMaxScore: maxInt(result.OverallMaxScore, 0),
		Anchors:         append([]string(nil), rubric.Anchors...),
		DimensionScores: normalizeDimensionScores(result.DimensionScores),
		Strengths:       normalizeEvaluationItems(result.Strengths),
		Gaps:            normalizeEvaluationItems(result.Gaps),
		Improvements:    normalizeEvaluationItems(result.Improvements),
	}
	if style == protocol.OutputInterviewPlusStudy {
		card.StudyPlan = normalizeEvaluationItems(result.StudyPlan)
	}
	if len(card.Improvements) == 0 {
		card.Improvements = append([]string(nil), card.Gaps...)
	}
	return normalizeEvaluationConsistency(card)
}

func normalizeDimensionScores(values []protocol.DimensionScore) []protocol.DimensionScore {
	if len(values) == 0 {
		return nil
	}
	out := make([]protocol.DimensionScore, 0, len(values))
	for _, value := range values {
		name := sanitizeEvaluationText(value.Name)
		if name == "" {
			continue
		}
		score := value.Score
		maxScore := inferDimensionMaxScore(value.Score, value.MaxScore)
		score = clampEvaluationScore(score, 1, maxScore)
		out = append(out, protocol.DimensionScore{
			Name:      name,
			Score:     score,
			MaxScore:  maxScore,
			Rationale: sanitizeEvaluationText(value.Rationale),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeEvaluationItems(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		cleaned := sanitizeEvaluationText(value)
		if cleaned == "" {
			continue
		}
		key := strings.ToLower(cleaned)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, cleaned)
	}
	return out
}

func firstNonEmptyEvaluationValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeEvaluationConsistency(card protocol.Scorecard) protocol.Scorecard {
	card.Summary = sanitizeEvaluationText(card.Summary)
	card.Strengths = removeEvaluationOverlaps(card.Strengths, card.Gaps, card.Improvements)
	card.Gaps = removeEvaluationOverlaps(card.Gaps, card.Strengths)
	card.StudyPlan = removeEvaluationOverlaps(card.StudyPlan, card.Strengths)
	card.OverallScore, card.OverallMaxScore = normalizeOverallScore(card)
	if strings.TrimSpace(card.Summary) == "" {
		card.Summary = buildEvaluationSummary(card)
	}
	return card
}

var (
	overallScorePattern    = regexp.MustCompile(`(?i)(?:总分|综合评分|最终评分|overall score)[^0-9]{0,24}\(?\s*(\d{1,3}(?:\.\d+)?)\s*/\s*(\d{1,3})\)?`)
	dimensionScorePattern  = regexp.MustCompile(`^[\-\*\d\.\s]*\*{0,2}([^:*：|]+?)\*{0,2}\s*[:：]\s*(\d{1,3}(?:\.\d+)?)\s*/\s*(\d{1,3})(.*)$`)
	markdownTableScoreLine = regexp.MustCompile(`^\|\s*([^|]+?)\s*\|\s*(\d{1,3}(?:\.\d+)?)\s*/\s*(\d{1,3})\s*\|`)
)

func enrichScorecardFromText(card protocol.Scorecard, output string) protocol.Scorecard {
	output = strings.TrimSpace(output)
	if output == "" {
		return card
	}

	if card.OverallScore == 0 {
		if score, maxScore, ok := parseOverallScoreFromText(output); ok {
			card.OverallScore = score
			card.OverallMaxScore = maxScore
		}
	}
	if len(card.DimensionScores) == 0 {
		card.DimensionScores = parseDimensionScoresFromText(output)
	}
	return card
}

func parseOverallScoreFromText(output string) (int, int, bool) {
	match := overallScorePattern.FindStringSubmatch(output)
	if len(match) != 3 {
		return 0, 0, false
	}
	score, err := parseEvaluationScoreNumber(match[1])
	if err != nil {
		return 0, 0, false
	}
	maxScore, err := strconv.Atoi(match[2])
	if err != nil || maxScore <= 0 {
		return 0, 0, false
	}
	return clampEvaluationScore(score, 0, maxScore), maxScore, true
}

func parseDimensionScoresFromText(output string) []protocol.DimensionScore {
	lines := splitEvaluationLines(output)
	out := make([]protocol.DimensionScore, 0, len(lines))
	for _, line := range lines {
		if score, ok := parseDimensionScoreLine(line); ok {
			out = append(out, score)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseDimensionScoreLine(line string) (protocol.DimensionScore, bool) {
	if match := dimensionScorePattern.FindStringSubmatch(strings.TrimSpace(line)); len(match) == 5 {
		name := sanitizeEvaluationText(match[1])
		if isOverallScoreLabel(name) {
			return protocol.DimensionScore{}, false
		}
		score, maxScore, ok := normalizeParsedDimensionScore(match[2], match[3])
		if !ok {
			return protocol.DimensionScore{}, false
		}
		return protocol.DimensionScore{
			Name:      name,
			Score:     score,
			MaxScore:  maxScore,
			Rationale: sanitizeEvaluationText(match[4]),
		}, name != ""
	}
	if match := markdownTableScoreLine.FindStringSubmatch(strings.TrimSpace(line)); len(match) == 4 {
		name := sanitizeEvaluationText(strings.Trim(strings.TrimSpace(match[1]), "*"))
		if isOverallScoreLabel(name) {
			return protocol.DimensionScore{}, false
		}
		score, maxScore, ok := normalizeParsedDimensionScore(match[2], match[3])
		if !ok {
			return protocol.DimensionScore{}, false
		}
		return protocol.DimensionScore{
			Name:     name,
			Score:    score,
			MaxScore: maxScore,
		}, name != ""
	}
	return protocol.DimensionScore{}, false
}

func normalizeParsedDimensionScore(scoreText string, maxText string) (int, int, bool) {
	rawScore, err := strconv.ParseFloat(strings.TrimSpace(scoreText), 64)
	if err != nil {
		return 0, 0, false
	}
	maxScore, err := strconv.Atoi(strings.TrimSpace(maxText))
	if err != nil || maxScore <= 0 {
		return 0, 0, false
	}
	if maxScore == 5 {
		return clampEvaluationScore(int(rawScore/float64(maxScore)*10+0.5), 1, 10), 10, true
	}
	return clampEvaluationScore(int(rawScore+0.5), 1, maxScore), maxScore, true
}

func parseEvaluationScoreNumber(value string) (int, error) {
	value = strings.TrimSpace(value)
	if strings.Contains(value, ".") {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, err
		}
		return int(parsed + 0.5), nil
	}
	return strconv.Atoi(value)
}

func isOverallScoreLabel(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "总分", "最终总分", "综合评分", "综合得分", "最终评分", "overall score":
		return true
	default:
		return false
	}
}

func normalizeOverallScore(card protocol.Scorecard) (int, int) {
	maxScore := card.OverallMaxScore
	if card.OverallScore > 0 {
		if maxScore <= 0 {
			maxScore = 100
		}
		return clampEvaluationScore(card.OverallScore, 0, maxScore), maxScore
	}
	if len(card.DimensionScores) == 0 {
		return 0, 0
	}
	total := 0
	for _, dimension := range card.DimensionScores {
		scale := inferDimensionMaxScore(dimension.Score, dimension.MaxScore)
		total += int(float64(clampEvaluationScore(dimension.Score, 0, scale)) / float64(scale) * 100)
	}
	return clampEvaluationScore(total/len(card.DimensionScores), 0, 100), 100
}

func inferDimensionMaxScore(score int, explicit int) int {
	if explicit > 0 {
		return explicit
	}
	if score > 5 {
		return 10
	}
	return 5
}

func clampEvaluationScore(value, minValue, maxValue int) int {
	if maxValue <= 0 {
		maxValue = minValue
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func fallbackMaxScore(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func removeEvaluationOverlaps(primary []string, conflicts ...[]string) []string {
	if len(primary) == 0 {
		return nil
	}

	blocked := make(map[string]struct{})
	for _, group := range conflicts {
		for _, item := range group {
			key := normalizedEvaluationKey(item)
			if key == "" {
				continue
			}
			blocked[key] = struct{}{}
		}
	}

	filtered := make([]string, 0, len(primary))
	for _, item := range primary {
		key := normalizedEvaluationKey(item)
		if key == "" {
			continue
		}
		if _, ok := blocked[key]; ok {
			continue
		}
		filtered = append(filtered, item)
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func normalizedEvaluationKey(value string) string {
	cleaned := strings.ToLower(sanitizeEvaluationText(value))
	cleaned = strings.ReplaceAll(cleaned, "，", ",")
	cleaned = strings.ReplaceAll(cleaned, "。", "")
	cleaned = strings.ReplaceAll(cleaned, ".", "")
	return strings.TrimSpace(cleaned)
}

func buildEvaluationSummary(card protocol.Scorecard) string {
	strength := firstEvaluationItem(card.Strengths)
	gap := firstEvaluationItem(card.Gaps)
	improvement := firstEvaluationItem(card.Improvements)

	switch {
	case strength != "" && gap != "":
		return fmt.Sprintf("优势是%s，当前最需要补齐的是%s。", strength, gap)
	case gap != "" && improvement != "":
		return fmt.Sprintf("当前主要短板是%s，建议优先%s。", gap, improvement)
	case strength != "":
		return fmt.Sprintf("本场表现较稳的部分是%s。", strength)
	case gap != "":
		return fmt.Sprintf("本场最明显的短板是%s。", gap)
	case improvement != "":
		return fmt.Sprintf("建议下一步优先%s。", improvement)
	default:
		return ""
	}
}

func firstEvaluationItem(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return sanitizeEvaluationText(values[0])
}

func splitEvaluationLines(output string) []string {
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

func collectEvaluationSection(lines []string, headers []string) []string {
	if len(lines) == 0 {
		return nil
	}
	active := false
	items := make([]string, 0, 6)
	for _, line := range lines {
		lower := strings.ToLower(line)
		if matchesEvaluationHeader(lower, headers) {
			active = true
			continue
		}
		if active && looksLikeEvaluationHeader(lower) {
			break
		}
		if !active {
			continue
		}
		item := cleanEvaluationBullet(line)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func matchesEvaluationHeader(line string, headers []string) bool {
	for _, header := range headers {
		if strings.Contains(line, header) {
			return true
		}
	}
	return false
}

func looksLikeEvaluationHeader(line string) bool {
	headers := []string{
		"strength", "strengths", "gap", "gaps", "improvement", "improvements",
		"study plan", "learning plan", "score", "scorecard",
		"优点", "不足", "改进", "学习计划", "评分", "亮点", "短板", "待改进",
		"短板与改进建议", "下一步建议", "维度评分", "综合评分", "综合得分",
		"面试策略总结", "策略建议", "追问树分析", "追问树总结", "画像分析", "候选人画像",
		"阶段推进评估", "阶段过程总结", "建议后续阶段",
	}
	return matchesEvaluationHeader(line, headers)
}

func cleanEvaluationBullet(line string) string {
	cleaned := strings.TrimSpace(line)
	cleaned = strings.TrimLeft(cleaned, "#")
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.TrimLeft(cleaned, "-*•0123456789. ")
	return sanitizeEvaluationText(cleaned)
}

func fallbackEvaluationBullets(lines []string, limit int) []string {
	items := make([]string, 0, limit)
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "-") && !strings.HasPrefix(strings.TrimSpace(line), "*") {
			continue
		}
		item := cleanEvaluationBullet(line)
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

func sanitizeEvaluationText(value string) string {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return ""
	}

	cleaned = strings.TrimLeft(cleaned, "#")
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.TrimLeft(cleaned, "-*•0123456789. ")
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.ReplaceAll(cleaned, "**", "")
	cleaned = strings.ReplaceAll(cleaned, "__", "")
	cleaned = strings.TrimSpace(cleaned)

	for {
		next := strings.TrimSpace(cleaned)
		next = strings.Trim(next, "`")
		next = strings.Trim(next, "\"'")
		next = strings.TrimSpace(next)
		next = strings.Trim(next, "[]{}")
		next = strings.TrimSpace(next)
		next = strings.TrimRight(next, ",，;；")
		next = strings.TrimSpace(next)
		if next == cleaned {
			break
		}
		cleaned = next
	}

	switch strings.ToLower(cleaned) {
	case "", "```", "json", "markdown":
		return ""
	}
	if isEvaluationNoiseItem(cleaned) {
		return ""
	}
	if !containsMeaningfulEvaluationText(cleaned) {
		return ""
	}
	return cleaned
}

func isEvaluationNoiseItem(value string) bool {
	normalized := strings.ToLower(strings.Trim(strings.TrimSpace(value), "：:"))
	switch normalized {
	case "面试策略总结", "策略建议", "追问树分析", "追问树总结", "画像分析", "候选人画像", "阶段推进评估", "阶段过程总结", "建议后续阶段",
		"strategy summary", "trace analysis", "profile analysis", "phase review":
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

func containsMeaningfulEvaluationText(value string) bool {
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return true
		}
	}
	return false
}
