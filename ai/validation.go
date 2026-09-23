package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	MaxRequestBytes     = 64 << 10
	MaxDescriptionRunes = 8000
	MaxAnswers          = 20
	MaxQuestionRunes    = 500
	MaxAnswerRunes      = 2000
	MaxAnswersRunes     = 16000
	MaxCardFieldRunes   = 8000
)

// InputError carries only fixed, safe messages, never request contents.
type InputError struct {
	Code     string
	Message  string
	TooLarge bool
}

func (e *InputError) Error() string { return e.Message }

func invalidInput() error {
	return &InputError{Code: "invalid_request", Message: "Expected one JSON object with the documented fields and types."}
}

func inputLimit() error {
	return &InputError{Code: "input_too_large", Message: "The request exceeds a documented size limit.", TooLarge: true}
}

// strictObject rejects duplicate keys (including escaped equivalents), null,
// arrays, trailing data and invalid UTF-8. A Go struct decode alone cannot do this.
func strictObject(raw []byte) (map[string]json.RawMessage, error) {
	if !utf8.Valid(raw) {
		return nil, errors.New("invalid UTF-8")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	start, err := d.Token()
	if err != nil || start != json.Delim('{') {
		return nil, errors.New("expected object")
	}
	fields := make(map[string]json.RawMessage)
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return nil, errors.New("invalid object key")
		}
		key, ok := token.(string)
		if !ok {
			return nil, errors.New("invalid object key")
		}
		if _, exists := fields[key]; exists {
			return nil, errors.New("duplicate object key")
		}
		var value json.RawMessage
		if err := d.Decode(&value); err != nil {
			return nil, errors.New("invalid object value")
		}
		fields[key] = value
	}
	if end, err := d.Token(); err != nil || end != json.Delim('}') {
		return nil, errors.New("invalid object end")
	}
	var trailing json.RawMessage
	if err := d.Decode(&trailing); err != io.EOF {
		return nil, errors.New("unexpected trailing data")
	}
	return fields, nil
}

func exactKeys(fields map[string]json.RawMessage, keys ...string) bool {
	if len(fields) != len(keys) {
		return false
	}
	for _, key := range keys {
		if _, ok := fields[key]; !ok {
			return false
		}
	}
	return true
}

func jsonString(raw json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '"' {
		return "", errors.New("expected string")
	}
	var value string
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return "", errors.New("expected string")
	}
	return value, nil
}

func ParseQuestionsRequest(raw []byte) (QuestionsRequest, error) {
	if len(raw) > MaxRequestBytes {
		return QuestionsRequest{}, inputLimit()
	}
	fields, err := strictObject(raw)
	if err != nil || !exactKeys(fields, "description") {
		return QuestionsRequest{}, invalidInput()
	}
	description, err := jsonString(fields["description"])
	if err != nil {
		return QuestionsRequest{}, invalidInput()
	}
	req := QuestionsRequest{Description: description}
	return req, ValidateQuestionsRequest(req)
}

func ParseCardRequest(raw []byte) (CardRequest, error) {
	if len(raw) > MaxRequestBytes {
		return CardRequest{}, inputLimit()
	}
	fields, err := strictObject(raw)
	if err != nil || !exactKeys(fields, "description", "answers") {
		return CardRequest{}, invalidInput()
	}
	description, err := jsonString(fields["description"])
	if err != nil {
		return CardRequest{}, invalidInput()
	}
	array := bytes.TrimSpace(fields["answers"])
	if len(array) == 0 || array[0] != '[' {
		return CardRequest{}, invalidInput()
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(array, &entries); err != nil {
		return CardRequest{}, invalidInput()
	}
	if len(entries) > MaxAnswers {
		return CardRequest{}, inputLimit()
	}
	req := CardRequest{Description: description, Answers: make([]Answer, 0, len(entries))}
	for _, entry := range entries {
		pair, err := strictObject(entry)
		if err != nil || !exactKeys(pair, "question", "answer") {
			return CardRequest{}, invalidInput()
		}
		question, qerr := jsonString(pair["question"])
		answer, aerr := jsonString(pair["answer"])
		if qerr != nil || aerr != nil {
			return CardRequest{}, invalidInput()
		}
		req.Answers = append(req.Answers, Answer{Question: question, Answer: answer})
	}
	return req, ValidateCardRequest(req)
}

func ValidateQuestionsRequest(req QuestionsRequest) error {
	if !utf8.ValidString(req.Description) || strings.TrimSpace(req.Description) == "" {
		return &InputError{Code: "invalid_description", Message: "description must be a nonempty string."}
	}
	if utf8.RuneCountInString(req.Description) > MaxDescriptionRunes {
		return inputLimit()
	}
	return nil
}

func ValidateCardRequest(req CardRequest) error {
	if err := ValidateQuestionsRequest(QuestionsRequest{Description: req.Description}); err != nil {
		return err
	}
	if req.Answers == nil {
		return invalidInput()
	}
	if len(req.Answers) > MaxAnswers {
		return inputLimit()
	}
	total := 0
	for _, pair := range req.Answers {
		if !utf8.ValidString(pair.Question) || !utf8.ValidString(pair.Answer) || strings.TrimSpace(pair.Question) == "" {
			return invalidInput()
		}
		q, a := utf8.RuneCountInString(pair.Question), utf8.RuneCountInString(pair.Answer)
		if q > MaxQuestionRunes || a > MaxAnswerRunes {
			return inputLimit()
		}
		total += q + a
	}
	if total > MaxAnswersRunes {
		return inputLimit()
	}
	return nil
}

func unwrapFence(raw []byte) []byte {
	s := strings.TrimSpace(string(raw))
	if strings.HasPrefix(s, "```") {
		lineEnd := strings.IndexByte(s, '\n')
		if lineEnd < 0 {
			return []byte(s)
		}
		first := strings.TrimSuffix(s[:lineEnd], "\r")
		if (first == "```" || first == "```json") && strings.HasSuffix(s, "\n```") {
			s = strings.TrimSpace(s[lineEnd+1 : len(s)-4])
		}
	}
	return []byte(s)
}

func normalizeQuestion(q string) string {
	q = strings.ToLower(q)
	q = strings.ReplaceAll(q, "ё", "е")
	// Punctuation, whitespace and invisible formatting cannot hide duplicates.
	q = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return -1
	}, q)
	return q
}

func ValidateQuestions(raw []byte) (QuestionsResponse, error) {
	if len(raw) > MaxProviderResponseBytes || !utf8.Valid(raw) {
		return QuestionsResponse{}, errors.New("invalid questions size or encoding")
	}
	fields, err := strictObject(unwrapFence(raw))
	if err != nil || !exactKeys(fields, "questions") {
		return QuestionsResponse{}, errors.New("invalid questions object")
	}
	var values []json.RawMessage
	if err := json.Unmarshal(fields["questions"], &values); err != nil || len(values) < 3 || len(values) > 5 {
		return QuestionsResponse{}, errors.New("expected 3 to 5 questions")
	}
	result := QuestionsResponse{Questions: make([]string, 0, len(values))}
	seen := map[string]bool{}
	for _, value := range values {
		q, err := jsonString(value)
		if err != nil || utf8.RuneCountInString(q) > MaxQuestionRunes {
			return QuestionsResponse{}, errors.New("invalid question")
		}
		q = strings.TrimSpace(q)
		normalized := normalizeQuestion(q)
		if normalized == "" || seen[normalized] {
			return QuestionsResponse{}, errors.New("empty or duplicate question")
		}
		seen[normalized] = true
		result.Questions = append(result.Questions, q)
	}
	return result, nil
}

func ValidateCard(raw []byte) (TaskCard, error) {
	if len(raw) > MaxProviderResponseBytes || !utf8.Valid(raw) {
		return TaskCard{}, errors.New("invalid card size or encoding")
	}
	clean := unwrapFence(raw)
	fields, err := strictObject(clean)
	if err != nil {
		return TaskCard{}, errors.New("invalid card object")
	}
	var schema struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(cardSchema, &schema); err != nil || len(schema.Required) == 0 {
		return TaskCard{}, errors.New("invalid embedded card schema")
	}
	if !exactKeys(fields, schema.Required...) {
		return TaskCard{}, errors.New("card fields differ from schema")
	}
	for _, key := range schema.Required {
		value, err := jsonString(fields[key])
		if err != nil || utf8.RuneCountInString(value) > MaxCardFieldRunes {
			return TaskCard{}, fmt.Errorf("invalid card field type or size")
		}
	}
	var card TaskCard
	if err := json.Unmarshal(clean, &card); err != nil {
		return TaskCard{}, errors.New("invalid card")
	}
	return card, nil
}
