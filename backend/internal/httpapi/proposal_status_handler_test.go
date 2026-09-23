package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"backend/internal/model"
)

func createProposalForStatusTest(t *testing.T, router http.Handler, taskID, teamID int64, idea string) model.Proposal {
	t.Helper()

	body := `{"team_id":` + strconv.FormatInt(teamID, 10) + `,"idea":"` + idea + `","plan":"Plan","deadline":"2026-10-15","prototype_url":"https://example.com"}`
	request := httptest.NewRequest(http.MethodPost, "/api/tasks/"+strconv.FormatInt(taskID, 10)+"/proposals", bytes.NewBufferString(body))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected proposal creation status %d, got %d", http.StatusCreated, recorder.Code)
	}

	var proposal model.Proposal
	if err := json.NewDecoder(recorder.Body).Decode(&proposal); err != nil {
		t.Fatalf("decode proposal: %v", err)
	}
	return proposal
}

func patchProposalStatus(t *testing.T, router http.Handler, proposalID int64, body string) (int, model.Proposal) {
	t.Helper()

	request := httptest.NewRequest(http.MethodPatch, "/api/proposals/"+strconv.FormatInt(proposalID, 10)+"/status", bytes.NewBufferString(body))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	var proposal model.Proposal
	if recorder.Code == http.StatusOK {
		if err := json.NewDecoder(recorder.Body).Decode(&proposal); err != nil {
			t.Fatalf("decode updated proposal: %v", err)
		}
	}
	return recorder.Code, proposal
}

func TestProposalStatusTransitionsHTTP(t *testing.T) {
	router, teamRepository := newProposalTestRouter(t)
	team := &model.Team{Name: "Team"}
	if err := teamRepository.CreateTeam(t.Context(), team); err != nil {
		t.Fatalf("create team: %v", err)
	}
	task := createHTTPTask(t, router, "Published task")
	publishHTTPTask(t, router, task.ID)
	proposal := createProposalForStatusTest(t, router, task.ID, team.ID, "Proposal")

	for _, test := range []struct {
		name   string
		status string
	}{
		{name: "pending to accepted", status: "accepted"},
		{name: "repeated accepted", status: "accepted"},
		{name: "accepted to rejected", status: "rejected"},
		{name: "rejected to accepted", status: "accepted"},
	} {
		t.Run(test.name, func(t *testing.T) {
			code, updated := patchProposalStatus(t, router, proposal.ID, `{"status":"`+test.status+`"}`)
			if code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, code)
			}
			if updated.Status != test.status {
				t.Fatalf("expected proposal status %q, got %q", test.status, updated.Status)
			}
		})
	}

	rejectedProposal := createProposalForStatusTest(t, router, task.ID, team.ID, "Rejected proposal")
	code, updated := patchProposalStatus(t, router, rejectedProposal.ID, `{"status":"rejected"}`)
	if code != http.StatusOK || updated.Status != "rejected" {
		t.Fatalf("pending to rejected failed: code=%d proposal=%+v", code, updated)
	}
	code, updated = patchProposalStatus(t, router, rejectedProposal.ID, `{"status":"rejected"}`)
	if code != http.StatusOK || updated.Status != "rejected" {
		t.Fatalf("repeated rejected failed: code=%d proposal=%+v", code, updated)
	}
}

func TestProposalStatusValidationHTTP(t *testing.T) {
	router, teamRepository := newProposalTestRouter(t)
	team := &model.Team{Name: "Team"}
	if err := teamRepository.CreateTeam(t.Context(), team); err != nil {
		t.Fatalf("create team: %v", err)
	}
	task := createHTTPTask(t, router, "Published task")
	publishHTTPTask(t, router, task.ID)
	proposal := createProposalForStatusTest(t, router, task.ID, team.ID, "Proposal")

	tests := []struct {
		name string
		path string
		body string
	}{
		{name: "invalid pending", path: "/api/proposals/" + strconv.FormatInt(proposal.ID, 10) + "/status", body: `{"status":"pending"}`},
		{name: "invalid approved", path: "/api/proposals/" + strconv.FormatInt(proposal.ID, 10) + "/status", body: `{"status":"approved"}`},
		{name: "missing status", path: "/api/proposals/" + strconv.FormatInt(proposal.ID, 10) + "/status", body: `{}`},
		{name: "malformed JSON", path: "/api/proposals/" + strconv.FormatInt(proposal.ID, 10) + "/status", body: `{"status":`},
		{name: "unknown field", path: "/api/proposals/" + strconv.FormatInt(proposal.ID, 10) + "/status", body: `{"status":"accepted","idea":"changed"}`},
		{name: "invalid proposal ID", path: "/api/proposals/invalid/status", body: `{"status":"accepted"}`},
		{name: "non-positive proposal ID", path: "/api/proposals/0/status", body: `{"status":"accepted"}`},
		{name: "nonexistent proposal", path: "/api/proposals/999/status", body: `{"status":"accepted"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPatch, test.path, bytes.NewBufferString(test.body))
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			expectedStatus := http.StatusBadRequest
			if test.name == "nonexistent proposal" {
				expectedStatus = http.StatusNotFound
			}
			if recorder.Code != expectedStatus {
				t.Fatalf("expected status %d, got %d", expectedStatus, recorder.Code)
			}
			if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
				t.Fatalf("expected JSON content type, got %q", contentType)
			}
		})
	}
}

func TestMultipleProposalStatusesRemainIndependent(t *testing.T) {
	router, teamRepository := newProposalTestRouter(t)
	teamOne := &model.Team{Name: "Team One"}
	teamTwo := &model.Team{Name: "Team Two"}
	if err := teamRepository.CreateTeam(t.Context(), teamOne); err != nil {
		t.Fatalf("create first team: %v", err)
	}
	if err := teamRepository.CreateTeam(t.Context(), teamTwo); err != nil {
		t.Fatalf("create second team: %v", err)
	}
	task := createHTTPTask(t, router, "Published task")
	publishHTTPTask(t, router, task.ID)
	first := createProposalForStatusTest(t, router, task.ID, teamOne.ID, "First proposal")
	second := createProposalForStatusTest(t, router, task.ID, teamTwo.ID, "Second proposal")

	if code, _ := patchProposalStatus(t, router, first.ID, `{"status":"accepted"}`); code != http.StatusOK {
		t.Fatalf("accept first proposal returned %d", code)
	}
	if code, _ := patchProposalStatus(t, router, second.ID, `{"status":"accepted"}`); code != http.StatusOK {
		t.Fatalf("accept second proposal returned %d", code)
	}
	if code, updated := patchProposalStatus(t, router, second.ID, `{"status":"rejected"}`); code != http.StatusOK || updated.Status != "rejected" {
		t.Fatalf("reject second proposal failed: code=%d proposal=%+v", code, updated)
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/tasks/"+strconv.FormatInt(task.ID, 10)+"/proposals", nil)
	listRecorder := httptest.NewRecorder()
	router.ServeHTTP(listRecorder, listRequest)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected proposal list status %d, got %d", http.StatusOK, listRecorder.Code)
	}

	var proposals []model.Proposal
	if err := json.NewDecoder(listRecorder.Body).Decode(&proposals); err != nil {
		t.Fatalf("decode proposal list: %v", err)
	}
	statuses := make(map[int64]string, len(proposals))
	for _, proposal := range proposals {
		statuses[proposal.ID] = proposal.Status
	}
	if statuses[first.ID] != "accepted" {
		t.Fatalf("accepting/rejecting another proposal changed first status: %v", statuses)
	}
	if statuses[second.ID] != "rejected" {
		t.Fatalf("expected second proposal rejected: %v", statuses)
	}
}
