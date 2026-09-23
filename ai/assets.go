package ai

import _ "embed"

//go:embed prompts/questions.txt
var questionsPrompt string

//go:embed prompts/card.txt
var cardPrompt string

//go:embed schemas/questions.schema.json
var questionsSchema []byte

//go:embed schemas/card.schema.json
var cardSchema []byte
