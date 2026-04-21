package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	lifecyclepkg "mockinterview/internal/control/lifecycle"
	runtimepkg "mockinterview/internal/control/runtime"
	domain "mockinterview/internal/interview"
	"mockinterview/internal/protocol"
)

type App struct {
	conversations ConversationRepository
	tasks         TaskRepository
	runs          RunRepository
	messages      MessageRepository
	events        EventRepository
	profiles      ProfileRepository
	checkpoints   CheckpointRepository
	clarifies     ClarifyRequestRepository
	memories      MemoryRepository
	artifacts     ArtifactRepository
	broker        *EventBroker
	activeRuns    *ActiveRuns
	engine        runtimepkg.Engine
	tools         runtimepkg.ToolGateway
	files         ArtifactFileStore
	scorecards    ScorecardGenerator
}

type ArtifactFileStore interface {
	Save(storageKey string, src io.Reader) (int64, error)
	Open(storageKey string) (io.ReadCloser, error)
	Delete(storageKey string) error
}

type ScorecardGenerator func(
	ctx context.Context,
	transcript string,
	cfg protocol.InterviewConfig,
	modelCfg protocol.ModelConfig,
	skill protocol.SkillSpec,
	rubric protocol.Rubric,
) (protocol.Scorecard, error)

func NewApp(
	conversations ConversationRepository,
	tasks TaskRepository,
	runs RunRepository,
	messages MessageRepository,
	events EventRepository,
	profiles ProfileRepository,
	checkpoints CheckpointRepository,
	clarifies ClarifyRequestRepository,
	memories MemoryRepository,
	artifacts ArtifactRepository,
	files ArtifactFileStore,
	deps AppDependencies,
) (*App, error) {
	deps = deps.withDefaults()
	if deps.Engine == nil {
		return nil, fmt.Errorf("app engine is required")
	}
	if deps.Tools == nil {
		return nil, fmt.Errorf("app tool gateway is required")
	}
	return &App{
		conversations: conversations,
		tasks:         tasks,
		runs:          runs,
		messages:      messages,
		events:        events,
		profiles:      profiles,
		checkpoints:   checkpoints,
		clarifies:     clarifies,
		memories:      memories,
		artifacts:     artifacts,
		broker:        NewEventBroker(),
		activeRuns:    NewActiveRuns(),
		engine:        deps.Engine,
		tools:         deps.Tools,
		files:         files,
		scorecards:    deps.Scorecards,
	}, nil
}

func (a *App) CreateConversation(ctx context.Context, title string) (protocol.Conversation, error) {
	now := time.Now()
	conversation := protocol.Conversation{
		ID:        uuid.NewString(),
		Title:     defaultTitle(title),
		Status:    "active",
		Pinned:    false,
		Archived:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return a.conversations.Create(ctx, conversation)
}

func (a *App) RecoverInterruptedRuns(ctx context.Context) error {
	conversations, err := a.conversations.List(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, conversation := range conversations {
		runs, err := a.runs.ListByConversation(ctx, conversation.ID)
		if err != nil {
			return err
		}

		for _, run := range runs {
			if run.Status != protocol.RunRunning && run.Status != protocol.RunResuming {
				continue
			}

			run.Status = protocol.RunFailed
			run.LastError = "run interrupted by server restart"
			run.CompletedAt = &now
			run.UpdatedAt = now
			if _, err := a.runs.Update(ctx, run); err != nil {
				return err
			}

			_ = a.clarifies.Resolve(ctx, run.ID)
			if err := a.RecordEvent(ctx, protocol.Event{
				ID:             uuid.NewString(),
				ConversationID: run.ConversationID,
				TaskID:         run.TaskID,
				RunID:          run.ID,
				Type:           protocol.EventRunFailed,
				Timestamp:      now,
				Payload: map[string]string{
					"status": string(protocol.RunFailed),
					"error":  run.LastError,
				},
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (a *App) ListConversations(ctx context.Context) ([]protocol.Conversation, error) {
	conversations, err := a.conversations.List(ctx)
	if err != nil {
		return nil, err
	}
	filtered := make([]protocol.Conversation, 0, len(conversations))
	for _, conversation := range conversations {
		if strings.EqualFold(conversation.Status, "deleted") {
			continue
		}
		filtered = append(filtered, conversation)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].Pinned != filtered[j].Pinned {
			return filtered[i].Pinned
		}
		if filtered[i].Archived != filtered[j].Archived {
			return !filtered[i].Archived
		}
		return filtered[i].UpdatedAt.After(filtered[j].UpdatedAt)
	})
	for i := range filtered {
		filtered[i] = a.enrichConversationRunStatus(ctx, filtered[i])
	}
	return filtered, nil
}

func (a *App) GetConversation(ctx context.Context, id string) (protocol.Conversation, []protocol.Task, []protocol.Run, error) {
	conversation, err := a.conversations.Get(ctx, id)
	if err != nil {
		return protocol.Conversation{}, nil, nil, err
	}
	if strings.EqualFold(conversation.Status, "deleted") {
		return protocol.Conversation{}, nil, nil, fmt.Errorf("conversation %s not found", id)
	}
	tasks, err := a.tasks.ListByConversation(ctx, id)
	if err != nil {
		return protocol.Conversation{}, nil, nil, err
	}
	runs, err := a.runs.ListByConversation(ctx, id)
	if err != nil {
		return protocol.Conversation{}, nil, nil, err
	}
	conversation = a.enrichConversationRunStatus(ctx, conversation)
	return conversation, tasks, runs, nil
}

func (a *App) GetReviewSnapshot(ctx context.Context, runID string) (protocol.ReviewSnapshot, error) {
	run, messages, events, err := a.GetRun(ctx, runID)
	if err != nil {
		return protocol.ReviewSnapshot{}, err
	}
	task, taskErr := a.tasks.Get(ctx, run.TaskID)
	if taskErr != nil {
		task = protocol.Task{ID: run.TaskID, Config: protocol.InterviewConfig{}.WithDefaults()}
	}

	snapshot := protocol.ReviewSnapshot{
		RunID:          run.ID,
		GeneratedAt:    run.UpdatedAt,
		InterviewState: run.InterviewState,
	}
	if run.TraceTree != nil {
		snapshot.Trace = run.TraceTree
	}

	if reviewEvent, ok := latestEventOfType(events, protocol.EventReviewGenerated); ok {
		if decoded, ok := decodeEventPayload[protocol.ReviewSnapshot](reviewEvent.Payload); ok {
			snapshot = decoded
			if snapshot.RunID == "" {
				snapshot.RunID = run.ID
			}
			if snapshot.GeneratedAt.IsZero() {
				snapshot.GeneratedAt = reviewEvent.Timestamp
			}
			if snapshot.InterviewState == nil {
				snapshot.InterviewState = run.InterviewState
			}
		}
	}

	if snapshot.Trace == nil {
		if traceEvent, ok := latestEventOfType(events, protocol.EventTraceGenerated); ok {
			if decoded, ok := decodeEventPayload[protocol.InterviewTraceTree](traceEvent.Payload); ok {
				snapshot.Trace = &decoded
			}
		}
	}

	if snapshot.Scorecard == nil {
		if scoreEvent, ok := latestEventOfType(events, protocol.EventScoreGenerated); ok {
			if decoded, ok := decodeEventPayload[protocol.Scorecard](scoreEvent.Payload); ok {
				snapshot.Scorecard = &decoded
			}
		}
	}

	if snapshot.Decision == nil {
		if decisionEvent, ok := latestEventOfType(events, protocol.EventDecisionGenerated); ok {
			if decoded, ok := decodeEventPayload[protocol.DecisionAudit](decisionEvent.Payload); ok {
				snapshot.Decision = &decoded
			}
		}
	}

	if snapshot.Profile == nil && a.profiles != nil {
		if profileEvent, ok := latestEventOfType(events, protocol.EventProfileUpdated); ok {
			if decoded, ok := decodeEventPayload[protocol.CandidateProfile](profileEvent.Payload); ok {
				snapshot.Profile = &decoded
			}
		} else if profile, err := a.profiles.Get(ctx, "global"); err == nil {
			snapshot.Profile = &profile
		}
	}

	if snapshot.Trace == nil && len(messages) > 0 {
		task, taskErr := a.tasks.Get(ctx, run.TaskID)
		if taskErr == nil {
			var assistant protocol.Message
			existing := make([]protocol.Message, 0, len(messages))
			for i := len(messages) - 1; i >= 0; i-- {
				if strings.EqualFold(messages[i].Role, "assistant") {
					assistant = messages[i]
					existing = append(existing, messages[:i]...)
					break
				}
			}
			if assistant.ID != "" {
				trace := domain.BuildTraceTree(task.Config.Persona, existing, assistant, run.InterviewState, snapshot.Profile)
				snapshot.Trace = &trace
			}
		}
	}

	if snapshot.GeneratedAt.IsZero() {
		snapshot.GeneratedAt = run.UpdatedAt
	}
	if snapshot.Summary == nil {
		if snapshot.Trace != nil {
			summary := domain.BuildReviewSummary(run, task.Config, *snapshot.Trace, snapshot.Scorecard, snapshot.Profile)
			snapshot.Summary = &summary
		}
	}
	return snapshot, nil
}

func (a *App) enrichConversationRunStatus(ctx context.Context, conversation protocol.Conversation) protocol.Conversation {
	if strings.TrimSpace(conversation.LatestRunID) == "" {
		return conversation
	}
	run, err := a.runs.Get(ctx, conversation.LatestRunID)
	if err != nil {
		return conversation
	}
	conversation.LatestRunStatus = run.Status
	return conversation
}

func (a *App) GetCandidateProfile(ctx context.Context) (protocol.CandidateProfile, error) {
	if a.profiles == nil {
		return protocol.CandidateProfile{ID: "global"}, nil
	}
	profile, err := a.profiles.Get(ctx, "global")
	if err != nil {
		return protocol.CandidateProfile{ID: "global"}, nil
	}
	return profile, nil
}

func latestEventOfType(events []protocol.Event, eventType protocol.EventType) (protocol.Event, bool) {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Type == eventType {
			return events[i], true
		}
	}
	return protocol.Event{}, false
}

func decodeEventPayload[T any](payload any) (T, bool) {
	var out T
	if payload == nil {
		return out, false
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return out, false
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, false
	}
	return out, true
}

func (a *App) SaveCandidateProfile(ctx context.Context, profile protocol.CandidateProfile) (protocol.CandidateProfile, error) {
	if a.profiles == nil {
		return profile, nil
	}
	return a.profiles.Save(ctx, profile)
}

func (a *App) UpdateConversation(ctx context.Context, id string, title *string, pinned *bool, archived *bool) (protocol.Conversation, error) {
	conversation, err := a.conversations.Get(ctx, id)
	if err != nil {
		return protocol.Conversation{}, err
	}
	if strings.EqualFold(conversation.Status, "deleted") {
		return protocol.Conversation{}, fmt.Errorf("conversation %s not found", id)
	}
	if title != nil {
		conversation.Title = defaultTitle(*title)
	}
	if pinned != nil {
		conversation.Pinned = *pinned
	}
	if archived != nil {
		conversation.Archived = *archived
	}
	conversation.UpdatedAt = time.Now()
	return a.conversations.Update(ctx, conversation)
}

func (a *App) DeleteConversation(ctx context.Context, id string) (protocol.Conversation, error) {
	conversation, err := a.conversations.Get(ctx, id)
	if err != nil {
		return protocol.Conversation{}, err
	}
	if strings.EqualFold(conversation.Status, "deleted") {
		return conversation, nil
	}
	conversation.Status = "deleted"
	conversation.UpdatedAt = time.Now()
	return a.conversations.Update(ctx, conversation)
}

func (a *App) CreateTask(ctx context.Context, conversationID, title, prompt string, artifactIDs []string, cfg protocol.InterviewConfig, modelCfg protocol.ModelConfig) (protocol.Task, error) {
	conversation, err := a.conversations.Get(ctx, conversationID)
	if err != nil {
		return protocol.Task{}, err
	}

	now := time.Now()
	task := protocol.Task{
		ID:             uuid.NewString(),
		ConversationID: conversation.ID,
		Title:          defaultTaskTitle(title, prompt),
		Prompt:         strings.TrimSpace(prompt),
		ArtifactIDs:    normalizeIDs(artifactIDs),
		Config:         cfg.WithDefaults(),
		ModelConfig:    modelCfg,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	task, err = a.tasks.Create(ctx, task)
	if err != nil {
		return protocol.Task{}, err
	}

	conversation.CurrentTask = task.ID
	conversation.UpdatedAt = time.Now()
	if _, err := a.conversations.Update(ctx, conversation); err != nil {
		return protocol.Task{}, err
	}
	return task, nil
}

func (a *App) CreateRun(ctx context.Context, request protocol.RunRequest) (protocol.Run, error) {
	task, err := a.tasks.Get(ctx, request.TaskID)
	if err != nil {
		return protocol.Run{}, err
	}

	now := time.Now()
	run := protocol.Run{
		ID:             uuid.NewString(),
		ConversationID: task.ConversationID,
		TaskID:         task.ID,
		ArtifactIDs:    normalizeIDs(firstNonEmptyIDs(request.ArtifactIDs, task.ArtifactIDs)),
		Status:         protocol.RunCreated,
		Phase:          protocol.RunPhaseInitial,
		InterviewState: interviewStatePtr(domain.DefaultRunInterviewState()),
		Input:          firstNonEmpty(strings.TrimSpace(request.Prompt), task.Prompt),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	run, err = a.runs.Create(ctx, run)
	if err != nil {
		return protocol.Run{}, err
	}

	if _, err := a.messages.Create(ctx, protocol.Message{
		ID:             uuid.NewString(),
		ConversationID: run.ConversationID,
		TaskID:         run.TaskID,
		RunID:          run.ID,
		Role:           "user",
		Content:        run.Input,
		CreatedAt:      now,
	}); err != nil {
		return protocol.Run{}, err
	}

	if err := a.RecordEvent(ctx, protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: run.ConversationID,
		TaskID:         run.TaskID,
		RunID:          run.ID,
		Type:           protocol.EventRunCreated,
		Timestamp:      now,
		Payload: map[string]string{
			"status": string(protocol.RunCreated),
		},
	}); err != nil {
		return protocol.Run{}, err
	}

	conversation, err := a.conversations.Get(ctx, run.ConversationID)
	if err == nil {
		conversation.LatestRunID = run.ID
		conversation.LatestRunStatus = run.Status
		conversation.UpdatedAt = time.Now()
		_, _ = a.conversations.Update(ctx, conversation)
	}

	go a.executeRun(run, task, request)

	return run, nil
}

func (a *App) ResumeRun(ctx context.Context, runID string, input protocol.ResumeInput) (protocol.Run, error) {
	run, err := a.runs.Get(ctx, runID)
	if err != nil {
		return protocol.Run{}, err
	}
	if (run.Status == protocol.RunRunning || run.Status == protocol.RunResuming) && a.activeRuns.Has(runID) {
		return protocol.Run{}, fmt.Errorf("run %s is still active; wait for the current turn to finish before resuming", runID)
	}
	task, err := a.tasks.Get(ctx, run.TaskID)
	if err != nil {
		return protocol.Run{}, err
	}
	snapshot, snapshotErr := a.checkpoints.Load(ctx, runID)
	if err := a.clarifies.Resolve(ctx, runID); err != nil {
		// Resume can still proceed if clarify state was not persisted.
	}
	if snapshotErr == nil {
		task.Config = snapshot.Config.WithDefaults()
		task.ModelConfig = snapshot.ModelConfig
		if snapshot.RunPhase != "" {
			run.Phase = snapshot.RunPhase
		}
		if snapshot.InterviewState != nil {
			state := domain.EnsureRunInterviewState(snapshot.InterviewState)
			run.InterviewState = &state
		}
	}
	if hasInterviewConfigOverride(input.Config) {
		task.Config = input.Config.WithDefaults()
	}
	if input.ArtifactIDs != nil {
		run.ArtifactIDs = normalizeIDs(input.ArtifactIDs)
		task.ArtifactIDs = normalizeIDs(input.ArtifactIDs)
	}
	messages, messagesErr := a.messages.ListByRun(ctx, runID)
	if messagesErr != nil {
		return protocol.Run{}, messagesErr
	}
	events, eventsErr := a.events.ListByRun(ctx, runID)
	if eventsErr != nil {
		return protocol.Run{}, eventsErr
	}
	if runHasFinalWrapup(messages, events) {
		return protocol.Run{}, fmt.Errorf("run %s already has a final wrapup and score; create a new run to continue", runID)
	}
	input.Message = strings.TrimSpace(input.Message)
	if input.Message == "" {
		return protocol.Run{}, fmt.Errorf("resume message is required")
	}

	if lifecyclepkg.ShouldGenerateEvaluationRequest(input.Message) && task.Config.OutputStyle != protocol.OutputInterviewOnly {
		run = lifecyclepkg.ApplyWrapupRequestedState(run)
	} else {
		run.Status = protocol.RunResuming
	}
	run.Input = input.Message
	run.UpdatedAt = time.Now()
	run, err = a.runs.Update(ctx, run)
	if err != nil {
		return protocol.Run{}, err
	}

	if _, err := a.messages.Create(ctx, protocol.Message{
		ID:             uuid.NewString(),
		ConversationID: run.ConversationID,
		TaskID:         run.TaskID,
		RunID:          run.ID,
		Role:           "user",
		Content:        input.Message,
		CreatedAt:      time.Now(),
	}); err != nil {
		return protocol.Run{}, err
	}

	if err := a.RecordEvent(ctx, protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: run.ConversationID,
		TaskID:         run.TaskID,
		RunID:          run.ID,
		Type:           protocol.EventClarifyResumed,
		Timestamp:      time.Now(),
		Payload: map[string]string{
			"message": input.Message,
		},
	}); err != nil {
		return protocol.Run{}, err
	}

	go a.executeRun(run, task, protocol.RunRequest{
		TaskID:          run.TaskID,
		ConversationID:  run.ConversationID,
		Prompt:          buildResumePrompt(firstNonEmpty(snapshot.Prompt, task.Prompt), messages, input.Message),
		Resume:          true,
		ClarifyResponse: input.Message,
	})

	return run, nil
}

func hasInterviewConfigOverride(cfg protocol.InterviewConfig) bool {
	return strings.TrimSpace(cfg.Skill) != "" ||
		len(cfg.SkillFocuses) > 0 ||
		cfg.Persona != "" ||
		strings.TrimSpace(cfg.Level) != "" ||
		strings.TrimSpace(cfg.Focus) != "" ||
		cfg.Mode != "" ||
		strings.TrimSpace(cfg.TimeBudget) != "" ||
		cfg.OutputStyle != "" ||
		cfg.EnableWebSearch
}

func (a *App) CancelRun(ctx context.Context, runID string) (protocol.Run, error) {
	run, err := a.runs.Get(ctx, runID)
	if err != nil {
		return protocol.Run{}, err
	}
	if isTerminalRun(run.Status) {
		return run, nil
	}

	now := time.Now()
	run.Status = protocol.RunCancelled
	run.LastError = "run cancelled"
	run.CompletedAt = &now
	run.UpdatedAt = now
	run, err = a.runs.Update(ctx, run)
	if err != nil {
		return protocol.Run{}, err
	}
	if err := a.RecordEvent(ctx, protocol.Event{
		ID:             uuid.NewString(),
		ConversationID: run.ConversationID,
		TaskID:         run.TaskID,
		RunID:          run.ID,
		Type:           protocol.EventRunCancelled,
		Timestamp:      now,
		Payload: map[string]string{
			"status": string(protocol.RunCancelled),
		},
	}); err != nil {
		return protocol.Run{}, err
	}

	_ = a.clarifies.Resolve(ctx, runID)
	a.activeRuns.Cancel(runID)
	return run, nil
}

func (a *App) GetRun(ctx context.Context, runID string) (protocol.Run, []protocol.Message, []protocol.Event, error) {
	run, err := a.runs.Get(ctx, runID)
	if err != nil {
		return protocol.Run{}, nil, nil, err
	}
	messages, err := a.messages.ListByRun(ctx, runID)
	if err != nil {
		return protocol.Run{}, nil, nil, err
	}
	events, err := a.events.ListByRun(ctx, runID)
	if err != nil {
		return protocol.Run{}, nil, nil, err
	}
	sortMessages(messages)
	sortEvents(events)
	return run, messages, events, nil
}

func sortMessages(messages []protocol.Message) {
	sort.SliceStable(messages, func(i, j int) bool {
		if messages[i].CreatedAt.Equal(messages[j].CreatedAt) {
			return messages[i].ID < messages[j].ID
		}
		return messages[i].CreatedAt.Before(messages[j].CreatedAt)
	})
}

func sortEvents(events []protocol.Event) {
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].Timestamp.Equal(events[j].Timestamp) {
			return events[i].ID < events[j].ID
		}
		return events[i].Timestamp.Before(events[j].Timestamp)
	})
}

func (a *App) GetHealthMetrics(ctx context.Context) (protocol.HealthMetrics, error) {
	conversations, err := a.conversations.List(ctx)
	if err != nil {
		return protocol.HealthMetrics{}, err
	}

	metrics := protocol.HealthMetrics{Status: "ok"}
	var totalDurationMs int64

	for _, conversation := range conversations {
		deletedConversation := strings.EqualFold(conversation.Status, "deleted")
		if deletedConversation {
			metrics.DeletedConversationCount++
		} else {
			metrics.VisibleConversationCount++
		}
		runs, err := a.runs.ListByConversation(ctx, conversation.ID)
		if err != nil {
			return protocol.HealthMetrics{}, err
		}
		for _, run := range runs {
			metrics.StoredRunCount++
			if isTerminalRunStatus(run.Status) {
				metrics.StoredTerminalRuns++
				switch run.Status {
				case protocol.RunCompleted:
					metrics.StoredCompletedRuns++
				case protocol.RunFailed:
					metrics.StoredFailedRuns++
				case protocol.RunCancelled:
					metrics.StoredCancelledRuns++
				}
			}
			if deletedConversation {
				continue
			}
			metrics.RunCount++
			if !isTerminalRunStatus(run.Status) {
				if a.activeRuns != nil && a.activeRuns.Has(run.ID) {
					metrics.ActiveCount++
				}
				continue
			}
			metrics.TerminalRuns++
			switch run.Status {
			case protocol.RunCompleted:
				metrics.CompletedRuns++
			case protocol.RunFailed:
				metrics.FailedRuns++
			case protocol.RunCancelled:
				metrics.CancelledRuns++
			}

			completedAt := run.UpdatedAt
			if run.CompletedAt != nil && !run.CompletedAt.IsZero() {
				completedAt = *run.CompletedAt
			}
			durationMs := completedAt.Sub(run.CreatedAt).Milliseconds()
			if durationMs < 0 {
				durationMs = 0
			}
			totalDurationMs += durationMs

			totalTokens := runtimepkg.EstimateTextTokens(run.Input) + runtimepkg.EstimateTextTokens(run.Output)
			messages, err := a.messages.ListByRun(ctx, run.ID)
			if err != nil {
				return protocol.HealthMetrics{}, err
			}
			for _, message := range messages {
				totalTokens += runtimepkg.EstimateTextTokens(message.Content)
			}
			metrics.TotalEstimatedTokens += totalTokens
		}
	}

	if metrics.TerminalRuns > 0 {
		metrics.SuccessRate = float64(metrics.CompletedRuns) / float64(metrics.TerminalRuns)
		metrics.AverageDurationMs = totalDurationMs / int64(metrics.TerminalRuns)
		metrics.AverageEstimatedTokens = metrics.TotalEstimatedTokens / metrics.TerminalRuns
	}

	return metrics, nil
}

func (a *App) Subscribe(runID string) (<-chan protocol.Event, func(), error) {
	if _, err := a.runs.Get(context.Background(), runID); err != nil {
		return nil, nil, err
	}
	ch, cancel := a.broker.Subscribe(runID)
	return ch, cancel, nil
}

func (a *App) RecordEvent(ctx context.Context, event protocol.Event) error {
	if _, err := a.events.Create(ctx, event); err != nil {
		return err
	}
	a.broker.Publish(event)
	return nil
}

func (a *App) RecordMessage(ctx context.Context, message protocol.Message) error {
	_, err := a.messages.Create(ctx, message)
	return err
}

func (a *App) UpdateRun(ctx context.Context, run protocol.Run) error {
	run.UpdatedAt = time.Now()
	updatedRun, err := a.runs.Update(ctx, run)
	if err != nil {
		return err
	}
	if a.conversations == nil {
		return nil
	}
	conversation, convErr := a.conversations.Get(ctx, updatedRun.ConversationID)
	if convErr == nil {
		conversation.LatestRunID = updatedRun.ID
		conversation.LatestRunStatus = updatedRun.Status
		conversation.UpdatedAt = updatedRun.UpdatedAt
		_, _ = a.conversations.Update(ctx, conversation)
	}
	return nil
}

func (a *App) executeRun(run protocol.Run, task protocol.Task, request protocol.RunRequest) {
	execCtx, cancel := context.WithCancel(context.Background())
	a.activeRuns.Set(run.ID, cancel)
	defer a.activeRuns.Delete(run.ID)
	defer cancel()

	messages, _ := a.messages.ListByRun(execCtx, run.ID)
	runCtx := &runtimepkg.RunContext{
		Request:         request,
		Prompt:          firstNonEmpty(strings.TrimSpace(request.Prompt), strings.TrimSpace(run.Input), task.Prompt),
		Task:            task,
		Run:             run,
		Messages:        messages,
		Recorder:        a,
		Callbacks:       []runtimepkg.Callback{NewEventTraceCallback(a, run)},
		Tools:           runtimepkg.ObserveTools(a.tools, a, run),
		ADKCheckpoints:  newADKCheckpointStore(a.checkpoints, run, task),
		ClarifyRequests: a.clarifies,
	}
	if err := a.engine.Run(execCtx, runCtx); err != nil {
		return
	}
	if err := a.completePostInterviewEvaluation(execCtx, runCtx); err != nil && !errors.Is(err, context.Canceled) {
		a.finalizeRunFailure(execCtx, runCtx, err)
	}
}

func buildResumePrompt(basePrompt string, messages []protocol.Message, resumeMessage string) string {
	resumeMessage = strings.TrimSpace(resumeMessage)
	if resumeMessage == "" {
		return strings.TrimSpace(basePrompt)
	}

	transcript := buildCompactTranscript(messages, 6)
	if transcript == "" {
		return strings.TrimSpace(basePrompt)
	}

	var b strings.Builder
	b.WriteString(strings.TrimSpace(basePrompt))
	b.WriteString("\n\nContinue the same interview instead of restarting.\n")
	b.WriteString("Use the recent transcript below as the active conversation state.\n")
	b.WriteString("Do not re-introduce the interview or repeat the opening question.\n")
	b.WriteString("Acknowledge the user's latest reply only by continuing naturally.\n\n")
	b.WriteString("Recent transcript:\n")
	b.WriteString(transcript)
	return b.String()
}

func runHasFinalWrapup(messages []protocol.Message, events []protocol.Event) bool {
	if !hasEvaluationArtifacts(events) {
		return false
	}
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		if strings.EqualFold(message.Role, "assistant") && domain.IsWrapupAssistantMessage(message.Content) {
			return true
		}
	}
	return false
}

func hasEvaluationArtifacts(events []protocol.Event) bool {
	for _, event := range events {
		if event.Type == protocol.EventScoreGenerated || event.Type == protocol.EventReviewGenerated {
			return true
		}
	}
	return false
}

func buildCompactTranscript(messages []protocol.Message, limit int) string {
	if len(messages) == 0 {
		return ""
	}
	if limit <= 0 || len(messages) < limit {
		limit = len(messages)
	}
	selected := messages[len(messages)-limit:]
	var b strings.Builder
	for _, message := range selected {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		role := "Assistant"
		if strings.EqualFold(message.Role, "user") {
			role = "User"
		}
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(content)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func isTerminalRunStatus(status protocol.RunStatus) bool {
	return status == protocol.RunCompleted || status == protocol.RunFailed || status == protocol.RunCancelled
}

func defaultTitle(title string) string {
	if strings.TrimSpace(title) == "" {
		return "Interview Workspace"
	}
	return strings.TrimSpace(title)
}

func defaultTaskTitle(title, prompt string) string {
	if strings.TrimSpace(title) != "" {
		return strings.TrimSpace(title)
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "Untitled Interview Task"
	}
	if len(prompt) > 48 {
		return prompt[:48] + "..."
	}
	return prompt
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonEmptyIDs(values ...[]string) []string {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}

func normalizeIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	out := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func interviewStatePtr(state protocol.RunInterviewState) *protocol.RunInterviewState {
	value := state
	return &value
}

func (a *App) RequireConversation(ctx context.Context, id string) (protocol.Conversation, error) {
	conversation, err := a.conversations.Get(ctx, id)
	if err != nil {
		return protocol.Conversation{}, fmt.Errorf("get conversation %s: %w", id, err)
	}
	return conversation, nil
}

func (a *App) SaveClarify(ctx context.Context, request protocol.ClarifyRequest) error {
	return a.clarifies.Save(ctx, request)
}

func (a *App) UploadArtifact(ctx context.Context, artifact protocol.Artifact, content io.Reader) (protocol.Artifact, error) {
	if _, err := a.conversations.Get(ctx, artifact.ConversationID); err != nil {
		return protocol.Artifact{}, err
	}
	if a.files == nil {
		return protocol.Artifact{}, fmt.Errorf("artifact file store is not configured")
	}
	size, err := a.files.Save(artifact.StorageKey, content)
	if err != nil {
		return protocol.Artifact{}, err
	}
	artifact.Size = size
	return a.artifacts.Create(ctx, artifact)
}

func (a *App) GetArtifact(ctx context.Context, id string) (protocol.Artifact, error) {
	return a.artifacts.Get(ctx, id)
}

func (a *App) GetArtifactContent(ctx context.Context, id string) (protocol.Artifact, string, error) {
	artifact, reader, err := a.OpenArtifact(ctx, id)
	if err != nil {
		return protocol.Artifact{}, "", err
	}
	defer reader.Close()

	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return protocol.Artifact{}, "", err
	}
	return artifact, string(content), nil
}

func (a *App) ListArtifacts(ctx context.Context, conversationID string) ([]protocol.Artifact, error) {
	return a.artifacts.ListByConversation(ctx, conversationID)
}

func (a *App) OpenArtifact(ctx context.Context, id string) (protocol.Artifact, io.ReadCloser, error) {
	artifact, err := a.artifacts.Get(ctx, id)
	if err != nil {
		return protocol.Artifact{}, nil, err
	}
	if a.files == nil {
		return protocol.Artifact{}, nil, fmt.Errorf("artifact file store is not configured")
	}
	reader, err := a.files.Open(artifact.StorageKey)
	if err != nil {
		return protocol.Artifact{}, nil, err
	}
	return artifact, reader, nil
}

func (a *App) DeleteArtifact(ctx context.Context, id string) error {
	artifact, err := a.artifacts.Get(ctx, id)
	if err != nil {
		return err
	}
	if a.files == nil {
		return fmt.Errorf("artifact file store is not configured")
	}
	if err := a.files.Delete(artifact.StorageKey); err != nil {
		return err
	}
	return a.artifacts.Delete(ctx, id)
}

func (a *App) CreateTextArtifact(ctx context.Context, artifact protocol.Artifact, content string) (protocol.Artifact, error) {
	if _, err := a.conversations.Get(ctx, artifact.ConversationID); err != nil {
		return protocol.Artifact{}, err
	}
	if a.files == nil {
		return protocol.Artifact{}, fmt.Errorf("artifact file store is not configured")
	}
	size, err := a.files.Save(artifact.StorageKey, strings.NewReader(content))
	if err != nil {
		return protocol.Artifact{}, err
	}
	artifact.Size = size
	return a.artifacts.Create(ctx, artifact)
}

func (a *App) UpdateTextArtifact(ctx context.Context, artifact protocol.Artifact, content string) (protocol.Artifact, error) {
	if _, err := a.conversations.Get(ctx, artifact.ConversationID); err != nil {
		return protocol.Artifact{}, err
	}
	if a.files == nil {
		return protocol.Artifact{}, fmt.Errorf("artifact file store is not configured")
	}
	size, err := a.files.Save(artifact.StorageKey, strings.NewReader(content))
	if err != nil {
		return protocol.Artifact{}, err
	}
	artifact.Size = size
	return a.artifacts.Update(ctx, artifact)
}

func isTerminalRun(status protocol.RunStatus) bool {
	return status == protocol.RunCompleted || status == protocol.RunFailed || status == protocol.RunCancelled
}
