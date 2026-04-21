package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"mockinterview/internal/protocol"
)

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		conversationID := strings.TrimSpace(r.URL.Query().Get("conversationId"))
		if conversationID == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "conversationId is required"})
			return
		}
		artifacts, err := s.app.ListArtifacts(r.Context(), conversationID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		if artifacts == nil {
			artifacts = make([]protocol.Artifact, 0)
		}
		writeJSON(w, http.StatusOK, ArtifactListResponse{Artifacts: artifacts})
	case http.MethodPost:
		if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			s.handleUploadFile(w, r)
			return
		}
		s.handleSaveTextFile(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: ErrMethodNotAllowed.Error()})
	}
}

func (s *Server) handleFileByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/files/")
	id = strings.Trim(id, "/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "file ID required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		if r.URL.Query().Get("download") == "1" {
			s.handleDownloadFile(w, r, id)
			return
		}
		if r.URL.Query().Get("content") == "1" {
			s.handleGetFileContent(w, r, id)
			return
		}
		artifact, err := s.app.GetArtifact(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, artifact)
	case http.MethodPut:
		s.handleUpdateTextFile(w, r, id)
	case http.MethodDelete:
		s.handleDeleteFile(w, r, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: ErrMethodNotAllowed.Error()})
	}
}

func (s *Server) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid multipart form"})
		return
	}

	conversationID := strings.TrimSpace(r.FormValue("conversationId"))
	if conversationID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "conversationId is required"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "file is required"})
		return
	}
	defer file.Close()

	artifact, err := s.app.UploadArtifactFile(
		r.Context(),
		conversationID,
		strings.TrimSpace(r.FormValue("taskId")),
		strings.TrimSpace(r.FormValue("runId")),
		header.Filename,
		header.Header.Get("Content-Type"),
		file,
	)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, artifact)
}

func (s *Server) handleSaveTextFile(w http.ResponseWriter, r *http.Request) {
	var req SaveArtifactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	artifact, err := s.app.CreateTextArtifactContent(
		r.Context(),
		req.ConversationID,
		req.TaskID,
		req.RunID,
		req.Name,
		req.ContentType,
		req.Content,
	)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, artifact)
}

func (s *Server) handleUpdateTextFile(w http.ResponseWriter, r *http.Request, id string) {
	var req SaveArtifactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	artifact, err := s.app.UpdateTextArtifactContent(
		r.Context(),
		id,
		req.Name,
		req.ContentType,
		req.TaskID,
		req.RunID,
		req.Content,
	)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, artifact)
}

func (s *Server) handleGetFileContent(w http.ResponseWriter, r *http.Request, id string) {
	artifact, content, err := s.app.GetArtifactContent(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
		return
	}
	if !isEditableTextArtifact(artifact) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "only text artifacts can be edited inline"})
		return
	}
	writeJSON(w, http.StatusOK, ArtifactDetailResponse{
		Artifact: artifact,
		Content:  content,
	})
}

func (s *Server) handleDeleteFile(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.app.DeleteArtifact(r.Context(), id); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleDownloadFile(w http.ResponseWriter, r *http.Request, id string) {
	artifact, reader, err := s.app.OpenArtifact(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", artifact.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", artifact.Name))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", artifact.Size))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}
