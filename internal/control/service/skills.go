package service

import (
	"context"
	"fmt"
	"strings"

	domain "mockinterview/internal/interview"
)

type SkillMetadata = domain.SkillMetadata
type SkillDocument = domain.SkillDocument

type SaveSkillInput struct {
	Name                 string
	Description          string
	Version              string
	FocusAreas           []string
	ComposedOf           []string
	CapabilityBoundaries []string
	SampleQuestions      []string
	FollowUps            []string
	Scenarios            []string
	Adversarial          []string
	Pressure             []string
	ScoringAnchors       []string
	InstallSource        string
	SourceURL            string
	Rating               float64
	RatingCount          int
	Content              string
}

func (in SaveSkillInput) toDocument(fallbackName string) domain.SkillDocument {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = strings.TrimSpace(fallbackName)
	}
	return domain.SkillDocument{
		Name:                 name,
		Description:          in.Description,
		Version:              in.Version,
		FocusAreas:           in.FocusAreas,
		ComposedOf:           in.ComposedOf,
		CapabilityBoundaries: in.CapabilityBoundaries,
		SampleQuestions:      in.SampleQuestions,
		FollowUps:            in.FollowUps,
		Scenarios:            in.Scenarios,
		Adversarial:          in.Adversarial,
		Pressure:             in.Pressure,
		ScoringAnchors:       in.ScoringAnchors,
		InstallSource:        in.InstallSource,
		SourceURL:            in.SourceURL,
		Rating:               in.Rating,
		RatingCount:          in.RatingCount,
		Content:              in.Content,
	}
}

func (a *App) ListSkills(ctx context.Context) ([]SkillMetadata, error) {
	_ = ctx
	skills, err := domain.ListInterviewSkills()
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}
	return skills, nil
}

func (a *App) GetSkill(ctx context.Context, name string) (SkillDocument, error) {
	_ = ctx
	doc, err := domain.LoadSkillDocument(strings.TrimSpace(name))
	if err != nil {
		return domain.SkillDocument{}, fmt.Errorf("load skill %q: %w", name, err)
	}
	return doc, nil
}

func (a *App) SaveSkill(ctx context.Context, input SaveSkillInput, fallbackName string) (SkillMetadata, error) {
	_ = ctx
	doc := input.toDocument(fallbackName)
	meta, err := domain.SaveSkillDocument(doc)
	if err != nil {
		return domain.SkillMetadata{}, fmt.Errorf("save skill %q: %w", doc.Name, err)
	}
	return meta, nil
}

func (a *App) ImportSkill(ctx context.Context, filename string, data []byte) (SkillMetadata, error) {
	_ = ctx
	meta, err := domain.ImportSkillFile(filename, data)
	if err != nil {
		return domain.SkillMetadata{}, fmt.Errorf("import skill %q: %w", filename, err)
	}
	return meta, nil
}
