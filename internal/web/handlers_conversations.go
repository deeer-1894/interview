package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"mockinterview/internal/protocol"
)

func (s *Server) handleConversations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		conversations, err := s.app.ListConversations(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, conversations)
	case http.MethodPost:
		var req CreateConversationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, context.Canceled) {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
			return
		}
		conversation, err := s.app.CreateConversation(r.Context(), req.Title)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, conversation)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: ErrMethodNotAllowed.Error()})
	}
}

func (s *Server) handleConversationByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/conversations/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "conversation ID required"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		conversation, tasks, runs, err := s.app.GetConversation(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
			return
		}
		if tasks == nil {
			tasks = make([]protocol.Task, 0)
		}
		if runs == nil {
			runs = make([]protocol.Run, 0)
		}
		writeJSON(w, http.StatusOK, ConversationDetail{
			Conversation: conversation,
			Tasks:        sanitizeTasksForClient(tasks),
			Runs:         runs,
		})
	case http.MethodPatch:
		var req UpdateConversationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
			return
		}
		conversation, err := s.app.UpdateConversation(r.Context(), id, req.Title, req.Pinned, req.Archived)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, conversation)
	case http.MethodDelete:
		conversation, err := s.app.DeleteConversation(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, conversation)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: ErrMethodNotAllowed.Error()})
	}
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: ErrMethodNotAllowed.Error()})
		return
	}

	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	req.Config = req.Config.WithDefaults()
	req.ModelConfig = withDefaultModelConfig(req.ModelConfig, toProtocolModelConfig(s.modelCfg))
	task, err := s.app.CreateTask(r.Context(), req.ConversationID, req.Title, req.Prompt, req.ArtifactIDs, req.Config, req.ModelConfig)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, sanitizeTaskForClient(task))
}
