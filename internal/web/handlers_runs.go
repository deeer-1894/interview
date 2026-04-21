package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"mockinterview/internal/protocol"
)

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: ErrMethodNotAllowed.Error()})
		return
	}

	var req CreateRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	run, err := s.app.CreateRun(r.Context(), protocol.RunRequest{
		TaskID:      req.TaskID,
		Prompt:      req.Prompt,
		ArtifactIDs: req.ArtifactIDs,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

func (s *Server) handleRunByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/runs/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "run ID required"})
		return
	}
	runID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch {
	case action == "" && r.Method == http.MethodGet:
		s.handleGetRun(w, r, runID)
	case action == "review" && r.Method == http.MethodGet:
		s.handleGetRunReview(w, r, runID)
	case action == "resume" && r.Method == http.MethodPost:
		s.handleResumeRun(w, r, runID)
	case action == "copilot" && r.Method == http.MethodPost:
		s.handleRunCopilot(w, r, runID)
	case action == "cancel" && r.Method == http.MethodPost:
		s.handleCancelRun(w, r, runID)
	case action == "events" && r.Method == http.MethodGet:
		s.handleRunEvents(w, r, runID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: ErrMethodNotAllowed.Error()})
	}
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request, runID string) {
	run, messages, events, err := s.app.GetRun(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
		return
	}
	if messages == nil {
		messages = make([]protocol.Message, 0)
	}
	if events == nil {
		events = make([]protocol.Event, 0)
	}
	writeJSON(w, http.StatusOK, RunDetail{
		Run:      run,
		Messages: messages,
		Events:   events,
	})
}

func (s *Server) handleGetRunReview(w http.ResponseWriter, r *http.Request, runID string) {
	review, err := s.app.GetReviewSnapshot(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, RunReviewDetail{Review: review})
}

func (s *Server) handleResumeRun(w http.ResponseWriter, r *http.Request, runID string) {
	var req ResumeRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}
	run, err := s.app.ResumeRun(r.Context(), runID, protocol.ResumeInput{
		Message:     req.Message,
		Config:      req.Config,
		ArtifactIDs: req.ArtifactIDs,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) handleRunCopilot(w http.ResponseWriter, r *http.Request, runID string) {
	result, err := s.app.RequestCopilotHint(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, CopilotAssistResponse{
		Feedback: result.Feedback,
		Hint:     result.Hint,
		Events:   result.Events,
	})
}

func (s *Server) handleCancelRun(w http.ResponseWriter, r *http.Request, runID string) {
	run, err := s.app.CancelRun(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) handleRunEvents(w http.ResponseWriter, r *http.Request, runID string) {
	sse := newSSEWriter(w)
	if sse == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: ErrStreamingNotSupported.Error()})
		return
	}

	run, _, events, err := s.app.GetRun(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
		return
	}

	for _, event := range events {
		sse.SendJSON("event", event)
	}
	if shouldCloseEventStream(run.Status) {
		return
	}

	eventsCh, cancel, err := s.app.Subscribe(runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
		return
	}
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-eventsCh:
			if !ok {
				return
			}
			sse.SendJSON("event", event)
			if event.Type == protocol.EventRunCompleted || event.Type == protocol.EventRunFailed || event.Type == protocol.EventRunCancelled {
				return
			}
			if event.Type == protocol.EventClarifyRequested || event.Type == protocol.EventMessageCompleted {
				currentRun, _, _, runErr := s.app.GetRun(r.Context(), runID)
				if runErr == nil && currentRun.Status == protocol.RunWaitingClarify {
					return
				}
			}
		case <-ticker.C:
			sse.SendJSON("event", protocol.Event{
				ID:        fmt.Sprintf("heartbeat_%d", time.Now().UnixNano()),
				RunID:     runID,
				Type:      protocol.EventHeartbeat,
				Timestamp: time.Now(),
				Payload: map[string]string{
					"status": "alive",
				},
			})
		}
	}
}

func shouldCloseEventStream(status protocol.RunStatus) bool {
	return status == protocol.RunWaitingClarify || isTerminal(status)
}
