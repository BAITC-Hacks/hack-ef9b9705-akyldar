package rating

import (
	"strings"

	"backend/internal/model"
)

type Result struct {
	Score     int            `json:"score"`
	Level     string         `json:"level"`
	Breakdown map[string]int `json:"breakdown"`
	Missing   []string       `json:"missing"`
}

func Calculate(task model.Task) Result {
	result := Result{
		Breakdown: map[string]int{
			"context_need":        0,
			"data":                0,
			"expected_result":     0,
			"success_criteria":    0,
			"constraints":         0,
			"users":               0,
			"business_connection": 0,
		},
	}

	if filled(task.Context) {
		result.Breakdown["context_need"] += 10
	} else {
		result.Missing = append(result.Missing, "context")
	}
	if filled(task.Need) {
		result.Breakdown["context_need"] += 10
	} else {
		result.Missing = append(result.Missing, "need")
	}
	if filled(task.Data) {
		result.Breakdown["data"] = 20
	} else {
		result.Missing = append(result.Missing, "data")
	}
	if filled(task.ExpectedResult) {
		result.Breakdown["expected_result"] = 15
	} else {
		result.Missing = append(result.Missing, "expected_result")
	}
	if filled(task.SuccessCriteria) {
		result.Breakdown["success_criteria"] = 15
	} else {
		result.Missing = append(result.Missing, "success_criteria")
	}
	if filled(task.Constraints) {
		result.Breakdown["constraints"] = 10
	} else {
		result.Missing = append(result.Missing, "constraints")
	}
	if filled(task.Users) {
		result.Breakdown["users"] = 10
	} else {
		result.Missing = append(result.Missing, "users")
	}
	if filled(task.Contact) {
		result.Breakdown["business_connection"] += 5
	} else {
		result.Missing = append(result.Missing, "contact")
	}
	if filled(task.InteractionFormat) {
		result.Breakdown["business_connection"] += 5
	} else {
		result.Missing = append(result.Missing, "interaction_format")
	}

	for _, points := range result.Breakdown {
		result.Score += points
	}
	if result.Score < 0 {
		result.Score = 0
	}
	if result.Score > 100 {
		result.Score = 100
	}
	result.Level = LevelForScore(result.Score)

	return result
}

func LevelForScore(score int) string {
	switch {
	case score < 40:
		return "draft"
	case score < 70:
		return "working"
	case score < 90:
		return "ready"
	default:
		return "priority"
	}
}

func filled(value string) bool {
	return strings.TrimSpace(value) != ""
}
