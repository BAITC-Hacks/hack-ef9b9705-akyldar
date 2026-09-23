package model

type Team struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	Interests    []string `json:"interests"`
	Skills       []string `json:"skills"`
	Technologies []string `json:"technologies"`
}
