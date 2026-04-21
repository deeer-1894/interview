package interview

import (
	"fmt"
	"strings"

	"mockinterview/internal/protocol"
)

type SkillPack struct {
	Name                 string   `json:"name"`
	Description          string   `json:"description,omitempty"`
	Version              string   `json:"version,omitempty"`
	InstallSource        string   `json:"installSource,omitempty"`
	SourceURL            string   `json:"sourceUrl,omitempty"`
	ComposedOf           []string `json:"composedOf,omitempty"`
	CapabilityBoundaries []string `json:"capabilityBoundaries,omitempty"`
	Rating               float64  `json:"rating,omitempty"`
	RatingCount          int      `json:"ratingCount,omitempty"`
	FocusAreas           []string `json:"focusAreas,omitempty"`
	SampleQs             []string `json:"sampleQuestions,omitempty"`
	FollowUps            []string `json:"followUps,omitempty"`
	Scenarios            []string `json:"scenarios,omitempty"`
	Adversarial          []string `json:"adversarial,omitempty"`
	Pressure             []string `json:"pressure,omitempty"`
	Anchors              []string `json:"anchors,omitempty"`
}

func LoadSkillPack(name string) (SkillPack, error) {
	doc, err := loadResolvedSkillDocument(name, nil)
	if err != nil {
		return SkillPack{}, err
	}
	return ParseSkillPack(doc)
}

func ParseSkillPack(doc SkillDocument) (SkillPack, error) {
	pack := SkillPack{
		Name:                 strings.TrimSpace(doc.Name),
		Description:          strings.TrimSpace(doc.Description),
		Version:              strings.TrimSpace(doc.Version),
		InstallSource:        strings.TrimSpace(doc.InstallSource),
		SourceURL:            strings.TrimSpace(doc.SourceURL),
		ComposedOf:           normalizeBulletList(doc.ComposedOf),
		CapabilityBoundaries: normalizeBulletList(doc.CapabilityBoundaries),
		Rating:               doc.Rating,
		RatingCount:          doc.RatingCount,
		FocusAreas:           normalizeBulletList(doc.FocusAreas),
		SampleQs:             normalizeBulletList(doc.SampleQuestions),
		FollowUps:            normalizeBulletList(doc.FollowUps),
		Scenarios:            normalizeBulletList(doc.Scenarios),
		Adversarial:          normalizeBulletList(doc.Adversarial),
		Pressure:             normalizeBulletList(doc.Pressure),
		Anchors:              normalizeBulletList(doc.ScoringAnchors),
	}
	return pack.WithDefaults(), nil
}

func (p SkillPack) WithDefaults() SkillPack {
	if strings.TrimSpace(p.Name) == "" {
		p.Name = "agent-interview-sim"
	}
	if len(p.FocusAreas) == 0 {
		p.FocusAreas = []string{"system design", "failure handling", "engineering tradeoffs"}
	}
	if len(p.Scenarios) == 0 {
		p.Scenarios = []string{
			"A critical dependency becomes slow and intermittent during peak traffic.",
			"Production traffic grows sharply within minutes while the workflow remains business critical.",
		}
	}
	if len(p.Adversarial) == 0 {
		p.Adversarial = []string{
			"What is the weakest assumption in your design?",
			"If your primary mitigation fails, what is your fallback path?",
		}
	}
	if len(p.Pressure) == 0 {
		p.Pressure = []string{
			"You have two minutes left. Give the final tradeoff and recommendation directly.",
			"Do not restate the background. Give the shortest production-ready answer.",
		}
	}
	return p
}

func (p SkillPack) ToSkillSpec() protocol.SkillSpec {
	return protocol.SkillSpec{
		Name:            p.Name,
		Description:     p.Description,
		Version:         p.Version,
		InstallSource:   p.InstallSource,
		SourceURL:       p.SourceURL,
		ComposedOf:      append([]string(nil), p.ComposedOf...),
		Rating:          p.Rating,
		RatingCount:     p.RatingCount,
		FocusAreas:      append([]string(nil), p.FocusAreas...),
		SampleQuestions: append([]string(nil), p.SampleQs...),
		FollowUps:       append([]string(nil), p.FollowUps...),
		Scenarios:       append([]string(nil), p.Scenarios...),
		Adversarial:     append([]string(nil), p.Adversarial...),
		Pressure:        append([]string(nil), p.Pressure...),
		ScoringAnchors:  append([]string(nil), p.Anchors...),
	}
}

func loadResolvedSkillDocument(name string, visited map[string]struct{}) (SkillDocument, error) {
	name = sanitizeSkillName(name)
	if name == "" {
		return SkillDocument{}, fmt.Errorf("skill name is required")
	}
	if visited == nil {
		visited = make(map[string]struct{})
	}
	if _, exists := visited[name]; exists {
		return SkillDocument{}, fmt.Errorf("skill composition cycle detected at %q", name)
	}
	visited[name] = struct{}{}
	defer delete(visited, name)

	doc, err := LoadSkillDocument(name)
	if err != nil {
		return SkillDocument{}, err
	}
	return mergeSkillDependencies(doc, visited)
}

func mergeSkillDependencies(doc SkillDocument, visited map[string]struct{}) (SkillDocument, error) {
	merged := SkillDocument{}
	for _, dependency := range normalizeBulletList(doc.ComposedOf) {
		dependencyDoc, err := loadResolvedSkillDocument(dependency, visited)
		if err != nil {
			return SkillDocument{}, err
		}
		merged = mergeSkillDocuments(merged, dependencyDoc)
	}
	merged = mergeSkillDocuments(merged, doc)
	return merged, nil
}

func mergeSkillDocuments(base SkillDocument, overlay SkillDocument) SkillDocument {
	return SkillDocument{
		Name:                 firstNonEmptySkillValue(strings.TrimSpace(overlay.Name), strings.TrimSpace(base.Name)),
		Description:          firstNonEmptySkillValue(strings.TrimSpace(overlay.Description), strings.TrimSpace(base.Description)),
		Version:              firstNonEmptySkillValue(strings.TrimSpace(overlay.Version), strings.TrimSpace(base.Version)),
		FocusAreas:           mergePriorityValues(normalizeBulletList(overlay.FocusAreas), normalizeBulletList(base.FocusAreas)),
		ComposedOf:           normalizeBulletList(append(append([]string(nil), base.ComposedOf...), overlay.ComposedOf...)),
		CapabilityBoundaries: mergePriorityValues(normalizeBulletList(overlay.CapabilityBoundaries), normalizeBulletList(base.CapabilityBoundaries)),
		SampleQuestions:      mergePriorityValues(normalizeBulletList(overlay.SampleQuestions), normalizeBulletList(base.SampleQuestions)),
		FollowUps:            mergePriorityValues(normalizeBulletList(overlay.FollowUps), normalizeBulletList(base.FollowUps)),
		Scenarios:            mergePriorityValues(normalizeBulletList(overlay.Scenarios), normalizeBulletList(base.Scenarios)),
		Adversarial:          mergePriorityValues(normalizeBulletList(overlay.Adversarial), normalizeBulletList(base.Adversarial)),
		Pressure:             mergePriorityValues(normalizeBulletList(overlay.Pressure), normalizeBulletList(base.Pressure)),
		ScoringAnchors:       mergePriorityValues(normalizeBulletList(overlay.ScoringAnchors), normalizeBulletList(base.ScoringAnchors)),
		InstallSource:        firstNonEmptySkillValue(strings.TrimSpace(overlay.InstallSource), strings.TrimSpace(base.InstallSource)),
		SourceURL:            firstNonEmptySkillValue(strings.TrimSpace(overlay.SourceURL), strings.TrimSpace(base.SourceURL)),
		Rating:               firstNonZeroFloat(overlay.Rating, base.Rating),
		RatingCount:          firstNonZeroInt(overlay.RatingCount, base.RatingCount),
		Content:              firstNonEmptySkillValue(strings.TrimSpace(overlay.Content), strings.TrimSpace(base.Content)),
	}
}

func firstNonZeroFloat(values ...float64) float64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func firstNonZeroInt(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func ApplyFocusConstraints(spec protocol.SkillSpec, focusAreas []string) protocol.SkillSpec {
	priority := normalizeBulletList(focusAreas)
	if len(priority) == 0 {
		return spec
	}

	spec.FocusAreas = mergePriorityValues(priority, spec.FocusAreas)
	spec.SampleQuestions = mergePriorityValues(buildFocusQuestions(priority), spec.SampleQuestions)
	spec.FollowUps = mergePriorityValues(buildFocusFollowUps(priority), spec.FollowUps)
	spec.Scenarios = mergePriorityValues(buildFocusScenarios(priority), spec.Scenarios)
	return spec
}

func ConstrainSkillSpecForDecision(spec protocol.SkillSpec, focusAreas []string, mode protocol.InterviewMode) protocol.SkillSpec {
	priority := normalizeBulletList(focusAreas)
	if len(priority) == 0 {
		return spec
	}

	spec.FocusAreas = prioritizeByFocus(spec.FocusAreas, priority, priority, 4)
	spec.SampleQuestions = prioritizeByFocus(spec.SampleQuestions, priority, buildFocusQuestions(priority), 4)
	spec.FollowUps = prioritizeByFocus(spec.FollowUps, priority, buildFocusFollowUps(priority), 4)
	spec.Scenarios = prioritizeByFocus(spec.Scenarios, priority, buildFocusScenarios(priority), 3)

	switch NormalizeInterviewMode(string(mode)) {
	case protocol.ModeStress:
		spec.Pressure = prioritizeByFocus(spec.Pressure, priority, []string{
			"Keep the candidate under time pressure and force a concise tradeoff.",
			"Do not let the candidate reset the topic; require a fast production recommendation.",
		}, 3)
	case protocol.ModeWeaknessFocused:
		spec.Adversarial = prioritizeByFocus(spec.Adversarial, priority, []string{
			"Stay on the weak area until the candidate provides a concrete mechanism and tradeoff.",
		}, 3)
	case protocol.ModeSystemDesign:
		spec.Scenarios = prioritizeByFocus(spec.Scenarios, mergePriorityValues(priority, []string{"system design", "reliability", "observability"}), buildFocusScenarios(priority), 4)
	case protocol.ModeResumeDeepDive:
		spec.FollowUps = prioritizeByFocus(spec.FollowUps, mergePriorityValues(priority, []string{"ownership", "resume evidence", "impact"}), buildFocusFollowUps(priority), 4)
	}

	return spec
}

func buildFocusQuestions(focusAreas []string) []string {
	out := make([]string, 0, len(focusAreas))
	for _, area := range focusAreas {
		out = append(out, fmt.Sprintf("Center this round on %s and ask for a concrete implementation path.", area))
	}
	return out
}

func buildFocusFollowUps(focusAreas []string) []string {
	out := make([]string, 0, len(focusAreas))
	for _, area := range focusAreas {
		out = append(out, fmt.Sprintf("If the candidate stays high level on %s, force them into tradeoffs, failure modes, and concrete mechanisms.", area))
	}
	return out
}

func buildFocusScenarios(focusAreas []string) []string {
	out := make([]string, 0, len(focusAreas))
	for _, area := range focusAreas {
		out = append(out, fmt.Sprintf("Inject a scenario that stresses %s under production pressure.", area))
	}
	return out
}

func mergePriorityValues(priority []string, existing []string) []string {
	out := make([]string, 0, len(priority)+len(existing))
	seen := map[string]struct{}{}
	appendUnique := func(values []string) {
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			key := strings.ToLower(value)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, value)
		}
	}
	appendUnique(priority)
	appendUnique(existing)
	return out
}

func prioritizeByFocus(existing []string, focusAreas []string, fallback []string, limit int) []string {
	if limit <= 0 {
		limit = len(existing)
	}
	matched := make([]string, 0, len(existing))
	unmatched := make([]string, 0, len(existing))
	for _, item := range normalizeBulletList(existing) {
		if matchesAnyFocus(item, focusAreas) {
			matched = append(matched, item)
			continue
		}
		unmatched = append(unmatched, item)
	}
	out := mergePriorityValues(matched, nil)
	if len(out) == 0 {
		out = mergePriorityValues(normalizeBulletList(fallback), out)
	}
	if len(out) < limit {
		out = mergePriorityValues(out, unmatched)
	}
	if len(out) > limit {
		return out[:limit]
	}
	return out
}

func matchesAnyFocus(value string, focusAreas []string) bool {
	value = normalizeFocusText(value)
	for _, focus := range focusAreas {
		normalized := normalizeFocusText(focus)
		if normalized == "" {
			continue
		}
		if strings.Contains(value, normalized) || strings.Contains(normalized, value) {
			return true
		}
		for _, token := range strings.Fields(normalized) {
			if len(token) < 3 {
				continue
			}
			if strings.Contains(value, token) {
				return true
			}
		}
	}
	return false
}

func normalizeFocusText(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	replacer := strings.NewReplacer("_", " ", "-", " ", "/", " ", ":", " ", ",", " ", ".", " ")
	value = replacer.Replace(value)
	return strings.Join(strings.Fields(value), " ")
}

func normalizeBulletList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}
