package repository

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"backend/internal/database"
	"backend/internal/model"
)

func newTestRepositories(t *testing.T) (*TaskRepository, *TeamRepository, *ProposalRepository) {
	t.Helper()

	db, err := database.Open(filepath.Join(t.TempDir(), "proposals.db"))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.InitSchema(db); err != nil {
		t.Fatalf("initialize test schema: %v", err)
	}

	return NewTaskRepository(db), NewTeamRepository(db), NewProposalRepository(db)
}

func TestTeamRepositoryCreateAndGet(t *testing.T) {
	_, teamRepository, _ := newTestRepositories(t)
	want := &model.Team{
		Name:         "Inventory Team",
		Interests:    []string{"logistics", "automation"},
		Skills:       []string{"backend", "data analysis"},
		Technologies: []string{"Go", "SQLite"},
	}

	if err := teamRepository.CreateTeam(context.Background(), want); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if want.ID <= 0 {
		t.Fatalf("expected positive team ID, got %d", want.ID)
	}

	got, err := teamRepository.GetTeamByID(context.Background(), want.ID)
	if err != nil {
		t.Fatalf("get team: %v", err)
	}
	if got.Name != want.Name || !reflect.DeepEqual(got.Interests, want.Interests) || !reflect.DeepEqual(got.Skills, want.Skills) || !reflect.DeepEqual(got.Technologies, want.Technologies) {
		t.Fatalf("team data did not round-trip: want=%+v got=%+v", want, got)
	}
}

func TestTeamRepositoryGetNotFound(t *testing.T) {
	_, teamRepository, _ := newTestRepositories(t)

	_, err := teamRepository.GetTeamByID(context.Background(), 999)
	if !errors.Is(err, ErrTeamNotFound) {
		t.Fatalf("expected ErrTeamNotFound, got %v", err)
	}
}
