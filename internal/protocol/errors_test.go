package protocol

import (
	"errors"
	"testing"
)

func TestWrapModelErrorPreservesStructure(t *testing.T) {
	t.Parallel()

	base := errors.New("stream timed out")
	err := WrapModelError("evaluation", "collect_output", true, base)

	payload := ErrorPayloadFromError(err)
	if payload == nil {
		t.Fatal("expected payload")
	}
	if payload.Kind != ErrorKindModel {
		t.Fatalf("expected model error kind, got %s", payload.Kind)
	}
	if payload.Stage != "evaluation" {
		t.Fatalf("expected evaluation stage, got %s", payload.Stage)
	}
	if payload.Operation != "collect_output" {
		t.Fatalf("expected collect_output operation, got %s", payload.Operation)
	}
	if !payload.Retryable {
		t.Fatal("expected retryable payload")
	}
}

func TestWrapRunErrorDoesNotDoubleWrap(t *testing.T) {
	t.Parallel()

	base := WrapToolError("tool_gateway", "skill.resolve", false, errors.New("missing skill"))
	wrapped := WrapInterviewError("executor", "run", false, base)

	payload := ErrorPayloadFromError(wrapped)
	if payload == nil {
		t.Fatal("expected payload")
	}
	if payload.Kind != ErrorKindTool {
		t.Fatalf("expected original tool error kind, got %s", payload.Kind)
	}
	if payload.Stage != "tool_gateway" {
		t.Fatalf("expected original stage, got %s", payload.Stage)
	}
}
