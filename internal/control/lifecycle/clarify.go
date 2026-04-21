package lifecycle

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"mockinterview/internal/protocol"
)

type ClarifyDecisionInput struct {
	Run    protocol.Run
	Prompt string
	Now    time.Time
}

type ClarifyDecisionOutput struct {
	NeedsClarify bool
	Request      protocol.ClarifyRequest
	Event        protocol.Event
	Run          protocol.Run
}

func BuildPromptClarifyDecision(input ClarifyDecisionInput) ClarifyDecisionOutput {
	if strings.TrimSpace(input.Prompt) != "" {
		return ClarifyDecisionOutput{}
	}

	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}

	request := protocol.ClarifyRequest{
		ID:        uuid.NewString(),
		RunID:     input.Run.ID,
		Question:  "请先描述你希望模拟的面试场景。",
		Field:     "prompt",
		Status:    "pending",
		CreatedAt: now,
	}
	event := protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: input.Run.ConversationID,
		TaskID:         input.Run.TaskID,
		RunID:          input.Run.ID,
		Type:           protocol.EventClarifyRequested,
		Timestamp:      now,
		Payload: map[string]string{
			"question": request.Question,
			"field":    request.Field,
		},
	}

	run := input.Run
	run.Status = protocol.RunWaitingClarify

	return ClarifyDecisionOutput{
		NeedsClarify: true,
		Request:      request,
		Event:        event,
		Run:          run,
	}
}
