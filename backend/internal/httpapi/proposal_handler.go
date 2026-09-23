package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"backend/internal/model"
	"backend/internal/repository"
)

type createProposalRequest struct {
	TeamID       int64  `json:"team_id"`
	Idea         string `json:"idea"`
	Plan         string `json:"plan"`
	Deadline     string `json:"deadline"`
	PrototypeURL string `json:"prototype_url"`
}

func (h *taskHandler) createProposalByTaskID(w http.ResponseWriter, r *http.Request, taskID int64) {
	var request createProposalRequest
	if err := decodeJSONBody(r.Body, &request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if request.TeamID == 0 {
		writeJSONError(w, http.StatusBadRequest, "team_id is required")
		return
	}
	if request.TeamID < 0 {
		writeJSONError(w, http.StatusBadRequest, "team_id must be positive")
		return
	}

	request.Idea = strings.TrimSpace(request.Idea)
	if request.Idea == "" {
		writeJSONError(w, http.StatusBadRequest, "idea is required")
		return
	}
	request.Plan = strings.TrimSpace(request.Plan)
	if request.Plan == "" {
		writeJSONError(w, http.StatusBadRequest, "plan is required")
		return
	}
	request.Deadline = strings.TrimSpace(request.Deadline)
	if request.Deadline == "" {
		writeJSONError(w, http.StatusBadRequest, "deadline is required")
		return
	}
	request.PrototypeURL = strings.TrimSpace(request.PrototypeURL)
	if request.PrototypeURL == "" {
		writeJSONError(w, http.StatusBadRequest, "prototype_url is required")
		return
	}

	task, err := h.repository.GetByID(r.Context(), taskID)
	if errors.Is(err, repository.ErrTaskNotFound) {
		writeJSONError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if !task.Published {
		writeJSONError(w, http.StatusConflict, "task must be published before accepting proposals")
		return
	}

	if _, err := h.teamRepository.GetTeamByID(r.Context(), request.TeamID); errors.Is(err, repository.ErrTeamNotFound) {
		writeJSONError(w, http.StatusNotFound, "team not found")
		return
	} else if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	proposal := &model.Proposal{
		TaskID:       taskID,
		TeamID:       request.TeamID,
		Idea:         request.Idea,
		Plan:         request.Plan,
		Deadline:     request.Deadline,
		PrototypeURL: request.PrototypeURL,
	}
	if err := h.proposalRepository.Create(r.Context(), proposal); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, proposal)
}

func (h *taskHandler) listProposalsByTaskID(w http.ResponseWriter, r *http.Request, taskID int64) {
	if _, err := h.repository.GetByID(r.Context(), taskID); errors.Is(err, repository.ErrTaskNotFound) {
		writeJSONError(w, http.StatusNotFound, "task not found")
		return
	} else if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	proposals, err := h.proposalRepository.ListByTaskID(r.Context(), taskID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, proposals)
}
