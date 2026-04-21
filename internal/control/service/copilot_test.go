package service

import (
	"context"
	"testing"
	"time"

	"mockinterview/internal/protocol"
)

func TestRequestCopilotHintRecordsFeedbackAndHintEvents(t *testing.T) {
	t.Parallel()

	now := time.Now()
	task := protocol.Task{
		ID:             "task_copilot",
		ConversationID: "conv_1",
		Config: protocol.InterviewConfig{
			Skill:       "go-agent",
			Mode:        protocol.ModeStandard,
			OutputStyle: protocol.OutputInterviewPlusScore,
		}.WithDefaults(),
	}
	run := protocol.Run{
		ID:             "run_copilot",
		ConversationID: "conv_1",
		TaskID:         task.ID,
		Status:         protocol.RunCompleted,
		Phase:          protocol.RunPhaseInterviewing,
		CreatedAt:      now.Add(-5 * time.Minute),
		UpdatedAt:      now.Add(-1 * time.Minute),
		InterviewState: &protocol.RunInterviewState{
			Phase: protocol.PhaseProbe,
			LastDecision: &protocol.NextStepDecision{
				Reason:           protocol.ReasonMissingTradeoff,
				Explanation:      "继续追问 tradeoff。",
				RecommendedFocus: []string{"tradeoff"},
			},
			History: []protocol.InterviewRoundSnapshot{
				{
					Round:       2,
					WeakSignals: []string{"too_generic"},
				},
			},
		},
	}
	messages := []protocol.Message{
		{
			ID:             "msg_assistant",
			ConversationID: run.ConversationID,
			TaskID:         run.TaskID,
			RunID:          run.ID,
			Role:           "assistant",
			Content:        "如果让你在 errgroup 和自定义 worker pool 之间做选择，你会怎么取舍？",
			CreatedAt:      now.Add(-2 * time.Minute),
		},
		{
			ID:             "msg_user",
			ConversationID: run.ConversationID,
			TaskID:         run.TaskID,
			RunID:          run.ID,
			Role:           "user",
			Content:        "我有点卡住了，不知道怎么展开。",
			CreatedAt:      now.Add(-90 * time.Second),
		},
	}

	app := &App{
		tasks: &memoryTaskRepo{
			items: map[string]protocol.Task{task.ID: task},
		},
		runs: &memoryRunRepo{
			items: map[string]protocol.Run{run.ID: run},
		},
		messages: &memoryMessageRepo{
			byRun: map[string][]protocol.Message{run.ID: messages},
		},
		events: &memoryEventRepo{
			byRun: make(map[string][]protocol.Event),
		},
		broker: NewEventBroker(),
	}

	result, err := app.RequestCopilotHint(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("RequestCopilotHint returned error: %v", err)
	}
	if result.Feedback.State != protocol.CopilotStateStuck {
		t.Fatalf("expected stuck feedback state, got %s", result.Feedback.State)
	}
	if len(result.Hint.Strategy) == 0 {
		t.Fatalf("expected non-empty hint strategy")
	}
	if len(result.Hint.Guardrails) == 0 {
		t.Fatalf("expected non-empty hint guardrails")
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected 2 recorded events, got %d", len(result.Events))
	}

	events, err := app.events.ListByRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("ListByRun returned error: %v", err)
	}
	assertEventRecorded(t, events, protocol.EventCopilotFeedback)
	assertEventRecorded(t, events, protocol.EventCopilotHint)
}
