package runtime

import (
	"context"
	"time"

	"github.com/google/uuid"

	"mockinterview/internal/protocol"
)

type ObservedToolGateway struct {
	base     ToolGateway
	recorder EventRecorder
	run      protocol.Run
}

func ObserveTools(base ToolGateway, recorder EventRecorder, run protocol.Run) ToolGateway {
	if base == nil || recorder == nil {
		return base
	}
	return &ObservedToolGateway{
		base:     base,
		recorder: recorder,
		run:      run,
	}
}

func (g *ObservedToolGateway) ResolveSkill(ctx context.Context, cfg protocol.InterviewConfig) (protocol.SkillSpec, error) {
	g.record(ctx, protocol.EventToolCalled, map[string]any{
		"tool":         "skill.resolve",
		"skill":        cfg.Skill,
		"skillFocuses": append([]string(nil), cfg.SkillFocuses...),
	})
	skill, err := g.base.ResolveSkill(ctx, cfg)
	err = protocol.WrapToolError("tool_gateway", "skill.resolve", false, err)
	g.record(ctx, protocol.EventToolCompleted, map[string]any{
		"tool":         "skill.resolve",
		"skill":        cfg.Skill,
		"skillFocuses": append([]string(nil), cfg.SkillFocuses...),
		"name":         skill.Name,
		"status":       toolStatus(err),
		"error":        errorString(err),
	})
	return skill, err
}

func (g *ObservedToolGateway) ResolveRubric(ctx context.Context, cfg protocol.InterviewConfig) (protocol.Rubric, error) {
	g.record(ctx, protocol.EventToolCalled, map[string]any{
		"tool":         "rubric.resolve",
		"skill":        cfg.Skill,
		"skillFocuses": append([]string(nil), cfg.SkillFocuses...),
	})
	rubric, err := g.base.ResolveRubric(ctx, cfg)
	err = protocol.WrapToolError("tool_gateway", "rubric.resolve", false, err)
	g.record(ctx, protocol.EventToolCompleted, map[string]any{
		"tool":         "rubric.resolve",
		"skill":        cfg.Skill,
		"skillFocuses": append([]string(nil), cfg.SkillFocuses...),
		"title":        rubric.Title,
		"anchorCount":  len(rubric.Anchors),
		"status":       toolStatus(err),
		"error":        errorString(err),
	})
	return rubric, err
}

func (g *ObservedToolGateway) AppendMemory(ctx context.Context, record protocol.MemoryRecord) error {
	g.record(ctx, protocol.EventToolCalled, map[string]any{
		"tool":  "memory.append",
		"runId": record.RunID,
	})
	err := g.base.AppendMemory(ctx, record)
	err = protocol.WrapToolError("tool_gateway", "memory.append", false, err)
	g.record(ctx, protocol.EventToolCompleted, map[string]any{
		"tool":   "memory.append",
		"runId":  record.RunID,
		"status": toolStatus(err),
		"error":  errorString(err),
	})
	return err
}

func (g *ObservedToolGateway) LoadMemory(ctx context.Context, runID string) ([]protocol.MemoryRecord, error) {
	g.record(ctx, protocol.EventToolCalled, map[string]any{
		"tool":  "memory.get",
		"runId": runID,
	})
	records, err := g.base.LoadMemory(ctx, runID)
	err = protocol.WrapToolError("tool_gateway", "memory.get", false, err)
	g.record(ctx, protocol.EventToolCompleted, map[string]any{
		"tool":   "memory.get",
		"runId":  runID,
		"count":  len(records),
		"status": toolStatus(err),
		"error":  errorString(err),
	})
	return records, err
}

func (g *ObservedToolGateway) SaveCheckpoint(ctx context.Context, snapshot protocol.CheckpointSnapshot) error {
	g.record(ctx, protocol.EventToolCalled, map[string]any{
		"tool":   "checkpoint.save",
		"runId":  snapshot.RunID,
		"status": snapshot.RunStatus,
	})
	err := g.base.SaveCheckpoint(ctx, snapshot)
	err = protocol.WrapToolError("tool_gateway", "checkpoint.save", false, err)
	g.record(ctx, protocol.EventToolCompleted, map[string]any{
		"tool":        "checkpoint.save",
		"runId":       snapshot.RunID,
		"status":      toolStatus(err),
		"error":       errorString(err),
		"resumeCount": snapshot.ResumeCount,
		"runStatus":   snapshot.RunStatus,
	})
	return err
}

func (g *ObservedToolGateway) LoadCheckpoint(ctx context.Context, runID string) (protocol.CheckpointSnapshot, error) {
	g.record(ctx, protocol.EventToolCalled, map[string]any{
		"tool":  "checkpoint.load",
		"runId": runID,
	})
	snapshot, err := g.base.LoadCheckpoint(ctx, runID)
	err = protocol.WrapToolError("tool_gateway", "checkpoint.load", false, err)
	g.record(ctx, protocol.EventToolCompleted, map[string]any{
		"tool":        "checkpoint.load",
		"runId":       runID,
		"status":      toolStatus(err),
		"error":       errorString(err),
		"resumeCount": snapshot.ResumeCount,
		"runStatus":   snapshot.RunStatus,
	})
	return snapshot, err
}

func (g *ObservedToolGateway) ListArtifacts(ctx context.Context, conversationID string) ([]protocol.Artifact, error) {
	g.record(ctx, protocol.EventToolCalled, map[string]any{
		"tool":           "artifact.list",
		"conversationId": conversationID,
	})
	artifacts, err := g.base.ListArtifacts(ctx, conversationID)
	err = protocol.WrapToolError("tool_gateway", "artifact.list", false, err)
	g.record(ctx, protocol.EventToolCompleted, map[string]any{
		"tool":           "artifact.list",
		"conversationId": conversationID,
		"count":          len(artifacts),
		"status":         toolStatus(err),
		"error":          errorString(err),
	})
	return artifacts, err
}

func (g *ObservedToolGateway) GetArtifact(ctx context.Context, id string) (protocol.Artifact, error) {
	g.record(ctx, protocol.EventToolCalled, map[string]any{
		"tool":       "artifact.get",
		"artifactId": id,
	})
	artifact, err := g.base.GetArtifact(ctx, id)
	err = protocol.WrapToolError("tool_gateway", "artifact.get", false, err)
	g.record(ctx, protocol.EventToolCompleted, map[string]any{
		"tool":       "artifact.get",
		"artifactId": id,
		"name":       artifact.Name,
		"status":     toolStatus(err),
		"error":      errorString(err),
	})
	return artifact, err
}

func (g *ObservedToolGateway) SearchWeb(ctx context.Context, query string, limit int) ([]protocol.WebSearchResult, error) {
	g.record(ctx, protocol.EventToolCalled, map[string]any{
		"tool":  "web.search",
		"query": query,
		"limit": limit,
	})
	results, err := g.base.SearchWeb(ctx, query, limit)
	err = protocol.WrapToolError("tool_gateway", "web.search", false, err)
	g.record(ctx, protocol.EventToolCompleted, map[string]any{
		"tool":   "web.search",
		"query":  query,
		"limit":  limit,
		"count":  len(results),
		"status": toolStatus(err),
		"error":  errorString(err),
	})
	return results, err
}

func (g *ObservedToolGateway) record(ctx context.Context, typ protocol.EventType, payload map[string]any) {
	_ = g.recorder.RecordEvent(ctx, protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: g.run.ConversationID,
		TaskID:         g.run.TaskID,
		RunID:          g.run.ID,
		Type:           typ,
		Timestamp:      time.Now(),
		Payload:        payload,
	})
}

func toolStatus(err error) string {
	if err != nil {
		return "error"
	}
	return "ok"
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
