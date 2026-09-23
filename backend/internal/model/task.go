package model

import "time"

type Task struct {
	ID                 int64     `json:"id"`
	Title              string    `json:"title"`
	InitialDescription string    `json:"initial_description"`
	Context            string    `json:"context"`
	Need               string    `json:"need"`
	Users              string    `json:"users"`
	Data               string    `json:"data"`
	Constraints        string    `json:"constraints"`
	ExpectedResult     string    `json:"expected_result"`
	SuccessCriteria    string    `json:"success_criteria"`
	Contact            string    `json:"contact"`
	InteractionFormat  string    `json:"interaction_format"`
	Topic              string    `json:"topic"`
	Rating             int       `json:"rating"`
	ReadinessLevel     string    `json:"readiness_level"`
	Confirmed          bool      `json:"confirmed"`
	Published          bool      `json:"published"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
