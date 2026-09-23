package seed

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"backend/internal/database"
	"backend/internal/rating"
	"backend/internal/repository"
)

func TestRunCreatesDeterministicDemoData(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "seed.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	if err := database.InitSchema(db); err != nil {
		t.Fatalf("initialize schema: %v", err)
	}

	ctx := context.Background()
	if err := Run(ctx, db); err != nil {
		t.Fatalf("run seed: %v", err)
	}

	assertRowCount(t, db, "tasks", 5)
	assertRowCount(t, db, "teams", 5)
	assertRowCount(t, db, "proposals", 5)

	taskRepository := repository.NewTaskRepository(db)
	tasks, err := taskRepository.ListPublished(ctx, repository.TaskListFilter{})
	if err != nil {
		t.Fatalf("list published demo tasks: %v", err)
	}
	if len(tasks) != 5 {
		t.Fatalf("expected 5 published demo tasks, got %d", len(tasks))
	}

	levels := make(map[string]bool)
	for _, task := range tasks {
		calculation := rating.Calculate(task)
		if task.Rating != calculation.Score || task.ReadinessLevel != calculation.Level {
			t.Fatalf("task %q has inconsistent rating: stored=%d/%q calculated=%d/%q", task.Title, task.Rating, task.ReadinessLevel, calculation.Score, calculation.Level)
		}
		levels[task.ReadinessLevel] = true
	}
	for _, level := range []string{"draft", "working", "ready", "priority"} {
		if !levels[level] {
			t.Fatalf("expected a published %s demo task, got levels %v", level, levels)
		}
	}

	statuses := make(map[string]bool)
	proposalRepository := repository.NewProposalRepository(db)
	for _, task := range tasks {
		proposals, err := proposalRepository.ListByTaskID(ctx, task.ID)
		if err != nil {
			t.Fatalf("list proposals for task %d: %v", task.ID, err)
		}
		for _, proposal := range proposals {
			statuses[proposal.Status] = true
		}
	}
	for _, status := range []string{"pending", "accepted", "rejected"} {
		if !statuses[status] {
			t.Fatalf("expected a %s demo proposal, got statuses %v", status, statuses)
		}
	}

	if err := Run(ctx, db); err != nil {
		t.Fatalf("run seed a second time: %v", err)
	}
	assertRowCount(t, db, "tasks", 5)
	assertRowCount(t, db, "teams", 5)
	assertRowCount(t, db, "proposals", 5)
}

func assertRowCount(t *testing.T, db *sql.DB, table string, expected int) {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != expected {
		t.Fatalf("expected %d %s, got %d", expected, table, count)
	}
}
