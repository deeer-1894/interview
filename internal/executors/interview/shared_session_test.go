package interview

import (
	"strings"
	"testing"
	"time"

	"mockinterview/internal/protocol"
)

func TestBuildSharedSessionContextPreservesRoleInputs(t *testing.T) {
	t.Parallel()

	shared := BuildSharedSessionContext(
		"run_1",
		"interviewer.v2",
		protocol.RunPhaseInterviewing,
		protocol.RunInterviewState{Phase: protocol.PhaseProbe, Round: 3},
		protocol.SkillSpec{Name: "go-agent", FocusAreas: []string{"observability"}},
		protocol.Rubric{Title: "Go Agent Rubric"},
		"User: hello\nAssistant: world",
	)

	if shared.RunID != "run_1" {
		t.Fatalf("unexpected run id: %s", shared.RunID)
	}
	if shared.Skill.Name != "go-agent" {
		t.Fatalf("unexpected skill name: %s", shared.Skill.Name)
	}
	if shared.Rubric.Title != "Go Agent Rubric" {
		t.Fatalf("unexpected rubric title: %s", shared.Rubric.Title)
	}
	if !strings.Contains(shared.Summary(), "prompt=interviewer.v2") {
		t.Fatalf("expected prompt version in summary, got %s", shared.Summary())
	}
}

func TestBuildAgentExecutionCarriesSharedContextSummary(t *testing.T) {
	t.Parallel()

	startedAt := time.Now().Add(-2 * time.Second)
	completedAt := time.Now()
	shared := BuildSharedSessionContext(
		"run_2",
		"",
		protocol.RunPhaseEvaluating,
		protocol.RunInterviewState{},
		protocol.SkillSpec{Name: "backend"},
		protocol.Rubric{Title: "Backend Rubric"},
		"User: answer",
	)

	execution := BuildAgentExecution(
		protocol.AgentRoleEvaluator,
		protocol.RunCompleted,
		shared,
		startedAt,
		completedAt,
		"very long input that should still be summarized cleanly",
		"summary output",
		nil,
	)

	if execution.Role != protocol.AgentRoleEvaluator {
		t.Fatalf("unexpected role: %s", execution.Role)
	}
	if execution.Status != protocol.RunCompleted {
		t.Fatalf("unexpected status: %s", execution.Status)
	}
	if execution.SharedContextSummary == "" {
		t.Fatal("expected shared context summary")
	}
	if execution.InputSummary == "" || execution.OutputSummary == "" {
		t.Fatalf("expected summarized input/output, got %#v", execution)
	}
}
