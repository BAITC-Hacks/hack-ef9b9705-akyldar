package ai

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

const validQuestionsJSON = `{"questions":["Кто будет пользоваться?","Какие данные доступны?","Какой результат нужен?"]}`

func TestValidateQuestions(t *testing.T) {
	cases := []struct {
		name, raw string
		valid     bool
	}{
		{"valid", validQuestionsJSON, true},
		{"five", `{"questions":["One?","Two?","Three?","Four?","Five?"]}`, true},
		{"two", `{"questions":["One?","Two?"]}`, false},
		{"six", `{"questions":["One?","Two?","Three?","Four?","Five?","Six?"]}`, false},
		{"blank", `{"questions":["One?","   ","Three?"]}`, false},
		{"punctuation_only", `{"questions":["One?","???","Three?"]}`, false},
		{"duplicate", `{"questions":[" Кто  будет? ","кто будет!","Three?"]}`, false},
		{"hidden_duplicate", `{"questions":["Кто?","К\u200bто?","Three?"]}`, false},
		{"wrong_item", `{"questions":["One?",42,"Three?"]}`, false},
		{"null_item", `{"questions":["One?",null,"Three?"]}`, false},
		{"null_array", `{"questions":null}`, false},
		{"string_array", `{"questions":"one two three"}`, false},
		{"missing", `{}`, false},
		{"extra", `{"questions":["One?","Two?","Three?"],"score":100}`, false},
		{"duplicate_key", `{"questions":[],"questions":["One?","Two?","Three?"]}`, false},
		{"malformed", `{"questions":["One?","Two?","Three?"]`, false},
		{"fence_json", "```json\n" + validQuestionsJSON + "\n```", true},
		{"fence_plain", "```\r\n" + validQuestionsJSON + "\r\n```", true},
		{"nested_fence", "```json\n```json\n" + validQuestionsJSON + "\n```\n```", false},
		{"prose_before", "Result: " + validQuestionsJSON, false},
		{"prose_after", validQuestionsJSON + " thanks", false},
		{"two_objects", validQuestionsJSON + `{}`, false},
		{"array_root", `[]`, false},
		{"null_root", `null`, false},
		{"too_long", `{"questions":["` + strings.Repeat("қ", MaxQuestionRunes+1) + `","Two?","Three?"]}`, false},
		{"oversize", strings.Repeat(" ", MaxProviderResponseBytes) + validQuestionsJSON, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateQuestions([]byte(tc.raw))
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%t err=%v", tc.valid, err)
			}
		})
	}
}

func TestValidateCard(t *testing.T) {
	base, _ := json.Marshal(TaskCard{Title: "Қойма есебі", Context: "Қазір қолмен санаймыз."})
	var fields map[string]any
	_ = json.Unmarshal(base, &fields)
	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"missing", func(v map[string]any) { delete(v, "contact") }},
		{"extra", func(v map[string]any) { v["score"] = 100 }},
		{"null", func(v map[string]any) { v["data"] = nil }},
		{"number", func(v map[string]any) { v["users"] = 10 }},
		{"array", func(v map[string]any) { v["constraints"] = []string{} }},
		{"oversize_field", func(v map[string]any) { v["need"] = strings.Repeat("x", MaxCardFieldRunes+1) }},
	}
	if _, err := ValidateCard(base); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			copy := make(map[string]any)
			for k, v := range fields {
				copy[k] = v
			}
			tc.mutate(copy)
			raw, _ := json.Marshal(copy)
			if _, err := ValidateCard(raw); err == nil {
				t.Fatal("accepted invalid card")
			}
		})
	}
	for _, raw := range []string{string(base) + " extra", "before " + string(base), string(base) + string(base), "null", "[]", string(base[:len(base)-1]) + `,"title":"override"}`} {
		if _, err := ValidateCard([]byte(raw)); err == nil {
			t.Fatal("accepted invalid card JSON")
		}
	}
	if _, err := ValidateCard([]byte("```json\n" + string(base) + "\n```")); err != nil {
		t.Fatal(err)
	}
}

func TestParseInput(t *testing.T) {
	for _, raw := range []string{`{"description":"Задача"}`, `{"description":"  Қойма есебі  "}`} {
		got, err := ParseQuestionsRequest([]byte(raw))
		if err != nil || got.Description == "" {
			t.Fatalf("%v", err)
		}
	}
	badDescriptions := []string{
		`{}`, `null`, `[]`, `{"description":null}`, `{"description":42}`, `{"description":[]}`,
		`{"description":" \t\n "}`, `{"description":"a","extra":true}`,
		`{"description":"a","description":"b"}`, `{"description":"a","\u0064escription":"b"}`,
		`{"Description":"a"}`, `{"description":"a"} {}`, `{"description":"a"}suffix`,
		"```json\n{\"description\":\"a\"}\n```",
	}
	for _, raw := range badDescriptions {
		if _, err := ParseQuestionsRequest([]byte(raw)); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	if _, err := ParseQuestionsRequest([]byte{'{', '"', 'd', 'e', 's', 'c', 'r', 'i', 'p', 't', 'i', 'o', 'n', '"', ':', '"', 0xff, '"', '}'}); err == nil {
		t.Fatal("invalid UTF-8 accepted")
	}
	for _, raw := range []string{
		`{"description":"a","answers":[]}`,
		`{"description":"a","answers":[{"question":"Кто?","answer":""}]}`,
	} {
		if _, err := ParseCardRequest([]byte(raw)); err != nil {
			t.Fatal(err)
		}
	}
	for _, raw := range []string{
		`{"description":"a"}`, `{"description":"a","answers":null}`, `{"description":"a","answers":{}}`,
		`{"description":"a","answers":[null]}`, `{"description":"a","answers":[{"question":"Q?"}]}`,
		`{"description":"a","answers":[{"question":"Q?","answer":null}]}`,
		`{"description":"a","answers":[{"question":"Q?","answer":7}]}`,
		`{"description":"a","answers":[{"question":" ","answer":"A"}]}`,
		`{"description":"a","answers":[{"question":"Q?","answer":"A","score":1}]}`,
		`{"description":"a","answers":[{"question":"Q?","answer":"A","answer":"B"}]}`,
	} {
		if _, err := ParseCardRequest([]byte(raw)); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

func TestInputLimits(t *testing.T) {
	valid := CardRequest{Description: strings.Repeat("қ", MaxDescriptionRunes), Answers: []Answer{}}
	if err := ValidateCardRequest(valid); err != nil {
		t.Fatal("Unicode rune limit", err)
	}
	tooMany := make([]Answer, MaxAnswers+1)
	total := make([]Answer, 10)
	for i := range total {
		total[i] = Answer{Question: "Q?", Answer: strings.Repeat("x", MaxAnswerRunes)}
	}
	cases := []CardRequest{
		{Description: valid.Description + "a", Answers: []Answer{}},
		{Description: "a", Answers: tooMany},
		{Description: "a", Answers: []Answer{{Question: strings.Repeat("x", MaxQuestionRunes+1), Answer: ""}}},
		{Description: "a", Answers: []Answer{{Question: "Q?", Answer: strings.Repeat("x", MaxAnswerRunes+1)}}},
		{Description: "a", Answers: total},
	}
	for _, req := range cases {
		var target *InputError
		err := ValidateCardRequest(req)
		if !errors.As(err, &target) || !target.TooLarge {
			t.Fatalf("expected size limit, got %v", err)
		}
	}
	if _, err := ParseCardRequest([]byte(strings.Repeat(" ", MaxRequestBytes+1))); err == nil {
		t.Fatal("body limit not enforced")
	}
}

func TestGroundingAnchors(t *testing.T) {
	req := CardRequest{Description: "Нужен учёт", Answers: []Answer{{Question: "Бюджет 500000, Excel и contact@example.com?", Answer: "Не знаю"}}}
	for _, card := range []TaskCard{
		{Constraints: "Бюджет 500000"}, {Data: "Excel"}, {Contact: "contact@example.com"},
		{Contact: "https://example.com/team"}, {Contact: "@invented"}, {Users: "10 сотрудников"},
	} {
		if err := CheckGrounding(req, card); err == nil {
			t.Fatalf("unsupported anchor accepted: %+v", card)
		}
	}
	req.Answers = []Answer{{Question: "Данные и контакт?", Answer: "Excel, 10 сотрудников, contact@example.com, https://example.com/team, @teamdemo"}}
	card := TaskCard{Data: "Excel", Users: "10 сотрудников", Contact: "contact@example.com; https://example.com/team; @teamdemo"}
	if err := CheckGrounding(req, card); err != nil {
		t.Fatal(err)
	}
	if err := CheckGrounding(req, TaskCard{Need: "Упростить ведение учёта"}); err != nil {
		t.Fatal("paraphrase was blocked", err)
	}
	if err := CheckGrounding(CardRequest{Description: "Порог 2,5", Answers: []Answer{}}, TaskCard{Constraints: "Порог 2.5"}); err != nil {
		t.Fatal("decimal formatting", err)
	}
}
