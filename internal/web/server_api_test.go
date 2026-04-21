package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	controlservice "mockinterview/internal/control/service"
	"mockinterview/internal/protocol"
	statebootstrap "mockinterview/internal/state/bootstrap"
)

func TestServerConversationTaskFlowSanitizesModelConfig(t *testing.T) {
	t.Parallel()

	conversations := &apiTestConversationRepo{items: map[string]protocol.Conversation{}}
	tasks := &apiTestTaskRepo{items: map[string]protocol.Task{}}
	server := newAPITestServer(conversations, tasks, &apiTestRunRepo{}, &apiTestMessageRepo{}, &apiTestEventRepo{}, &apiTestProfileRepo{})

	createdConversation := performJSON[protocol.Conversation](t, server, http.MethodPost, "/api/conversations", CreateConversationRequest{
		Title: "Go Interview",
	}, http.StatusCreated)

	createdTask := performJSON[protocol.Task](t, server, http.MethodPost, "/api/tasks", CreateTaskRequest{
		ConversationID: createdConversation.ID,
		Title:          "Agent Deep Dive",
		Prompt:         "请继续追问并发和超时设计。",
		Config:         protocol.InterviewConfig{Skill: "go-agent"},
		ModelConfig: protocol.ModelConfig{
			Provider: "openai-compatible",
			Model:    "gpt-test",
			APIKey:   "secret-key",
			BaseURL:  "https://example.com/v1",
		},
	}, http.StatusCreated)

	if createdTask.ModelConfig.APIKey != "" {
		t.Fatalf("expected API key to be sanitized in task response, got %q", createdTask.ModelConfig.APIKey)
	}
	if createdTask.ModelConfig.Provider != "openai-compatible" || createdTask.ModelConfig.Model != "gpt-test" {
		t.Fatalf("expected provider/model to be preserved, got %#v", createdTask.ModelConfig)
	}
	if createdTask.Config.Level == "" || createdTask.Config.Mode == "" {
		t.Fatalf("expected task config defaults to be applied, got %#v", createdTask.Config)
	}

	detail := performJSON[ConversationDetail](t, server, http.MethodGet, "/api/conversations/"+createdConversation.ID, nil, http.StatusOK)

	if detail.Conversation.CurrentTask != createdTask.ID {
		t.Fatalf("expected current task to be updated, got %#v", detail.Conversation)
	}
	if len(detail.Tasks) != 1 {
		t.Fatalf("expected 1 task in conversation detail, got %#v", detail.Tasks)
	}
	if detail.Tasks[0].ModelConfig.APIKey != "" {
		t.Fatalf("expected API key to stay sanitized in conversation detail, got %#v", detail.Tasks[0].ModelConfig)
	}
}

func TestServerRunAndReviewEndpointsReturnStoredData(t *testing.T) {
	t.Parallel()

	now := time.Now()
	conversation := protocol.Conversation{
		ID:        "conv_review",
		Title:     "Run Review",
		Status:    "active",
		CreatedAt: now.Add(-10 * time.Minute),
		UpdatedAt: now.Add(-5 * time.Minute),
	}
	task := protocol.Task{
		ID:             "task_review",
		ConversationID: conversation.ID,
		Title:          "Review Task",
		Prompt:         "请继续面试",
		Config:         protocol.InterviewConfig{Skill: "go-agent"}.WithDefaults(),
		CreatedAt:      now.Add(-9 * time.Minute),
		UpdatedAt:      now.Add(-5 * time.Minute),
	}
	run := protocol.Run{
		ID:             "run_review",
		ConversationID: conversation.ID,
		TaskID:         task.ID,
		Status:         protocol.RunCompleted,
		Phase:          protocol.RunPhaseEvaluating,
		Input:          task.Prompt,
		Output:         "这里是回答",
		CreatedAt:      now.Add(-8 * time.Minute),
		UpdatedAt:      now.Add(-2 * time.Minute),
		InterviewState: &protocol.RunInterviewState{Phase: protocol.PhaseWrapup, Round: 3},
	}
	review := protocol.ReviewSnapshot{
		RunID:       run.ID,
		GeneratedAt: now.Add(-1 * time.Minute),
		Scorecard: &protocol.Scorecard{
			Title:   "Go Agent Review",
			Summary: "回答总体不错，但 tradeoff 还可以更明确。",
		},
	}

	server := newAPITestServer(
		&apiTestConversationRepo{items: map[string]protocol.Conversation{conversation.ID: conversation}},
		&apiTestTaskRepo{items: map[string]protocol.Task{task.ID: task}},
		&apiTestRunRepo{items: map[string]protocol.Run{run.ID: run}},
		&apiTestMessageRepo{byRun: map[string][]protocol.Message{
			run.ID: {
				{ID: "m1", RunID: run.ID, Role: "user", Content: "我会先用 errgroup 控制并发。", CreatedAt: now.Add(-7 * time.Minute)},
				{ID: "m2", RunID: run.ID, Role: "assistant", Content: "那失败隔离怎么做？", CreatedAt: now.Add(-6 * time.Minute)},
			},
		}},
		&apiTestEventRepo{byRun: map[string][]protocol.Event{
			run.ID: {
				{ID: "e1", RunID: run.ID, Type: protocol.EventReviewGenerated, Timestamp: review.GeneratedAt, Payload: review},
			},
		}},
		&apiTestProfileRepo{},
	)

	runDetail := performJSON[RunDetail](t, server, http.MethodGet, "/api/runs/"+run.ID, nil, http.StatusOK)
	if runDetail.Run.ID != run.ID {
		t.Fatalf("expected run detail for %s, got %#v", run.ID, runDetail.Run)
	}
	if len(runDetail.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %#v", runDetail.Messages)
	}
	if len(runDetail.Events) != 1 || runDetail.Events[0].Type != protocol.EventReviewGenerated {
		t.Fatalf("expected review event to be returned, got %#v", runDetail.Events)
	}

	reviewDetail := performJSON[RunReviewDetail](t, server, http.MethodGet, "/api/runs/"+run.ID+"/review", nil, http.StatusOK)
	if reviewDetail.Review.RunID != run.ID {
		t.Fatalf("expected review for run %s, got %#v", run.ID, reviewDetail.Review)
	}
	if reviewDetail.Review.Scorecard == nil || reviewDetail.Review.Scorecard.Title != "Go Agent Review" {
		t.Fatalf("expected scorecard in review response, got %#v", reviewDetail.Review.Scorecard)
	}
}

func TestServerHealthAndProfileEndpointsReturnStructuredPayloads(t *testing.T) {
	t.Parallel()

	now := time.Now()
	completedAt := now.Add(-2 * time.Minute)
	server := newAPITestServer(
		&apiTestConversationRepo{items: map[string]protocol.Conversation{
			"conv_health": {
				ID:        "conv_health",
				Title:     "Health",
				Status:    "active",
				CreatedAt: now.Add(-20 * time.Minute),
				UpdatedAt: now.Add(-1 * time.Minute),
			},
			"conv_deleted": {
				ID:        "conv_deleted",
				Title:     "Deleted",
				Status:    "deleted",
				CreatedAt: now.Add(-25 * time.Minute),
				UpdatedAt: now.Add(-15 * time.Minute),
			},
		}},
		&apiTestTaskRepo{},
		&apiTestRunRepo{items: map[string]protocol.Run{
			"run_ok": {
				ID:             "run_ok",
				ConversationID: "conv_health",
				TaskID:         "task_ok",
				Status:         protocol.RunCompleted,
				Input:          "请模拟一场 Go 面试",
				Output:         "这里是完整回答",
				CreatedAt:      now.Add(-10 * time.Minute),
				UpdatedAt:      completedAt,
				CompletedAt:    &completedAt,
			},
			"run_failed": {
				ID:             "run_failed",
				ConversationID: "conv_health",
				TaskID:         "task_failed",
				Status:         protocol.RunFailed,
				Input:          "继续",
				CreatedAt:      now.Add(-8 * time.Minute),
				UpdatedAt:      now.Add(-6 * time.Minute),
			},
			"run_deleted": {
				ID:             "run_deleted",
				ConversationID: "conv_deleted",
				TaskID:         "task_deleted",
				Status:         protocol.RunCompleted,
				Input:          "历史会话",
				Output:         "已软删",
				CreatedAt:      now.Add(-18 * time.Minute),
				UpdatedAt:      now.Add(-16 * time.Minute),
			},
		}},
		&apiTestMessageRepo{byRun: map[string][]protocol.Message{
			"run_ok": {
				{RunID: "run_ok", Content: "用户回答包括 goroutine、timeout、trace"},
			},
			"run_failed": {
				{RunID: "run_failed", Content: "这里也会算进 token 估算"},
			},
		}},
		&apiTestEventRepo{},
		&apiTestProfileRepo{items: map[string]protocol.CandidateProfile{
			"global": {
				ID:               "global",
				InterviewCount:   4,
				RecurringGaps:    []string{"observability"},
				RecommendedFocus: []string{"tradeoff"},
				UpdatedAt:        now.Add(-30 * time.Minute),
			},
		}},
	)

	metrics := performJSON[protocol.HealthMetrics](t, server, http.MethodGet, "/api/health", nil, http.StatusOK)
	if metrics.RunCount != 2 || metrics.CompletedRuns != 1 || metrics.FailedRuns != 1 {
		t.Fatalf("unexpected health metrics payload: %#v", metrics)
	}
	if metrics.VisibleConversationCount != 1 || metrics.DeletedConversationCount != 1 {
		t.Fatalf("unexpected conversation counts: %#v", metrics)
	}
	if metrics.StoredRunCount != 3 || metrics.StoredCompletedRuns != 2 || metrics.StoredTerminalRuns != 3 {
		t.Fatalf("unexpected stored run counts: %#v", metrics)
	}
	if metrics.SuccessRate != 0.5 {
		t.Fatalf("expected success rate 0.5, got %#v", metrics)
	}

	profile := performJSON[CandidateProfileResponse](t, server, http.MethodGet, "/api/profile", nil, http.StatusOK)
	if profile.Profile.ID != "global" {
		t.Fatalf("expected global profile, got %#v", profile.Profile)
	}
	if len(profile.Profile.RecurringGaps) != 1 || profile.Profile.RecurringGaps[0] != "observability" {
		t.Fatalf("unexpected profile payload: %#v", profile.Profile)
	}
}

func newAPITestServer(
	conversations controlservice.ConversationRepository,
	tasks controlservice.TaskRepository,
	runs controlservice.RunRepository,
	messages controlservice.MessageRepository,
	events controlservice.EventRepository,
	profiles controlservice.ProfileRepository,
) *Server {
	deps, err := statebootstrap.NewAppDependencies(
		apiTestCheckpointRepo{},
		apiTestMemoryRepo{},
		&apiTestArtifactRepo{},
	)
	if err != nil {
		panic(err)
	}
	app, err := controlservice.NewApp(
		conversations,
		tasks,
		runs,
		messages,
		events,
		profiles,
		apiTestCheckpointRepo{},
		apiTestClarifyRepo{},
		apiTestMemoryRepo{},
		&apiTestArtifactRepo{},
		apiTestFileStore{},
		deps,
	)
	if err != nil {
		panic(err)
	}
	server := &Server{
		app: app,
		mux: http.NewServeMux(),
	}
	server.registerRoutes()
	return server
}

func performJSON[T any](t *testing.T, server *Server, method, path string, body any, wantStatus int) T {
	t.Helper()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(payload)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()

	server.mux.ServeHTTP(recorder, req)

	if recorder.Code != wantStatus {
		t.Fatalf("expected status %d, got %d: %s", wantStatus, recorder.Code, recorder.Body.String())
	}

	var result T
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response body: %v; body=%s", err, recorder.Body.String())
	}
	return result
}

type apiTestConversationRepo struct {
	items map[string]protocol.Conversation
}

func (r *apiTestConversationRepo) Create(ctx context.Context, conversation protocol.Conversation) (protocol.Conversation, error) {
	_ = ctx
	if r.items == nil {
		r.items = make(map[string]protocol.Conversation)
	}
	r.items[conversation.ID] = conversation
	return conversation, nil
}

func (r *apiTestConversationRepo) Update(ctx context.Context, conversation protocol.Conversation) (protocol.Conversation, error) {
	_ = ctx
	if r.items == nil {
		r.items = make(map[string]protocol.Conversation)
	}
	r.items[conversation.ID] = conversation
	return conversation, nil
}

func (r *apiTestConversationRepo) Get(ctx context.Context, id string) (protocol.Conversation, error) {
	_ = ctx
	conversation, ok := r.items[id]
	if !ok {
		return protocol.Conversation{}, fmt.Errorf("conversation %s not found", id)
	}
	return conversation, nil
}

func (r *apiTestConversationRepo) List(ctx context.Context) ([]protocol.Conversation, error) {
	_ = ctx
	items := make([]protocol.Conversation, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	return items, nil
}

type apiTestTaskRepo struct {
	items map[string]protocol.Task
}

func (r *apiTestTaskRepo) Create(ctx context.Context, task protocol.Task) (protocol.Task, error) {
	_ = ctx
	if r.items == nil {
		r.items = make(map[string]protocol.Task)
	}
	r.items[task.ID] = task
	return task, nil
}

func (r *apiTestTaskRepo) Get(ctx context.Context, id string) (protocol.Task, error) {
	_ = ctx
	task, ok := r.items[id]
	if !ok {
		return protocol.Task{}, fmt.Errorf("task %s not found", id)
	}
	return task, nil
}

func (r *apiTestTaskRepo) ListByConversation(ctx context.Context, conversationID string) ([]protocol.Task, error) {
	_ = ctx
	tasks := make([]protocol.Task, 0, len(r.items))
	for _, task := range r.items {
		if task.ConversationID == conversationID {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

type apiTestRunRepo struct {
	items map[string]protocol.Run
}

func (r *apiTestRunRepo) Create(ctx context.Context, run protocol.Run) (protocol.Run, error) {
	_ = ctx
	if r.items == nil {
		r.items = make(map[string]protocol.Run)
	}
	r.items[run.ID] = run
	return run, nil
}

func (r *apiTestRunRepo) Update(ctx context.Context, run protocol.Run) (protocol.Run, error) {
	_ = ctx
	if r.items == nil {
		r.items = make(map[string]protocol.Run)
	}
	r.items[run.ID] = run
	return run, nil
}

func (r *apiTestRunRepo) Get(ctx context.Context, id string) (protocol.Run, error) {
	_ = ctx
	run, ok := r.items[id]
	if !ok {
		return protocol.Run{}, fmt.Errorf("run %s not found", id)
	}
	return run, nil
}

func (r *apiTestRunRepo) ListByConversation(ctx context.Context, conversationID string) ([]protocol.Run, error) {
	_ = ctx
	runs := make([]protocol.Run, 0, len(r.items))
	for _, run := range r.items {
		if run.ConversationID == conversationID {
			runs = append(runs, run)
		}
	}
	return runs, nil
}

type apiTestMessageRepo struct {
	byRun map[string][]protocol.Message
}

func (r *apiTestMessageRepo) Create(ctx context.Context, message protocol.Message) (protocol.Message, error) {
	_ = ctx
	if r.byRun == nil {
		r.byRun = make(map[string][]protocol.Message)
	}
	r.byRun[message.RunID] = append(r.byRun[message.RunID], message)
	return message, nil
}

func (r *apiTestMessageRepo) ListByRun(ctx context.Context, runID string) ([]protocol.Message, error) {
	_ = ctx
	return append([]protocol.Message(nil), r.byRun[runID]...), nil
}

type apiTestEventRepo struct {
	byRun map[string][]protocol.Event
}

func (r *apiTestEventRepo) Create(ctx context.Context, event protocol.Event) (protocol.Event, error) {
	_ = ctx
	if r.byRun == nil {
		r.byRun = make(map[string][]protocol.Event)
	}
	r.byRun[event.RunID] = append(r.byRun[event.RunID], event)
	return event, nil
}

func (r *apiTestEventRepo) ListByRun(ctx context.Context, runID string) ([]protocol.Event, error) {
	_ = ctx
	return append([]protocol.Event(nil), r.byRun[runID]...), nil
}

type apiTestProfileRepo struct {
	items map[string]protocol.CandidateProfile
}

func (r *apiTestProfileRepo) Get(ctx context.Context, id string) (protocol.CandidateProfile, error) {
	_ = ctx
	profile, ok := r.items[id]
	if !ok {
		return protocol.CandidateProfile{}, fmt.Errorf("profile %s not found", id)
	}
	return profile, nil
}

func (r *apiTestProfileRepo) Save(ctx context.Context, profile protocol.CandidateProfile) (protocol.CandidateProfile, error) {
	_ = ctx
	if r.items == nil {
		r.items = make(map[string]protocol.CandidateProfile)
	}
	r.items[profile.ID] = profile
	return profile, nil
}

type apiTestCheckpointRepo struct{}

func (apiTestCheckpointRepo) Save(context.Context, protocol.CheckpointSnapshot) error {
	return nil
}

func (apiTestCheckpointRepo) Load(context.Context, string) (protocol.CheckpointSnapshot, error) {
	return protocol.CheckpointSnapshot{}, fmt.Errorf("checkpoint not found")
}

type apiTestClarifyRepo struct{}

func (apiTestClarifyRepo) Save(context.Context, protocol.ClarifyRequest) error {
	return nil
}

func (apiTestClarifyRepo) GetPending(context.Context, string) (protocol.ClarifyRequest, error) {
	return protocol.ClarifyRequest{}, fmt.Errorf("clarify request not found")
}

func (apiTestClarifyRepo) Resolve(context.Context, string) error {
	return nil
}

type apiTestMemoryRepo struct{}

func (apiTestMemoryRepo) Append(context.Context, protocol.MemoryRecord) error {
	return nil
}

func (apiTestMemoryRepo) List(context.Context, string) ([]protocol.MemoryRecord, error) {
	return nil, nil
}

type apiTestArtifactRepo struct {
	items map[string]protocol.Artifact
}

func (r *apiTestArtifactRepo) Create(ctx context.Context, artifact protocol.Artifact) (protocol.Artifact, error) {
	if r.items == nil {
		r.items = make(map[string]protocol.Artifact)
	}
	r.items[artifact.ID] = artifact
	return artifact, nil
}

func (r *apiTestArtifactRepo) Update(ctx context.Context, artifact protocol.Artifact) (protocol.Artifact, error) {
	if r.items == nil {
		r.items = make(map[string]protocol.Artifact)
	}
	r.items[artifact.ID] = artifact
	return artifact, nil
}

func (r *apiTestArtifactRepo) Get(ctx context.Context, id string) (protocol.Artifact, error) {
	artifact, ok := r.items[id]
	if !ok {
		return protocol.Artifact{}, fmt.Errorf("artifact %s not found", id)
	}
	return artifact, nil
}

func (r *apiTestArtifactRepo) ListByConversation(ctx context.Context, conversationID string) ([]protocol.Artifact, error) {
	items := make([]protocol.Artifact, 0, len(r.items))
	for _, item := range r.items {
		if item.ConversationID == conversationID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (r *apiTestArtifactRepo) Delete(ctx context.Context, id string) error {
	if r.items != nil {
		delete(r.items, id)
	}
	return nil
}

type apiTestFileStore struct{}

func (apiTestFileStore) Save(string, io.Reader) (int64, error) {
	return 0, nil
}

func (apiTestFileStore) Open(string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (apiTestFileStore) Delete(string) error {
	return nil
}
