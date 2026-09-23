package rating

import (
	"reflect"
	"testing"

	"backend/internal/model"
)

func TestCalculate(t *testing.T) {
	tests := []struct {
		name          string
		task          model.Task
		expectScore   int
		expectLevel   string
		expectMissing []string
	}{
		{name: "empty structured card", expectLevel: "draft", expectMissing: []string{"context", "need", "data", "expected_result", "success_criteria", "constraints", "users", "contact", "interaction_format"}},
		{name: "context", task: model.Task{Context: "Some context"}, expectScore: 10, expectLevel: "draft"},
		{name: "context and need", task: model.Task{Context: "Some context", Need: "Some need"}, expectScore: 20, expectLevel: "draft"},
		{name: "data", task: model.Task{Data: "Some data"}, expectScore: 20, expectLevel: "draft"},
		{name: "expected result", task: model.Task{ExpectedResult: "Some result"}, expectScore: 15, expectLevel: "draft"},
		{name: "success criteria", task: model.Task{SuccessCriteria: "Some criteria"}, expectScore: 15, expectLevel: "draft"},
		{name: "constraints", task: model.Task{Constraints: "Some constraints"}, expectScore: 10, expectLevel: "draft"},
		{name: "users", task: model.Task{Users: "Some users"}, expectScore: 10, expectLevel: "draft"},
		{name: "contact", task: model.Task{Contact: "contact@example.com"}, expectScore: 5, expectLevel: "draft"},
		{name: "business connection", task: model.Task{Contact: "contact@example.com", InteractionFormat: "Weekly"}, expectScore: 10, expectLevel: "draft"},
		{name: "whitespace is missing", task: model.Task{Context: "  \t", Need: "\n"}, expectScore: 0, expectLevel: "draft", expectMissing: []string{"context", "need", "data", "expected_result", "success_criteria", "constraints", "users", "contact", "interaction_format"}},
		{name: "fully filled", task: model.Task{Context: "Context", Need: "Need", Data: "Data", ExpectedResult: "Result", SuccessCriteria: "Criteria", Constraints: "Constraints", Users: "Users", Contact: "contact@example.com", InteractionFormat: "Weekly"}, expectScore: 100, expectLevel: "priority"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Calculate(test.task)
			if result.Score != test.expectScore {
				t.Fatalf("expected score %d, got %d", test.expectScore, result.Score)
			}
			if result.Level != test.expectLevel {
				t.Fatalf("expected level %q, got %q", test.expectLevel, result.Level)
			}
			if test.expectMissing != nil && !reflect.DeepEqual(result.Missing, test.expectMissing) {
				t.Fatalf("expected missing %v, got %v", test.expectMissing, result.Missing)
			}
		})
	}
}

func TestLevelForScore(t *testing.T) {
	tests := []struct {
		score int
		level string
	}{
		{score: 39, level: "draft"},
		{score: 40, level: "working"},
		{score: 69, level: "working"},
		{score: 70, level: "ready"},
		{score: 89, level: "ready"},
		{score: 90, level: "priority"},
		{score: 100, level: "priority"},
	}

	for _, test := range tests {
		if got := LevelForScore(test.score); got != test.level {
			t.Fatalf("score %d: expected level %q, got %q", test.score, test.level, got)
		}
	}
}
