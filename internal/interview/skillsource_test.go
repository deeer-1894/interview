package interview

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveInterviewFocusAreasCombinesPrimaryAndExtraFocuses(t *testing.T) {
	t.Parallel()

	focuses := ResolveInterviewFocusAreas(InterviewConfig{
		Focus:        "observability",
		SkillFocuses: []string{"reliability", "observability", "ownership"},
	})

	if len(focuses) != 3 {
		t.Fatalf("expected 3 unique focuses, got %#v", focuses)
	}
	if focuses[0] != "observability" || focuses[1] != "reliability" || focuses[2] != "ownership" {
		t.Fatalf("unexpected focus ordering: %#v", focuses)
	}
}

func TestRenderSkillDocumentPreservesExtendedMetadata(t *testing.T) {
	t.Parallel()

	rendered, err := renderSkillDocument(SkillDocument{
		Name:                 "go-agent",
		Description:          "Go agent interview skill",
		Version:              "1.2.0",
		FocusAreas:           []string{"observability", "reliability"},
		ComposedOf:           []string{"backend-core", "ops-foundation"},
		CapabilityBoundaries: []string{"只覆盖 Go agent runtime", "不覆盖前端系统设计"},
		InstallSource:        "marketplace",
		SourceURL:            "https://example.com/go-agent",
		Rating:               4.7,
		RatingCount:          18,
		Content:              "# Go Agent\n",
	})
	if err != nil {
		t.Fatalf("renderSkillDocument returned error: %v", err)
	}

	fm, body, err := parseSkillDocument(rendered)
	if err != nil {
		t.Fatalf("parseSkillDocument returned error: %v", err)
	}
	if fm.Version != "1.2.0" {
		t.Fatalf("expected version to survive render/parse, got %q", fm.Version)
	}
	if len(fm.ComposedOf) != 2 || fm.ComposedOf[0] != "backend-core" {
		t.Fatalf("expected composedOf to survive render/parse, got %#v", fm.ComposedOf)
	}
	if fm.InstallSource != "marketplace" || fm.SourceURL != "https://example.com/go-agent" {
		t.Fatalf("unexpected source metadata: %#v", fm)
	}
	if fm.Rating != 4.7 || fm.RatingCount != 18 {
		t.Fatalf("unexpected rating metadata: %#v", fm)
	}
	if body != "# Go Agent" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestMergeSkillDocumentsPrefersOverlayAndKeepsDependencies(t *testing.T) {
	t.Parallel()

	merged := mergeSkillDocuments(
		SkillDocument{
			Name:                 "backend-core",
			Description:          "Base skill",
			Version:              "0.9.0",
			FocusAreas:           []string{"reliability", "scalability"},
			CapabilityBoundaries: []string{"base boundary"},
			SampleQuestions:      []string{"How do you handle retries?"},
			InstallSource:        "imported",
			Rating:               4.2,
		},
		SkillDocument{
			Name:                 "go-agent",
			Description:          "Overlay skill",
			Version:              "1.0.0",
			FocusAreas:           []string{"observability"},
			ComposedOf:           []string{"backend-core"},
			CapabilityBoundaries: []string{"overlay boundary"},
			SampleQuestions:      []string{"How do you wire errgroup with context?"},
			InstallSource:        "marketplace",
			Rating:               4.8,
		},
	)

	if merged.Name != "go-agent" || merged.Version != "1.0.0" {
		t.Fatalf("expected overlay metadata to win, got %#v", merged)
	}
	if len(merged.FocusAreas) != 3 || merged.FocusAreas[0] != "observability" {
		t.Fatalf("expected overlay focus first with base fallback, got %#v", merged.FocusAreas)
	}
	if len(merged.CapabilityBoundaries) != 2 || merged.CapabilityBoundaries[0] != "overlay boundary" {
		t.Fatalf("expected merged capability boundaries, got %#v", merged.CapabilityBoundaries)
	}
	if merged.InstallSource != "marketplace" || merged.Rating != 4.8 {
		t.Fatalf("expected overlay commercial metadata to win, got %#v", merged)
	}
	if len(merged.ComposedOf) != 1 || merged.ComposedOf[0] != "backend-core" {
		t.Fatalf("expected composedOf to be preserved, got %#v", merged.ComposedOf)
	}
}

func TestCurrentSkillSourceBoundaryDeclaresDeferredResponsibilities(t *testing.T) {
	t.Parallel()

	boundary := CurrentSkillSourceBoundary()
	if boundary.StorageScope == "" {
		t.Fatalf("expected storage scope to be declared")
	}
	if len(boundary.Supports) == 0 || len(boundary.Defers) == 0 {
		t.Fatalf("expected boundary declaration to include supports and defers, got %#v", boundary)
	}
}

func TestSkillSourceRoundTripSaveLoadListAndImport(t *testing.T) {
	savedName := "test-skill-save-load"
	importedName := "test-skill-import"
	archivedName := "test-skill-archive"
	for _, name := range []string{savedName, importedName, archivedName} {
		_ = os.RemoveAll(filepath.Join(SkillsBaseDir(), name))
		defer os.RemoveAll(filepath.Join(SkillsBaseDir(), name))
	}

	meta, err := SaveSkillDocument(SkillDocument{
		Name:                 "Test Skill Save Load",
		Description:          "round trip metadata",
		Version:              "1.0.1",
		FocusAreas:           []string{"observability", "reliability"},
		ComposedOf:           []string{"backend-core"},
		CapabilityBoundaries: []string{"only for test"},
		SampleQuestions:      []string{"How do you trace fan-out?"},
		FollowUps:            []string{"What signal proves the fix worked?"},
		Scenarios:            []string{"A dependency starts timing out."},
		InstallSource:        "test",
		SourceURL:            "https://example.com/test-skill",
		Rating:               4.5,
		RatingCount:          7,
		Content:              "# Test Skill\n",
	})
	if err != nil {
		t.Fatalf("SaveSkillDocument returned error: %v", err)
	}
	if meta.Name != savedName {
		t.Fatalf("expected sanitized saved name %q, got %#v", savedName, meta)
	}

	loadedMeta, err := LoadSkillMetadata(savedName)
	if err != nil {
		t.Fatalf("LoadSkillMetadata returned error: %v", err)
	}
	if loadedMeta.Version != "1.0.1" || len(loadedMeta.FocusAreas) != 2 {
		t.Fatalf("unexpected loaded metadata: %#v", loadedMeta)
	}

	loadedDoc, err := LoadSkillDocument(savedName)
	if err != nil {
		t.Fatalf("LoadSkillDocument returned error: %v", err)
	}
	if loadedDoc.Content != "# Test Skill" || loadedDoc.InstallSource != "test" {
		t.Fatalf("unexpected loaded document: %#v", loadedDoc)
	}

	skills, err := ListInterviewSkills()
	if err != nil {
		t.Fatalf("ListInterviewSkills returned error: %v", err)
	}
	if !hasSkillNamed(skills, savedName) {
		t.Fatalf("expected saved skill to appear in list, got %#v", skills)
	}

	importedMeta, err := ImportSkillFile("custom.md", []byte(`---
name: Test Skill Import
description: imported markdown
focusAreas:
  - tracing
  - retries
---

# Imported Skill
`))
	if err != nil {
		t.Fatalf("ImportSkillFile markdown returned error: %v", err)
	}
	if importedMeta.Name != importedName {
		t.Fatalf("unexpected imported markdown metadata: %#v", importedMeta)
	}

	var archive bytes.Buffer
	zipWriter := zip.NewWriter(&archive)
	skillFile, err := zipWriter.Create("bundle/SKILL.md")
	if err != nil {
		t.Fatalf("Create zip entry returned error: %v", err)
	}
	if _, err := skillFile.Write([]byte(`---
name: Test Skill Archive
description: imported archive
followUps:
  - Push on tradeoff evidence
---

# Archived Skill
`)); err != nil {
		t.Fatalf("Write zip entry returned error: %v", err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("Close zip writer returned error: %v", err)
	}

	archivedMeta, err := ImportSkillFile("custom.skill", archive.Bytes())
	if err != nil {
		t.Fatalf("ImportSkillFile archive returned error: %v", err)
	}
	if archivedMeta.Name != archivedName {
		t.Fatalf("unexpected imported archive metadata: %#v", archivedMeta)
	}
}

func TestParseSkillFrontMatterAndSanitizeSkillName(t *testing.T) {
	t.Parallel()

	fm, err := parseSkillFrontMatter(`---
name: Fancy Skill
description: test
version: 2.0.0
---

# Body`)
	if err != nil {
		t.Fatalf("parseSkillFrontMatter returned error: %v", err)
	}
	if fm.Name != "Fancy Skill" || fm.Version != "2.0.0" {
		t.Fatalf("unexpected frontmatter: %#v", fm)
	}

	if name := sanitizeSkillName(" Fancy Skill! v2 "); name != "fancy-skill-v2" {
		t.Fatalf("unexpected sanitized name: %q", name)
	}
}

func TestSkillsBaseHelpersResolveDefaults(t *testing.T) {
	t.Parallel()

	if base := SkillsBaseDir(); base == "" {
		t.Fatalf("expected skills base dir to resolve")
	}
	if dir := projectSkillsDir(); dir == "" {
		t.Fatalf("expected project skills dir to resolve inside workspace")
	}
	if name := ResolveSkillName(InterviewConfig{}); name != "agent-interview-sim" {
		t.Fatalf("expected default skill name, got %q", name)
	}
}

func TestParseSkillDocumentRejectsBrokenFrontMatter(t *testing.T) {
	t.Parallel()

	if _, _, err := parseSkillDocument("# Missing frontmatter"); err == nil {
		t.Fatalf("expected missing frontmatter to fail")
	}
	if _, _, err := parseSkillDocument("---\nname: broken\nbody"); err == nil {
		t.Fatalf("expected missing closing frontmatter delimiter to fail")
	}
}

func TestLoadSkillMetadataRejectsEmptyNameAndResolveSkillNamePrefersExplicitValue(t *testing.T) {
	t.Parallel()

	if _, err := LoadSkillMetadata(""); err == nil {
		t.Fatalf("expected empty skill name to fail")
	}
	if name := ResolveSkillName(InterviewConfig{Skill: "custom-skill"}); name != "custom-skill" {
		t.Fatalf("expected explicit skill name to win, got %q", name)
	}
}

func hasSkillNamed(skills []SkillMetadata, name string) bool {
	for _, skill := range skills {
		if strings.EqualFold(skill.Name, name) {
			return true
		}
	}
	return false
}
