package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend/internal/model"
)

func TestProposalRepositoryCreateAndList(t *testing.T) {
	taskRepository, teamRepository, proposalRepository := newTestRepositories(t)
	task := &model.Task{InitialDescription: "Published task"}
	if err := taskRepository.Create(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	team := &model.Team{Name: "Proposal Team"}
	if err := teamRepository.CreateTeam(context.Background(), team); err != nil {
		t.Fatalf("create team: %v", err)
	}

	first := &model.Proposal{
		TaskID:       task.ID,
		TeamID:       team.ID,
		Idea:         "Build a dashboard",
		Plan:         "Analyze data, build API, create prototype",
		Deadline:     "2026-10-15",
		PrototypeURL: "https://example.com/one",
		Status:       "accepted",
	}
	if err := proposalRepository.Create(context.Background(), first); err != nil {
		t.Fatalf("create proposal: %v", err)
	}
	if first.ID <= 0 {
		t.Fatalf("expected positive proposal ID, got %d", first.ID)
	}
	if first.Status != "pending" {
		t.Fatalf("expected pending status, got %q", first.Status)
	}

	second := &model.Proposal{
		TaskID:       task.ID,
		TeamID:       team.ID,
		Idea:         "Build a report",
		Plan:         "Create a compact reporting prototype",
		Deadline:     "2026-10-20",
		PrototypeURL: "https://example.com/two",
	}
	if err := proposalRepository.Create(context.Background(), second); err != nil {
		t.Fatalf("create second proposal: %v", err)
	}

	proposals, err := proposalRepository.ListByTaskID(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("list proposals: %v", err)
	}
	if len(proposals) != 2 {
		t.Fatalf("expected 2 proposals, got %d", len(proposals))
	}
	if proposals[0].ID != second.ID || proposals[1].ID != first.ID {
		t.Fatalf("expected newest-first ordering, got IDs %d and %d", proposals[0].ID, proposals[1].ID)
	}
	if proposals[1].TaskID != task.ID || proposals[1].TeamID != team.ID || proposals[1].Idea != first.Idea || proposals[1].Plan != first.Plan || proposals[1].Deadline != first.Deadline || proposals[1].PrototypeURL != first.PrototypeURL || proposals[1].Status != "pending" {
		t.Fatalf("proposal data did not round-trip: %+v", proposals[1])
	}

	empty, err := proposalRepository.ListByTaskID(context.Background(), 999)
	if err != nil {
		t.Fatalf("list proposals for empty task: %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("expected non-nil empty proposal list, got %+v", empty)
	}
}

func TestProposalRepositoryUpdateStatusTransitions(t *testing.T) {
	taskRepository, teamRepository, proposalRepository := newTestRepositories(t)
	task := &model.Task{InitialDescription: "Task"}
	if err := taskRepository.Create(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	team := &model.Team{Name: "Team"}
	if err := teamRepository.CreateTeam(context.Background(), team); err != nil {
		t.Fatalf("create team: %v", err)
	}

	createProposal := func(idea string) *model.Proposal {
		t.Helper()
		proposal := &model.Proposal{
			TaskID:       task.ID,
			TeamID:       team.ID,
			Idea:         idea,
			Plan:         "Plan",
			Deadline:     "2026-10-15",
			PrototypeURL: "https://example.com",
		}
		if err := proposalRepository.Create(context.Background(), proposal); err != nil {
			t.Fatalf("create proposal: %v", err)
		}
		return proposal
	}

	accepted := createProposal("Accepted proposal")
	rejected := createProposal("Rejected proposal")
	originalCreatedAt := accepted.CreatedAt
	originalIdea := accepted.Idea
	originalPlan := accepted.Plan
	originalDeadline := accepted.Deadline
	originalURL := accepted.PrototypeURL
	time.Sleep(time.Millisecond)

	updated, err := proposalRepository.UpdateStatus(context.Background(), accepted.ID, "accepted")
	if err != nil {
		t.Fatalf("pending to accepted: %v", err)
	}
	if updated.Status != "accepted" {
		t.Fatalf("expected accepted status, got %q", updated.Status)
	}
	if !updated.CreatedAt.Equal(originalCreatedAt) || !updated.UpdatedAt.After(accepted.UpdatedAt) {
		t.Fatalf("unexpected timestamps after acceptance: %+v", updated)
	}
	if updated.Idea != originalIdea || updated.Plan != originalPlan || updated.Deadline != originalDeadline || updated.PrototypeURL != originalURL || updated.TaskID != task.ID || updated.TeamID != team.ID {
		t.Fatalf("non-status fields changed: %+v", updated)
	}

	repeatedAccepted, err := proposalRepository.UpdateStatus(context.Background(), accepted.ID, "accepted")
	if err != nil || repeatedAccepted.Status != "accepted" {
		t.Fatalf("repeated accepted update failed: proposal=%+v err=%v", repeatedAccepted, err)
	}

	updated, err = proposalRepository.UpdateStatus(context.Background(), accepted.ID, "rejected")
	if err != nil || updated.Status != "rejected" {
		t.Fatalf("accepted to rejected failed: proposal=%+v err=%v", updated, err)
	}
	updated, err = proposalRepository.UpdateStatus(context.Background(), accepted.ID, "accepted")
	if err != nil || updated.Status != "accepted" {
		t.Fatalf("rejected to accepted failed: proposal=%+v err=%v", updated, err)
	}

	updated, err = proposalRepository.UpdateStatus(context.Background(), rejected.ID, "rejected")
	if err != nil || updated.Status != "rejected" {
		t.Fatalf("pending to rejected failed: proposal=%+v err=%v", updated, err)
	}
	repeatedRejected, err := proposalRepository.UpdateStatus(context.Background(), rejected.ID, "rejected")
	if err != nil || repeatedRejected.Status != "rejected" {
		t.Fatalf("repeated rejected update failed: proposal=%+v err=%v", repeatedRejected, err)
	}
}

func TestProposalRepositoryUpdateStatusNotFound(t *testing.T) {
	_, _, proposalRepository := newTestRepositories(t)

	_, err := proposalRepository.UpdateStatus(context.Background(), 999, "accepted")
	if !errors.Is(err, ErrProposalNotFound) {
		t.Fatalf("expected ErrProposalNotFound, got %v", err)
	}
}
