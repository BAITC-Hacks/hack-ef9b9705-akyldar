package repository

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"backend/internal/database"
	"backend/internal/model"
)

func newTestTaskRepository(t *testing.T) *TaskRepository {
	t.Helper()

	db, err := database.Open(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.InitSchema(db); err != nil {
		t.Fatalf("initialize test schema: %v", err)
	}

	return NewTaskRepository(db)
}

func TestTaskRepositoryCreateAndGetByID(t *testing.T) {
	repo := newTestTaskRepository(t)
	task := &model.Task{
		InitialDescription: "Need a warehouse automation system",
		Topic:              "logistics",
	}

	if err := repo.Create(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if task.ID <= 0 {
		t.Fatalf("expected a positive task ID, got %d", task.ID)
	}
	if task.Rating != 0 {
		t.Fatalf("expected rating 0, got %d", task.Rating)
	}
	if task.ReadinessLevel != "draft" {
		t.Fatalf("expected readiness level draft, got %q", task.ReadinessLevel)
	}
	if task.Confirmed {
		t.Fatal("expected confirmed to be false")
	}
	if task.Published {
		t.Fatal("expected published to be false")
	}

	got, err := repo.GetByID(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("get task by ID: %v", err)
	}
	if got.ID != task.ID {
		t.Fatalf("expected ID %d, got %d", task.ID, got.ID)
	}
	if got.InitialDescription != task.InitialDescription {
		t.Fatalf("expected description %q, got %q", task.InitialDescription, got.InitialDescription)
	}
	if got.Topic != task.Topic {
		t.Fatalf("expected topic %q, got %q", task.Topic, got.Topic)
	}
}

func TestTaskRepositoryGetByIDNotFound(t *testing.T) {
	repo := newTestTaskRepository(t)

	_, err := repo.GetByID(context.Background(), 999)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}
