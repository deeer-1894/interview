package interview

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type SkillMetadata struct {
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	Version              string   `json:"version,omitempty"`
	FocusAreas           []string `json:"focusAreas,omitempty"`
	ComposedOf           []string `json:"composedOf,omitempty"`
	CapabilityBoundaries []string `json:"capabilityBoundaries,omitempty"`
	InstallSource        string   `json:"installSource,omitempty"`
	SourceURL            string   `json:"sourceUrl,omitempty"`
	Rating               float64  `json:"rating,omitempty"`
	RatingCount          int      `json:"ratingCount,omitempty"`
}

type SkillDocument struct {
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	Version              string   `json:"version,omitempty"`
	FocusAreas           []string `json:"focusAreas,omitempty"`
	ComposedOf           []string `json:"composedOf,omitempty"`
	CapabilityBoundaries []string `json:"capabilityBoundaries,omitempty"`
	SampleQuestions      []string `json:"sampleQuestions,omitempty"`
	FollowUps            []string `json:"followUps,omitempty"`
	Scenarios            []string `json:"scenarios,omitempty"`
	Adversarial          []string `json:"adversarial,omitempty"`
	Pressure             []string `json:"pressure,omitempty"`
	ScoringAnchors       []string `json:"scoringAnchors,omitempty"`
	InstallSource        string   `json:"installSource,omitempty"`
	SourceURL            string   `json:"sourceUrl,omitempty"`
	Rating               float64  `json:"rating,omitempty"`
	RatingCount          int      `json:"ratingCount,omitempty"`
	Content              string   `json:"content"`
}

type SkillSourceBoundary struct {
	StorageScope string   `json:"storageScope"`
	Supports     []string `json:"supports"`
	Defers       []string `json:"defers"`
}

func CurrentSkillSourceBoundary() SkillSourceBoundary {
	return SkillSourceBoundary{
		StorageScope: "local skill documents rooted at project /skills or CODEX_HOME",
		Supports: []string{
			"skill metadata extraction and persistence",
			"skill version and install source fields",
			"skill composition via composedOf",
			"frontmatter-driven focus/scenario/follow-up content",
		},
		Defers: []string{
			"remote marketplace discovery and ranking",
			"installation workflow orchestration",
			"runtime sandbox or permission policy for third-party skills",
		},
	}
}

func SkillsBaseDir() string {
	if dir := projectSkillsDir(); dir != "" {
		return dir
	}
	if codexHome := strings.TrimSpace(os.Getenv("CODEX_HOME")); codexHome != "" {
		return filepath.Join(codexHome, "skills")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".codex", "skills")
}

func projectSkillsDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	dir := filepath.Join(filepath.Dir(file), "..", "..", "skills")
	dir = filepath.Clean(dir)
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return ""
	}
	return dir
}

func ResolveSkillName(cfg InterviewConfig) string {
	cfg = cfg.WithDefaults()
	if skill := strings.TrimSpace(cfg.Skill); skill != "" {
		return skill
	}
	return "agent-interview-sim"
}

func ResolveInterviewFocusAreas(cfg InterviewConfig) []string {
	cfg = cfg.WithDefaults()
	return normalizeBulletList(append([]string{cfg.Focus}, cfg.SkillFocuses...))
}

func LoadSkillMetadata(name string) (SkillMetadata, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return SkillMetadata{}, fmt.Errorf("skill name is required")
	}

	path := filepath.Join(SkillsBaseDir(), name, "SKILL.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return SkillMetadata{}, fmt.Errorf("read skill %q: %w", name, err)
	}

	fm, err := parseSkillFrontMatter(string(content))
	if err != nil {
		return SkillMetadata{}, fmt.Errorf("parse skill %q: %w", name, err)
	}

	return SkillMetadata{
		Name:                 firstNonEmptySkillValue(strings.TrimSpace(fm.Name), name),
		Description:          strings.TrimSpace(fm.Description),
		Version:              strings.TrimSpace(fm.Version),
		FocusAreas:           normalizeBulletList(fm.FocusAreas),
		ComposedOf:           normalizeBulletList(fm.ComposedOf),
		CapabilityBoundaries: normalizeBulletList(fm.CapabilityBoundaries),
		InstallSource:        strings.TrimSpace(fm.InstallSource),
		SourceURL:            strings.TrimSpace(fm.SourceURL),
		Rating:               fm.Rating,
		RatingCount:          fm.RatingCount,
	}, nil
}

func ListInterviewSkills() ([]SkillMetadata, error) {
	root := SkillsBaseDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read skills dir: %w", err)
	}

	skills := make([]SkillMetadata, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		meta, loadErr := LoadSkillMetadata(entry.Name())
		if loadErr != nil {
			continue
		}
		skills = append(skills, meta)
	}
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})
	return skills, nil
}

func LoadSkillDocument(name string) (SkillDocument, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return SkillDocument{}, fmt.Errorf("skill name is required")
	}

	path := filepath.Join(SkillsBaseDir(), name, "SKILL.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return SkillDocument{}, fmt.Errorf("read skill %q: %w", name, err)
	}

	fm, body, err := parseSkillDocument(string(content))
	if err != nil {
		return SkillDocument{}, fmt.Errorf("parse skill %q: %w", name, err)
	}

	return SkillDocument{
		Name:                 firstNonEmptySkillValue(strings.TrimSpace(fm.Name), name),
		Description:          strings.TrimSpace(fm.Description),
		Version:              strings.TrimSpace(fm.Version),
		FocusAreas:           normalizeBulletList(fm.FocusAreas),
		ComposedOf:           normalizeBulletList(fm.ComposedOf),
		CapabilityBoundaries: normalizeBulletList(fm.CapabilityBoundaries),
		SampleQuestions:      normalizeBulletList(fm.SampleQuestions),
		FollowUps:            normalizeBulletList(fm.FollowUps),
		Scenarios:            normalizeBulletList(fm.Scenarios),
		Adversarial:          normalizeBulletList(fm.Adversarial),
		Pressure:             normalizeBulletList(fm.Pressure),
		ScoringAnchors:       normalizeBulletList(fm.ScoringAnchors),
		InstallSource:        strings.TrimSpace(fm.InstallSource),
		SourceURL:            strings.TrimSpace(fm.SourceURL),
		Rating:               fm.Rating,
		RatingCount:          fm.RatingCount,
		Content:              strings.TrimSpace(body),
	}, nil
}

func SaveSkillDocument(doc SkillDocument) (SkillMetadata, error) {
	name := sanitizeSkillName(doc.Name)
	if name == "" {
		return SkillMetadata{}, fmt.Errorf("skill name is required")
	}

	dir := filepath.Join(SkillsBaseDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return SkillMetadata{}, fmt.Errorf("create skill dir: %w", err)
	}

	markdown, err := renderSkillDocument(SkillDocument{
		Name:                 name,
		Description:          strings.TrimSpace(doc.Description),
		Version:              strings.TrimSpace(doc.Version),
		FocusAreas:           normalizeBulletList(doc.FocusAreas),
		ComposedOf:           normalizeBulletList(doc.ComposedOf),
		CapabilityBoundaries: normalizeBulletList(doc.CapabilityBoundaries),
		SampleQuestions:      normalizeBulletList(doc.SampleQuestions),
		FollowUps:            normalizeBulletList(doc.FollowUps),
		Scenarios:            normalizeBulletList(doc.Scenarios),
		Adversarial:          normalizeBulletList(doc.Adversarial),
		Pressure:             normalizeBulletList(doc.Pressure),
		ScoringAnchors:       normalizeBulletList(doc.ScoringAnchors),
		InstallSource:        strings.TrimSpace(doc.InstallSource),
		SourceURL:            strings.TrimSpace(doc.SourceURL),
		Rating:               doc.Rating,
		RatingCount:          doc.RatingCount,
		Content:              strings.TrimSpace(doc.Content),
	})
	if err != nil {
		return SkillMetadata{}, err
	}

	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(markdown), 0o644); err != nil {
		return SkillMetadata{}, fmt.Errorf("write skill file: %w", err)
	}

	return SkillMetadata{
		Name:                 name,
		Description:          strings.TrimSpace(doc.Description),
		Version:              strings.TrimSpace(doc.Version),
		FocusAreas:           normalizeBulletList(doc.FocusAreas),
		ComposedOf:           normalizeBulletList(doc.ComposedOf),
		CapabilityBoundaries: normalizeBulletList(doc.CapabilityBoundaries),
		InstallSource:        strings.TrimSpace(doc.InstallSource),
		SourceURL:            strings.TrimSpace(doc.SourceURL),
		Rating:               doc.Rating,
		RatingCount:          doc.RatingCount,
	}, nil
}

func ImportSkillFile(filename string, data []byte) (SkillMetadata, error) {
	filename = strings.ToLower(strings.TrimSpace(filename))
	switch {
	case strings.HasSuffix(filename, ".zip"), strings.HasSuffix(filename, ".skill"):
		return importSkillArchive(data)
	default:
		return importSkillMarkdown(data)
	}
}

type skillFrontMatter struct {
	Name                 string   `yaml:"name"`
	Description          string   `yaml:"description"`
	Version              string   `yaml:"version,omitempty"`
	FocusAreas           []string `yaml:"focusAreas,omitempty"`
	ComposedOf           []string `yaml:"composedOf,omitempty"`
	CapabilityBoundaries []string `yaml:"capabilityBoundaries,omitempty"`
	SampleQuestions      []string `yaml:"sampleQuestions,omitempty"`
	FollowUps            []string `yaml:"followUps,omitempty"`
	Scenarios            []string `yaml:"scenarios,omitempty"`
	Adversarial          []string `yaml:"adversarial,omitempty"`
	Pressure             []string `yaml:"pressure,omitempty"`
	ScoringAnchors       []string `yaml:"scoringAnchors,omitempty"`
	InstallSource        string   `yaml:"installSource,omitempty"`
	SourceURL            string   `yaml:"sourceUrl,omitempty"`
	Rating               float64  `yaml:"rating,omitempty"`
	RatingCount          int      `yaml:"ratingCount,omitempty"`
}

func parseSkillFrontMatter(content string) (skillFrontMatter, error) {
	fm, _, err := parseSkillDocument(content)
	return fm, err
}

func parseSkillDocument(content string) (skillFrontMatter, string, error) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return skillFrontMatter{}, "", fmt.Errorf("missing frontmatter")
	}

	rest := content[len("---"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return skillFrontMatter{}, "", fmt.Errorf("missing closing frontmatter delimiter")
	}

	raw := strings.TrimSpace(rest[:end])
	var fm skillFrontMatter
	if err := yaml.Unmarshal([]byte(raw), &fm); err != nil {
		return skillFrontMatter{}, "", err
	}
	body := strings.TrimSpace(rest[end+len("\n---"):])
	return fm, body, nil
}

func firstNonEmptySkillValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func renderSkillDocument(doc SkillDocument) (string, error) {
	fm := skillFrontMatter{
		Name:                 strings.TrimSpace(doc.Name),
		Description:          strings.TrimSpace(doc.Description),
		Version:              strings.TrimSpace(doc.Version),
		FocusAreas:           normalizeBulletList(doc.FocusAreas),
		ComposedOf:           normalizeBulletList(doc.ComposedOf),
		CapabilityBoundaries: normalizeBulletList(doc.CapabilityBoundaries),
		SampleQuestions:      normalizeBulletList(doc.SampleQuestions),
		FollowUps:            normalizeBulletList(doc.FollowUps),
		Scenarios:            normalizeBulletList(doc.Scenarios),
		Adversarial:          normalizeBulletList(doc.Adversarial),
		Pressure:             normalizeBulletList(doc.Pressure),
		ScoringAnchors:       normalizeBulletList(doc.ScoringAnchors),
		InstallSource:        strings.TrimSpace(doc.InstallSource),
		SourceURL:            strings.TrimSpace(doc.SourceURL),
		Rating:               doc.Rating,
		RatingCount:          doc.RatingCount,
	}
	raw, err := yaml.Marshal(fm)
	if err != nil {
		return "", fmt.Errorf("marshal frontmatter: %w", err)
	}
	var b strings.Builder
	b.WriteString("---\n")
	b.Write(raw)
	b.WriteString("---\n\n")
	b.WriteString(strings.TrimSpace(doc.Content))
	if !strings.HasSuffix(b.String(), "\n") {
		b.WriteString("\n")
	}
	return b.String(), nil
}

func sanitizeSkillName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, " ", "-")
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "-_")
}

func importSkillMarkdown(data []byte) (SkillMetadata, error) {
	fm, body, err := parseSkillDocument(string(data))
	if err != nil {
		return SkillMetadata{}, err
	}
	return SaveSkillDocument(SkillDocument{
		Name:                 firstNonEmptySkillValue(fm.Name),
		Description:          strings.TrimSpace(fm.Description),
		Version:              strings.TrimSpace(fm.Version),
		FocusAreas:           normalizeBulletList(fm.FocusAreas),
		ComposedOf:           normalizeBulletList(fm.ComposedOf),
		CapabilityBoundaries: normalizeBulletList(fm.CapabilityBoundaries),
		SampleQuestions:      normalizeBulletList(fm.SampleQuestions),
		FollowUps:            normalizeBulletList(fm.FollowUps),
		Scenarios:            normalizeBulletList(fm.Scenarios),
		Adversarial:          normalizeBulletList(fm.Adversarial),
		Pressure:             normalizeBulletList(fm.Pressure),
		ScoringAnchors:       normalizeBulletList(fm.ScoringAnchors),
		InstallSource:        strings.TrimSpace(fm.InstallSource),
		SourceURL:            strings.TrimSpace(fm.SourceURL),
		Rating:               fm.Rating,
		RatingCount:          fm.RatingCount,
		Content:              body,
	})
}

func importSkillArchive(data []byte) (SkillMetadata, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return SkillMetadata{}, fmt.Errorf("open archive: %w", err)
	}

	var (
		skillDoc   *SkillDocument
		targetName string
	)

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Base(file.Name), "SKILL.md") {
			rc, openErr := file.Open()
			if openErr != nil {
				return SkillMetadata{}, openErr
			}
			content, readErr := io.ReadAll(rc)
			_ = rc.Close()
			if readErr != nil {
				return SkillMetadata{}, readErr
			}
			fm, body, parseErr := parseSkillDocument(string(content))
			if parseErr != nil {
				return SkillMetadata{}, parseErr
			}
			name := sanitizeSkillName(firstNonEmptySkillValue(fm.Name, strings.TrimSuffix(filepath.Base(filepath.Dir(file.Name)), "/")))
			if name == "" {
				return SkillMetadata{}, fmt.Errorf("skill name is required in archive")
			}
			targetName = name
			skillDoc = &SkillDocument{
				Name:                 name,
				Description:          strings.TrimSpace(fm.Description),
				Version:              strings.TrimSpace(fm.Version),
				FocusAreas:           normalizeBulletList(fm.FocusAreas),
				ComposedOf:           normalizeBulletList(fm.ComposedOf),
				CapabilityBoundaries: normalizeBulletList(fm.CapabilityBoundaries),
				SampleQuestions:      normalizeBulletList(fm.SampleQuestions),
				FollowUps:            normalizeBulletList(fm.FollowUps),
				Scenarios:            normalizeBulletList(fm.Scenarios),
				Adversarial:          normalizeBulletList(fm.Adversarial),
				Pressure:             normalizeBulletList(fm.Pressure),
				ScoringAnchors:       normalizeBulletList(fm.ScoringAnchors),
				InstallSource:        strings.TrimSpace(fm.InstallSource),
				SourceURL:            strings.TrimSpace(fm.SourceURL),
				Rating:               fm.Rating,
				RatingCount:          fm.RatingCount,
				Content:              body,
			}
			break
		}
	}

	if skillDoc == nil {
		return SkillMetadata{}, fmt.Errorf("archive does not contain SKILL.md")
	}

	meta, err := SaveSkillDocument(*skillDoc)
	if err != nil {
		return SkillMetadata{}, err
	}

	dir := filepath.Join(SkillsBaseDir(), targetName)
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		cleanName := filepath.Clean(file.Name)
		if strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
			continue
		}
		dst := filepath.Join(dir, cleanName)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return SkillMetadata{}, err
		}
		rc, openErr := file.Open()
		if openErr != nil {
			return SkillMetadata{}, openErr
		}
		content, readErr := io.ReadAll(rc)
		_ = rc.Close()
		if readErr != nil {
			return SkillMetadata{}, readErr
		}
		if err := os.WriteFile(dst, content, 0o644); err != nil {
			return SkillMetadata{}, err
		}
	}

	return meta, nil
}
