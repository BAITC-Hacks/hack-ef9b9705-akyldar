package repository

import (
	"context"
	"testing"

	"backend/internal/model"
	"backend/internal/rating"
)

func publishCatalogTask(t *testing.T, repo *TaskRepository, task *model.Task) {
	t.Helper()

	if err := repo.Create(context.Background(), task); err != nil {
		t.Fatalf("create catalog task: %v", err)
	}
	calculation := rating.Calculate(*task)
	task.Rating = calculation.Score
	task.ReadinessLevel = calculation.Level
	if err := repo.Update(context.Background(), task); err != nil {
		t.Fatalf("update catalog task rating: %v", err)
	}
	if err := repo.Confirm(context.Background(), task); err != nil {
		t.Fatalf("confirm catalog task: %v", err)
	}
	if err := repo.Publish(context.Background(), task); err != nil {
		t.Fatalf("publish catalog task: %v", err)
	}
}

func TestTaskRepositoryListPublished(t *testing.T) {
	repo := newTestTaskRepository(t)

	unpublished := &model.Task{Title: "Unpublished", Topic: "logistics"}
	if err := repo.Create(context.Background(), unpublished); err != nil {
		t.Fatalf("create unpublished task: %v", err)
	}

	draft := &model.Task{Title: "Draft", Topic: "logistics", Context: "Context", Need: "Need"}
	working := &model.Task{Title: "Working", Topic: "logistics", Context: "Context", Need: "Need", Data: "Data", ExpectedResult: "Result"}
	ready := &model.Task{Title: "Ready", Topic: "operations", Context: "Context", Need: "Need", Data: "Data", ExpectedResult: "Result", Constraints: "Constraints", Users: "Users"}
	priority := &model.Task{Title: "Priority", Topic: "logistics", Context: "Context", Need: "Need", Data: "Data", ExpectedResult: "Result", SuccessCriteria: "Criteria", Constraints: "Constraints", Users: "Users", Contact: "contact@example.com"}

	for _, task := range []*model.Task{draft, working, ready, priority} {
		publishCatalogTask(t, repo, task)
	}

	all, err := repo.ListPublished(context.Background(), TaskListFilter{})
	if err != nil {
		t.Fatalf("list published tasks: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("expected 4 published tasks, got %d", len(all))
	}
	for _, task := range all {
		if !task.Published {
			t.Fatalf("unpublished task returned: %+v", task)
		}
		if task.ID == unpublished.ID {
			t.Fatalf("unpublished task returned: %d", task.ID)
		}
	}

	sorted, err := repo.ListPublished(context.Background(), TaskListFilter{Sort: "rating"})
	if err != nil {
		t.Fatalf("list sorted published tasks: %v", err)
	}
	expectedRatings := []int{95, 75, 55, 20}
	for index, expected := range expectedRatings {
		if sorted[index].Rating != expected {
			t.Fatalf("expected rating %d at index %d, got %d", expected, index, sorted[index].Rating)
		}
	}

	logistics, err := repo.ListPublished(context.Background(), TaskListFilter{Topic: "logistics"})
	if err != nil {
		t.Fatalf("list logistics tasks: %v", err)
	}
	if len(logistics) != 3 {
		t.Fatalf("expected 3 logistics tasks, got %d", len(logistics))
	}

	readyTasks, err := repo.ListPublished(context.Background(), TaskListFilter{Level: "ready"})
	if err != nil {
		t.Fatalf("list ready tasks: %v", err)
	}
	if len(readyTasks) != 1 || readyTasks[0].ReadinessLevel != "ready" {
		t.Fatalf("unexpected ready tasks: %+v", readyTasks)
	}

	combined, err := repo.ListPublished(context.Background(), TaskListFilter{Topic: "logistics", Level: "priority", Sort: "rating"})
	if err != nil {
		t.Fatalf("list combined tasks: %v", err)
	}
	if len(combined) != 1 || combined[0].Title != "Priority" {
		t.Fatalf("unexpected combined result: %+v", combined)
	}

	empty, err := repo.ListPublished(context.Background(), TaskListFilter{Topic: "missing"})
	if err != nil {
		t.Fatalf("list empty result: %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("expected non-nil empty result, got %+v", empty)
	}
}

func TestTaskRepositoryListPublishedRatingTieBreaksByID(t *testing.T) {
	repo := newTestTaskRepository(t)
	first := &model.Task{Title: "First", Context: "Context", Need: "Need"}
	second := &model.Task{Title: "Second", Context: "Context", Need: "Need"}

	publishCatalogTask(t, repo, first)
	publishCatalogTask(t, repo, second)

	tasks, err := repo.ListPublished(context.Background(), TaskListFilter{Sort: "rating"})
	if err != nil {
		t.Fatalf("list tied published tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tied tasks, got %d", len(tasks))
	}
	if tasks[0].Rating != tasks[1].Rating || tasks[0].ID <= tasks[1].ID {
		t.Fatalf("expected equal ratings ordered by descending ID, got %+v", tasks)
	}
}
