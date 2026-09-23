// Package ai provides an isolated AI integration for business task drafts.
// It does not save, rate, confirm or publish tasks.
package ai

import (
	"context"
	"encoding/json"
)

type QuestionsRequest struct {
	Description string `json:"description"`
}

type Answer struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type CardRequest struct {
	Description string   `json:"description"`
	Answers     []Answer `json:"answers"`
}

type QuestionsResponse struct {
	Questions []string `json:"questions"`
}

// TaskCard has exactly the eleven named fields in the supplied contract.
// Empty strings mean unknown information; IDs and ratings belong to the backend.
type TaskCard struct {
	Title             string `json:"title"`
	Context           string `json:"context"`
	Need              string `json:"need"`
	Users             string `json:"users"`
	Data              string `json:"data"`
	Constraints       string `json:"constraints"`
	ExpectedResult    string `json:"expected_result"`
	SuccessCriteria   string `json:"success_criteria"`
	Contact           string `json:"contact"`
	InteractionFormat string `json:"interaction_format"`
	Topic             string `json:"topic"`
}

type Mode string

const (
	ModeLive     Mode = "live"
	ModeFallback Mode = "fallback"
)

type GenerationRequest struct {
	Name         string
	SystemPrompt string
	Input        json.RawMessage
	Schema       json.RawMessage
}

// Provider makes one attempt and must honor ctx cancellation and its deadline.
// Retries are owned exclusively by Service.
type Provider interface {
	Generate(context.Context, GenerationRequest) ([]byte, error)
}
