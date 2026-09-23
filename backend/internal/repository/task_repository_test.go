package repository

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

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

func TestTaskRepositoryUpdate(t *testing.T) {
	repo := newTestTaskRepository(t)
	task := &model.Task{
		InitialDescription: "Original draft description",
		Title:              "Original title",
		Topic:              "original-topic",
	}

	if err := repo.Create(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	originalCreatedAt := task.CreatedAt
	originalUpdatedAt := task.UpdatedAt
	time.Sleep(time.Millisecond)

	task.Title = "Updated title"
	task.Context = "Updated context"
	task.Need = "Updated need"
	task.Users = "Updated users"
	task.Data = "Updated data"
	task.Constraints = "Updated constraints"
	task.ExpectedResult = "Updated expected result"
	task.SuccessCriteria = "Updated success criteria"
	task.Contact = "updated@example.com"
	task.InteractionFormat = "Updated interaction"
	task.Topic = "updated-topic"
	task.Rating = 55
	task.ReadinessLevel = "working"
	if err := repo.Update(context.Background(), task); err != nil {
		t.Fatalf("update task: %v", err)
	}

	got, err := repo.GetByID(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("get updated task: %v", err)
	}
	if got.Title != "Updated title" || got.Context != "Updated context" || got.Need != "Updated need" || got.Users != "Updated users" || got.Data != "Updated data" || got.Constraints != "Updated constraints" || got.ExpectedResult != "Updated expected result" || got.SuccessCriteria != "Updated success criteria" || got.Contact != "updated@example.com" || got.InteractionFormat != "Updated interaction" || got.Topic != "updated-topic" {
		t.Fatalf("editable fields were not updated: %+v", got)
	}
	if got.InitialDescription != "Original draft description" {
		t.Fatalf("expected initial description to remain unchanged, got %q", got.InitialDescription)
	}
	if got.Rating != 55 {
		t.Fatalf("expected rating 55, got %d", got.Rating)
	}
	if got.ReadinessLevel != "working" {
		t.Fatalf("expected readiness level working, got %q", got.ReadinessLevel)
	}
	if got.Confirmed || got.Published {
		t.Fatalf("expected confirmed and published to remain false: %+v", got)
	}
	if !got.CreatedAt.Equal(originalCreatedAt) {
		t.Fatalf("expected created_at to remain unchanged: before %s, after %s", originalCreatedAt, got.CreatedAt)
	}
	if !got.UpdatedAt.After(originalUpdatedAt) {
		t.Fatalf("expected updated_at to change: before %s, after %s", originalUpdatedAt, got.UpdatedAt)
	}
}

func TestTaskRepositoryUpdateNotFound(t *testing.T) {
	repo := newTestTaskRepository(t)

	err := repo.Update(context.Background(), &model.Task{ID: 999})
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}
