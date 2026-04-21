package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	controlservice "mockinterview/internal/control/service"
)

func (s *Server) handleSkills(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		skills, err := s.app.ListSkills(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, SkillListResponse{Skills: skills})
	case http.MethodPost:
		if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			s.handleUploadSkill(w, r)
			return
		}
		var req SaveSkillRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
			return
		}
		meta, err := s.app.SaveSkill(r.Context(), skillInputFromRequest(req), "")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, meta)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: ErrMethodNotAllowed.Error()})
	}
}

func (s *Server) handleSkillByName(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/skills/")
	name = strings.TrimSpace(strings.Trim(name, "/"))
	if name == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "skill name required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		doc, err := s.app.GetSkill(r.Context(), name)
		if err != nil {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, SkillDetailResponse{Skill: doc})
	case http.MethodPut:
		var req SaveSkillRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
			return
		}
		meta, err := s.app.SaveSkill(r.Context(), skillInputFromRequest(req), name)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, meta)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: ErrMethodNotAllowed.Error()})
	}
}

func (s *Server) handleUploadSkill(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid multipart form"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "file is required"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	meta, err := s.app.ImportSkill(r.Context(), header.Filename, data)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, meta)
}

func skillInputFromRequest(req SaveSkillRequest) controlservice.SaveSkillInput {
	return controlservice.SaveSkillInput{
		Name:                 req.Name,
		Description:          req.Description,
		Version:              req.Version,
		FocusAreas:           req.FocusAreas,
		ComposedOf:           req.ComposedOf,
		CapabilityBoundaries: req.CapabilityBoundaries,
		SampleQuestions:      req.SampleQuestions,
		FollowUps:            req.FollowUps,
		Scenarios:            req.Scenarios,
		Adversarial:          req.Adversarial,
		Pressure:             req.Pressure,
		ScoringAnchors:       req.ScoringAnchors,
		InstallSource:        req.InstallSource,
		SourceURL:            req.SourceURL,
		Rating:               req.Rating,
		RatingCount:          req.RatingCount,
		Content:              req.Content,
	}
}
