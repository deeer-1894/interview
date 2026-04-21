package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	controlservice "mockinterview/internal/control/service"
	domain "mockinterview/internal/interview"
	"mockinterview/internal/protocol"
	statebootstrap "mockinterview/internal/state/bootstrap"
	artifactstorage "mockinterview/internal/storage/artifacts"
)

type Server struct {
	modelCfg interviewModelConfig
	app      *controlservice.App
	mux      *http.ServeMux
	helper   requestHelper
	closeFn  func(context.Context) error
}

type interviewModelConfig = domain.ModelConfig

type InterviewRequest struct {
	Prompt      string                   `json:"prompt"`
	Config      protocol.InterviewConfig `json:"config"`
	ModelConfig *ClientModelConfig       `json:"modelConfig,omitempty"`
}

type ClientModelConfig struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"apiKey,omitempty"`
	BaseURL  string `json:"baseUrl,omitempty"`
}

type InterviewResponse struct {
	Content        string                   `json:"content"`
	Config         protocol.InterviewConfig `json:"config"`
	ModelInfo      ModelInfo                `json:"modelInfo"`
	ConversationID string                   `json:"conversationId"`
	TaskID         string                   `json:"taskId"`
	RunID          string                   `json:"runId"`
}

type ModelInfo struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	BaseURL  string `json:"baseUrl,omitempty"`
}

type CreateConversationRequest struct {
	Title string `json:"title"`
}

type UpdateConversationRequest struct {
	Title    *string `json:"title,omitempty"`
	Pinned   *bool   `json:"pinned,omitempty"`
	Archived *bool   `json:"archived,omitempty"`
}

type CreateTaskRequest struct {
	ConversationID string                   `json:"conversationId"`
	Title          string                   `json:"title"`
	Prompt         string                   `json:"prompt"`
	ArtifactIDs    []string                 `json:"artifactIds,omitempty"`
	Config         protocol.InterviewConfig `json:"config"`
	ModelConfig    protocol.ModelConfig     `json:"modelConfig"`
}

type CreateRunRequest struct {
	TaskID      string   `json:"taskId"`
	Prompt      string   `json:"prompt,omitempty"`
	ArtifactIDs []string `json:"artifactIds,omitempty"`
}

type ResumeRunRequest struct {
	Message     string                   `json:"message"`
	Config      protocol.InterviewConfig `json:"config,omitempty"`
	ArtifactIDs []string                 `json:"artifactIds,omitempty"`
}

type CopilotAssistResponse struct {
	Feedback protocol.CopilotFeedback `json:"feedback"`
	Hint     protocol.CopilotHint     `json:"hint"`
	Events   []protocol.Event         `json:"events,omitempty"`
}

type ConversationDetail struct {
	Conversation protocol.Conversation `json:"conversation"`
	Tasks        []protocol.Task       `json:"tasks"`
	Runs         []protocol.Run        `json:"runs"`
}

type CandidateProfileResponse struct {
	Profile protocol.CandidateProfile `json:"profile"`
}

type RunDetail struct {
	Run      protocol.Run       `json:"run"`
	Messages []protocol.Message `json:"messages"`
	Events   []protocol.Event   `json:"events"`
}

type RunReviewDetail struct {
	Review protocol.ReviewSnapshot `json:"review"`
}

type ArtifactListResponse struct {
	Artifacts []protocol.Artifact `json:"artifacts"`
}

type ArtifactDetailResponse struct {
	Artifact protocol.Artifact `json:"artifact"`
	Content  string            `json:"content"`
}

type SkillListResponse struct {
	Skills []controlservice.SkillMetadata `json:"skills"`
}

type SkillDetailResponse struct {
	Skill controlservice.SkillDocument `json:"skill"`
}

type SaveSkillRequest struct {
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	Version              string   `json:"version,omitempty"`
	FocusAreas           []string `json:"focusAreas,omitempty"`
	ComposedOf           []string `json:"composedOf,omitempty"`
	CapabilityBoundaries []string `json:"capabilityBoundaries,omitempty"`
	SampleQuestions      []string `json:"sampleQuestions,omitempty"`
	FollowUps            []string `json:"followUps,omitempty"`
	Scenarios            []string `json:"scenarios,omitempty"`
	Adversarial          []string `json:"adversarial,omitempty"`
	Pressure             []string `json:"pressure,omitempty"`
	ScoringAnchors       []string `json:"scoringAnchors,omitempty"`
	InstallSource        string   `json:"installSource,omitempty"`
	SourceURL            string   `json:"sourceUrl,omitempty"`
	Rating               float64  `json:"rating,omitempty"`
	RatingCount          int      `json:"ratingCount,omitempty"`
	Content              string   `json:"content"`
}

type SaveArtifactRequest struct {
	ConversationID string `json:"conversationId"`
	TaskID         string `json:"taskId,omitempty"`
	RunID          string `json:"runId,omitempty"`
	Name           string `json:"name"`
	ContentType    string `json:"contentType,omitempty"`
	Content        string `json:"content"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusCapturingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusCapturingResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func sanitizeModelConfigForClient(cfg protocol.ModelConfig) protocol.ModelConfig {
	cfg.APIKey = ""
	return cfg
}

func sanitizeTaskForClient(task protocol.Task) protocol.Task {
	task.ModelConfig = sanitizeModelConfigForClient(task.ModelConfig)
	return task
}

func sanitizeTasksForClient(tasks []protocol.Task) []protocol.Task {
	if len(tasks) == 0 {
		return tasks
	}
	sanitized := make([]protocol.Task, len(tasks))
	for index, task := range tasks {
		sanitized[index] = sanitizeTaskForClient(task)
	}
	return sanitized
}

func NewServer(modelCfg domain.ModelConfig) (*Server, error) {
	bundle, err := statebootstrap.NewFromEnv(context.Background())
	if err != nil {
		return nil, fmt.Errorf("bootstrap persistent state: %w", err)
	}
	s := &Server{
		modelCfg: modelCfg.WithDefaults(),
		mux:      http.NewServeMux(),
		closeFn:  bundle.Close,
	}
	s.app, err = statebootstrap.NewApp(bundle, artifactstorage.New(os.Getenv("ARTIFACT_STORAGE_DIR")))
	if err != nil {
		_ = bundle.Close(context.Background())
		return nil, fmt.Errorf("build app dependencies: %w", err)
	}
	if err := s.app.RecoverInterruptedRuns(context.Background()); err != nil {
		_ = bundle.Close(context.Background())
		return nil, fmt.Errorf("recover interrupted runs: %w", err)
	}
	s.registerRoutes()
	return s, nil
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/interview", s.handleInterview)
	s.mux.HandleFunc("/api/conversations", s.handleConversations)
	s.mux.HandleFunc("/api/conversations/", s.handleConversationByID)
	s.mux.HandleFunc("/api/tasks", s.handleTasks)
	s.mux.HandleFunc("/api/runs", s.handleRuns)
	s.mux.HandleFunc("/api/runs/", s.handleRunByID)
	s.mux.HandleFunc("/api/profile", s.handleProfile)
	s.mux.HandleFunc("/api/skills", s.handleSkills)
	s.mux.HandleFunc("/api/skills/", s.handleSkillByName)
	s.mux.HandleFunc("/api/files", s.handleFiles)
	s.mux.HandleFunc("/api/files/", s.handleFileByID)
	s.mux.Handle("/", s.handleApp())
}

func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           s.withCORS(s.withLogging(s.mux)),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		<-ctx.Done()
		if s.closeFn != nil {
			if err := s.closeFn(context.Background()); err != nil {
				slog.Error("close_state_bundle_failed", "error", err)
				select {
				case errCh <- protocol.WrapInterviewError("shutdown", "close_state_bundle", false, err):
				default:
				}
				return
			}
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		errCh <- server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	default:
	}

	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	metrics, err := s.app.GetHealthMetrics(context.Background())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

func (s *Server) handleInterview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: ErrMethodNotAllowed.Error()})
		return
	}

	var req InterviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	req.Config = req.Config.WithDefaults()
	if strings.TrimSpace(req.Prompt) == "" {
		req.Prompt = "请模拟一场 agent 开发岗位的技术面试。"
	}
	modelCfg := s.helper.parseModelConfig(toProtocolModelConfig(s.modelCfg), req.ModelConfig)

	conversation, err := s.app.CreateConversation(r.Context(), "Legacy Interview")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	task, err := s.app.CreateTask(r.Context(), conversation.ID, "", req.Prompt, nil, req.Config, modelCfg)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	run, err := s.app.CreateRun(r.Context(), protocol.RunRequest{
		TaskID:         task.ID,
		ConversationID: conversation.ID,
		Prompt:         req.Prompt,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	content := s.waitForLegacyResponseReady(r.Context(), run.ID, 60*time.Second)

	writeJSON(w, http.StatusOK, InterviewResponse{
		Content:        content,
		Config:         task.Config,
		ConversationID: conversation.ID,
		TaskID:         task.ID,
		RunID:          run.ID,
		ModelInfo: ModelInfo{
			Provider: modelCfg.Provider,
			Model:    modelCfg.Model,
			BaseURL:  modelCfg.BaseURL,
		},
	})
}

func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: ErrMethodNotAllowed.Error()})
		return
	}
	profile, err := s.app.GetCandidateProfile(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, CandidateProfileResponse{Profile: profile})
}

func (s *Server) handleApp() http.Handler {
	distDir := filepath.Join("web", "dist")
	if dirInfo, err := os.Stat(distDir); err == nil && dirInfo.IsDir() {
		fs := http.FileServer(http.Dir(distDir))
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			target := filepath.Join(distDir, filepath.Clean(r.URL.Path))
			if info, err := os.Stat(target); err == nil && !info.IsDir() {
				fs.ServeHTTP(w, r)
				return
			}
			http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprintln(w, "frontend build not found. Run the web app from /web or build it into /web/dist.")
	})
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusCapturingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		slog.Info(
			"http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration_ms", time.Since(start).Round(time.Millisecond).Milliseconds(),
		)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func isTerminal(status protocol.RunStatus) bool {
	return status == protocol.RunCompleted || status == protocol.RunFailed || status == protocol.RunCancelled
}

func isEditableTextArtifact(artifact protocol.Artifact) bool {
	contentType := strings.ToLower(strings.TrimSpace(artifact.ContentType))
	if strings.HasPrefix(contentType, "text/") {
		return true
	}

	switch contentType {
	case "application/json", "application/ld+json", "application/xml", "application/yaml", "application/x-yaml":
		return true
	}

	switch strings.ToLower(filepath.Ext(artifact.Name)) {
	case ".md", ".txt", ".json", ".yaml", ".yml", ".xml", ".csv", ".tsv":
		return true
	default:
		return false
	}
}

func (s *Server) waitForRunCompletion(ctx context.Context, runID string, timeout time.Duration) (string, error) {
	deadline := time.After(timeout)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-deadline:
			return "", fmt.Errorf("run %s timed out", runID)
		case <-ticker.C:
			run, _, _, err := s.app.GetRun(ctx, runID)
			if err != nil {
				return "", err
			}
			switch run.Status {
			case protocol.RunCompleted:
				return run.Output, nil
			case protocol.RunFailed:
				if run.LastError == "" {
					run.LastError = "run failed"
				}
				return "", fmt.Errorf("%s", run.LastError)
			case protocol.RunWaitingClarify:
				return "", fmt.Errorf("run %s requires clarify before completion", runID)
			}
		}
	}
}

func (s *Server) waitForLegacyResponseReady(ctx context.Context, runID string, timeout time.Duration) string {
	deadline := time.After(timeout)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ""
		case <-deadline:
			run, _, _, err := s.app.GetRun(ctx, runID)
			if err != nil {
				return ""
			}
			return run.Output
		case <-ticker.C:
			run, _, _, err := s.app.GetRun(ctx, runID)
			if err != nil {
				return ""
			}
			review, reviewErr := s.app.GetReviewSnapshot(ctx, runID)
			if strings.TrimSpace(run.Output) != "" &&
				reviewErr == nil &&
				review.Decision != nil &&
				review.Trace != nil &&
				review.Trace.QuestionCount > 0 {
				return run.Output
			}
			switch run.Status {
			case protocol.RunCompleted, protocol.RunFailed, protocol.RunCancelled, protocol.RunWaitingClarify:
				return run.Output
			}
		}
	}
}

func toProtocolModelConfig(cfg domain.ModelConfig) protocol.ModelConfig {
	return protocol.ModelConfig{
		Provider: string(cfg.Provider),
		Model:    cfg.Model,
		APIKey:   cfg.APIKey,
		BaseURL:  cfg.BaseURL,
	}
}

func withDefaultModelConfig(cfg, defaults protocol.ModelConfig) protocol.ModelConfig {
	if strings.TrimSpace(cfg.Provider) == "" {
		cfg.Provider = defaults.Provider
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = defaults.Model
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		cfg.APIKey = defaults.APIKey
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = defaults.BaseURL
	}
	return cfg
}
