package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/repository"
)

type updateProposalStatusRequest struct {
	Status string `json:"status"`
}

type proposalStatusHandler struct {
	repository *repository.ProposalRepository
}

func newProposalStatusHandler(proposalRepository *repository.ProposalRepository) *proposalStatusHandler {
	return &proposalStatusHandler{repository: proposalRepository}
}

func (h *proposalStatusHandler) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		w.Header().Set("Allow", http.MethodPatch)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, err := parseProposalStatusID(r.URL.Path)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid proposal id")
		return
	}

	var request updateProposalStatusRequest
	if err := decodeJSONBody(r.Body, &request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if request.Status != "accepted" && request.Status != "rejected" {
		writeJSONError(w, http.StatusBadRequest, "invalid proposal status")
		return
	}

	proposal, err := h.repository.UpdateStatus(r.Context(), id, request.Status)
	if errors.Is(err, repository.ErrProposalNotFound) {
		writeJSONError(w, http.StatusNotFound, "proposal not found")
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, proposal)
}

func parseProposalStatusID(path string) (int64, error) {
	const prefix = "/api/proposals/"
	const suffix = "/status"

	rawID := strings.TrimPrefix(path, prefix)
	rawID = strings.TrimSuffix(rawID, suffix)
	if rawID == "" || strings.Contains(rawID, "/") {
		return 0, errors.New("invalid proposal id")
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid proposal id")
	}

	return id, nil
}
