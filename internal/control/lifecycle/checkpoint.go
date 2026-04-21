package lifecycle

import (
	"strings"
	"time"

	"github.com/google/uuid"

	domain "mockinterview/internal/interview"
	"mockinterview/internal/protocol"
)

type CheckpointResumeInput struct {
	Request protocol.RunRequest
	Task    protocol.Task
	Run     protocol.Run
	Prompt  string
}

type CheckpointResumeOutcome struct {
	Snapshot protocol.CheckpointSnapshot
	Task     protocol.Task
	Run      protocol.Run
	Prompt   string
}

type CheckpointSaveInput struct {
	Request  protocol.RunRequest
	Task     protocol.Task
	Run      protocol.Run
	Prompt   string
	Previous *protocol.CheckpointSnapshot
	SavedAt  time.Time
}

func ApplyCheckpointSnapshot(input CheckpointResumeInput, snapshot protocol.CheckpointSnapshot) CheckpointResumeOutcome {
	task := input.Task
	run := input.Run
	prompt := strings.TrimSpace(input.Prompt)

	task.Config = snapshot.Config.WithDefaults()
	task.ModelConfig = snapshot.ModelConfig
	if snapshot.RunPhase != "" {
		run.Phase = snapshot.RunPhase
	}
	if snapshot.InterviewState != nil {
		state := domain.EnsureRunInterviewState(snapshot.InterviewState)
		run.InterviewState = &state
	}
	if prompt == "" {
		prompt = snapshot.Prompt
	}
	if snapshot.RunStatus == protocol.RunWaitingClarify &&
		snapshot.PendingClarifyFor == "prompt" &&
		strings.TrimSpace(input.Request.ClarifyResponse) != "" {
		prompt = AppendResumeResponse(prompt, input.Request.ClarifyResponse)
	}
	if strings.TrimSpace(run.Input) == "" {
		run.Input = snapshot.Input
	}

	return CheckpointResumeOutcome{
		Snapshot: snapshot,
		Task:     task,
		Run:      run,
		Prompt:   prompt,
	}
}

func BuildCheckpointLoadedEvent(run protocol.Run, snapshot protocol.CheckpointSnapshot, at time.Time) protocol.Event {
	if at.IsZero() {
		at = time.Now()
	}
	return protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: run.ConversationID,
		TaskID:         run.TaskID,
		RunID:          run.ID,
		Type:           protocol.EventCheckpointLoaded,
		Timestamp:      at,
		Payload: map[string]any{
			"runStatus":   string(snapshot.RunStatus),
			"resumeCount": snapshot.ResumeCount,
		},
	}
}

func BuildCheckpointSnapshot(input CheckpointSaveInput) protocol.CheckpointSnapshot {
	savedAt := input.SavedAt
	if savedAt.IsZero() {
		savedAt = time.Now()
	}

	snapshot := protocol.CheckpointSnapshot{
		RunID:             input.Run.ID,
		ConversationID:    input.Run.ConversationID,
		TaskID:            input.Run.TaskID,
		Prompt:            input.Prompt,
		Input:             input.Run.Input,
		RunStatus:         input.Run.Status,
		RunPhase:          input.Run.Phase,
		InterviewState:    input.Run.InterviewState,
		Config:            input.Task.Config,
		ModelConfig:       input.Task.ModelConfig,
		PendingClarifyFor: "",
		UpdatedAt:         savedAt,
	}
	if input.Run.Status == protocol.RunWaitingClarify {
		snapshot.PendingClarifyFor = "prompt"
	}

	if input.Previous != nil {
		snapshot.ResumeCount = input.Previous.ResumeCount
		snapshot.RawState = append([]byte(nil), input.Previous.RawState...)
		if input.Request.Resume {
			snapshot.ResumeCount++
		}
	} else if input.Request.Resume {
		snapshot.ResumeCount = 1
	}

	return snapshot
}

func BuildCheckpointSavedEvent(run protocol.Run, snapshot protocol.CheckpointSnapshot, at time.Time) protocol.Event {
	if at.IsZero() {
		at = time.Now()
	}
	return protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: run.ConversationID,
		TaskID:         run.TaskID,
		RunID:          run.ID,
		Type:           protocol.EventCheckpointSaved,
		Timestamp:      at,
		Payload: map[string]any{
			"status":      string(snapshot.RunStatus),
			"resumeCount": snapshot.ResumeCount,
		},
	}
}

func AppendResumeResponse(prompt, response string) string {
	prompt = strings.TrimSpace(prompt)
	response = strings.TrimSpace(response)
	if response == "" {
		return prompt
	}
	if prompt == "" {
		return response
	}
	var b strings.Builder
	b.WriteString(prompt)
	b.WriteString("\n\nLatest user continuation:\n")
	b.WriteString(response)
	return b.String()
}
