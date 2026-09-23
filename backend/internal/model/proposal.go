package model

import "time"

type Proposal struct {
	ID           int64     `json:"id"`
	TaskID       int64     `json:"task_id"`
	TeamID       int64     `json:"team_id"`
	Idea         string    `json:"idea"`
	Plan         string    `json:"plan"`
	Deadline     string    `json:"deadline"`
	PrototypeURL string    `json:"prototype_url"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
