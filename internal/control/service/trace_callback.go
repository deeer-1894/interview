package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	runtimepkg "mockinterview/internal/control/runtime"
	"mockinterview/internal/protocol"
)

type EventTraceCallback struct {
	recorder runtimepkg.EventRecorder
	run      protocol.Run
}

func NewEventTraceCallback(recorder runtimepkg.EventRecorder, run protocol.Run) *EventTraceCallback {
	return &EventTraceCallback{
		recorder: recorder,
		run:      run,
	}
}

func (c *EventTraceCallback) OnSpanStart(ctx context.Context, span runtimepkg.Span) {
	_ = c.recorder.RecordEvent(ctx, protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: c.run.ConversationID,
		TaskID:         c.run.TaskID,
		RunID:          c.run.ID,
		Type:           protocol.EventTraceSpan,
		Timestamp:      time.Now(),
		Payload: map[string]any{
			"scope": span.Scope,
			"name":  span.Name,
			"phase": "start",
		},
	})
}

func (c *EventTraceCallback) OnSpanEnd(ctx context.Context, span runtimepkg.Span, err error, duration time.Duration) {
	payload := map[string]any{
		"scope":      span.Scope,
		"name":       span.Name,
		"phase":      "end",
		"durationMs": duration.Milliseconds(),
		"status":     "ok",
	}
	if err != nil {
		payload["status"] = "error"
		payload["error"] = err.Error()
	}

	_ = c.recorder.RecordEvent(ctx, protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: c.run.ConversationID,
		TaskID:         c.run.TaskID,
		RunID:          c.run.ID,
		Type:           protocol.EventTraceSpan,
		Timestamp:      time.Now(),
		Payload:        payload,
	})
}
