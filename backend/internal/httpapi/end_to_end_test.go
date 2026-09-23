package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"backend/internal/model"
	"backend/internal/rating"
)

func TestEndToEndBackendFlow(t *testing.T) {
	router, teamRepository := newProposalTestRouter(t)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{"initial_description":"Improve warehouse handoffs"}`))
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected task creation status %d, got %d", http.StatusCreated, createRecorder.Code)
	}

	var task model.Task
	if err := json.NewDecoder(createRecorder.Body).Decode(&task); err != nil {
		t.Fatalf("decode created task: %v", err)
	}
	if task.Rating != 0 || task.ReadinessLevel != "draft" {
		t.Fatalf("unexpected initial task rating state: %+v", task)
	}

	taskPath := "/api/tasks/" + strconv.FormatInt(task.ID, 10)
	updateBody := `{"context":"Warehouse work is tracked manually.","need":"Reduce handoff delays.","data":"Existing inventory CSV files.","expected_result":"A shared handoff view."}`
	updateRequest := httptest.NewRequest(http.MethodPut, taskPath, bytes.NewBufferString(updateBody))
	updateRecorder := httptest.NewRecorder()
	router.ServeHTTP(updateRecorder, updateRequest)
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected task update status %d, got %d", http.StatusOK, updateRecorder.Code)
	}

	ratingRequest := httptest.NewRequest(http.MethodGet, taskPath+"/rating", nil)
	ratingRecorder := httptest.NewRecorder()
	router.ServeHTTP(ratingRecorder, ratingRequest)
	if ratingRecorder.Code != http.StatusOK {
		t.Fatalf("expected rating status %d, got %d", http.StatusOK, ratingRecorder.Code)
	}
	var ratingResult rating.Result
	if err := json.NewDecoder(ratingRecorder.Body).Decode(&ratingResult); err != nil {
		t.Fatalf("decode rating response: %v", err)
	}
	if ratingResult.Score != 55 || ratingResult.Level != "working" {
		t.Fatalf("expected rating 55/working, got %d/%q", ratingResult.Score, ratingResult.Level)
	}

	confirmRequest := httptest.NewRequest(http.MethodPost, taskPath+"/confirm", nil)
	confirmRecorder := httptest.NewRecorder()
	router.ServeHTTP(confirmRecorder, confirmRequest)
	if confirmRecorder.Code != http.StatusOK {
		t.Fatalf("expected confirmation status %d, got %d", http.StatusOK, confirmRecorder.Code)
	}
	var confirmedTask model.Task
	if err := json.NewDecoder(confirmRecorder.Body).Decode(&confirmedTask); err != nil {
		t.Fatalf("decode confirmed task: %v", err)
	}
	if !confirmedTask.Confirmed {
		t.Fatal("expected task to be confirmed")
	}

	publishRequest := httptest.NewRequest(http.MethodPost, taskPath+"/publish", nil)
	publishRecorder := httptest.NewRecorder()
	router.ServeHTTP(publishRecorder, publishRequest)
	if publishRecorder.Code != http.StatusOK {
		t.Fatalf("expected publication status %d, got %d", http.StatusOK, publishRecorder.Code)
	}
	var publishedTask model.Task
	if err := json.NewDecoder(publishRecorder.Body).Decode(&publishedTask); err != nil {
		t.Fatalf("decode published task: %v", err)
	}
	if !publishedTask.Published {
		t.Fatal("expected task to be published")
	}

	catalogRequest := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	catalogRecorder := httptest.NewRecorder()
	router.ServeHTTP(catalogRecorder, catalogRequest)
	if catalogRecorder.Code != http.StatusOK {
		t.Fatalf("expected catalog status %d, got %d", http.StatusOK, catalogRecorder.Code)
	}
	var catalog []model.Task
	if err := json.NewDecoder(catalogRecorder.Body).Decode(&catalog); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}
	if !containsTask(catalog, task.ID) {
		t.Fatalf("published task %d was not visible in catalog", task.ID)
	}

	team := &model.Team{Name: "End-to-end team"}
	if err := teamRepository.CreateTeam(t.Context(), team); err != nil {
		t.Fatalf("create test team: %v", err)
	}
	proposalBody := `{"team_id":` + strconv.FormatInt(team.ID, 10) + `,"idea":"Build a handoff dashboard","plan":"Prototype the shared view","deadline":"2026-12-01","prototype_url":"https://example.com/handoff"}`
	proposalRequest := httptest.NewRequest(http.MethodPost, taskPath+"/proposals", bytes.NewBufferString(proposalBody))
	proposalRecorder := httptest.NewRecorder()
	router.ServeHTTP(proposalRecorder, proposalRequest)
	if proposalRecorder.Code != http.StatusCreated {
		t.Fatalf("expected proposal creation status %d, got %d", http.StatusCreated, proposalRecorder.Code)
	}
	var proposal model.Proposal
	if err := json.NewDecoder(proposalRecorder.Body).Decode(&proposal); err != nil {
		t.Fatalf("decode created proposal: %v", err)
	}
	if proposal.Status != "pending" {
		t.Fatalf("expected new proposal to be pending, got %q", proposal.Status)
	}

	listRequest := httptest.NewRequest(http.MethodGet, taskPath+"/proposals", nil)
	listRecorder := httptest.NewRecorder()
	router.ServeHTTP(listRecorder, listRequest)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected proposal list status %d, got %d", http.StatusOK, listRecorder.Code)
	}
	var proposals []model.Proposal
	if err := json.NewDecoder(listRecorder.Body).Decode(&proposals); err != nil {
		t.Fatalf("decode proposal list: %v", err)
	}
	if len(proposals) != 1 || proposals[0].ID != proposal.ID {
		t.Fatalf("expected created proposal in list, got %+v", proposals)
	}

	statusRequest := httptest.NewRequest(http.MethodPatch, "/api/proposals/"+strconv.FormatInt(proposal.ID, 10)+"/status", bytes.NewBufferString(`{"status":"accepted"}`))
	statusRecorder := httptest.NewRecorder()
	router.ServeHTTP(statusRecorder, statusRequest)
	if statusRecorder.Code != http.StatusOK {
		t.Fatalf("expected proposal status update %d, got %d", http.StatusOK, statusRecorder.Code)
	}
	var acceptedProposal model.Proposal
	if err := json.NewDecoder(statusRecorder.Body).Decode(&acceptedProposal); err != nil {
		t.Fatalf("decode accepted proposal: %v", err)
	}
	if acceptedProposal.Status != "accepted" {
		t.Fatalf("expected accepted proposal, got %q", acceptedProposal.Status)
	}
}

func containsTask(tasks []model.Task, id int64) bool {
	for _, task := range tasks {
		if task.ID == id {
			return true
		}
	}
	return false
}
