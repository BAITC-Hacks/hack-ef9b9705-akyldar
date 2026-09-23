package httpapi

import (
	"encoding/json"
	"net/http"

	"backend/internal/repository"
)

func NewRouter(taskRepository *repository.TaskRepository, teamRepository *repository.TeamRepository, proposalRepository *repository.ProposalRepository) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", healthHandler)

	tasks := newTaskHandler(taskRepository, teamRepository, proposalRepository)
	mux.HandleFunc("/api/tasks", tasks.handleCollection)
	mux.HandleFunc("/api/tasks/", tasks.handleByID)

	proposalStatus := newProposalStatusHandler(proposalRepository)
	mux.HandleFunc("/api/proposals/", proposalStatus.handle)

	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Status string `json:"status"`
	}{
		Status: "ok",
	})
}
