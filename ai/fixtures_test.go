package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fixtureEnvelope[T any] struct {
	Synthetic bool `json:"synthetic"`
	Items     []T  `json:"items"`
}

type draftFixture struct {
	ID          int    `json:"id"`
	Topic       string `json:"topic"`
	Description string `json:"description"`
}

type cardFixture struct {
	ID     int             `json:"id"`
	Source json.RawMessage `json:"source"`
	Card   json.RawMessage `json:"card"`
}

type teamFixture struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Interests    []string `json:"interests"`
	Skills       []string `json:"skills"`
	Technologies []string `json:"technologies"`
}

type proposalFixture struct {
	TaskID       int    `json:"task_id"`
	TeamID       int    `json:"team_id"`
	Idea         string `json:"idea"`
	Plan         string `json:"plan"`
	Deadline     string `json:"deadline"`
	PrototypeURL string `json:"prototype_url"`
}

type evaluationFixture struct {
	ID string `json:"id"`
	CardRequest
	Expectations       []string `json:"expectations"`
	ForbiddenAdditions []string `json:"forbidden_additions"`
}

func readFixtures[T any](t *testing.T, name string) []T {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	fields, err := strictObject(raw)
	if err != nil || !exactKeys(fields, "synthetic", "items") {
		t.Fatalf("%s: expected exactly one synthetic/items envelope: %v", name, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var envelope fixtureEnvelope[T]
	if err := decoder.Decode(&envelope); err != nil {
		t.Fatalf("%s: invalid fixture: %v", name, err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		t.Fatalf("%s: unexpected trailing content", name)
	}
	if !envelope.Synthetic || len(envelope.Items) == 0 {
		t.Fatalf("%s: must be marked synthetic with nonempty items", name)
	}
	return envelope.Items
}

func requireFixtureStrings(t *testing.T, label string, values ...string) {
	t.Helper()
	if len(values) == 0 {
		t.Fatalf("%s: expected nonempty values", label)
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			t.Errorf("%s: empty value", label)
		}
	}
}

func TestSyntheticFixtureCollectionsAndReferences(t *testing.T) {
	drafts := readFixtures[draftFixture](t, "drafts")
	cards := readFixtures[cardFixture](t, "cards")
	teams := readFixtures[teamFixture](t, "teams")
	proposals := readFixtures[proposalFixture](t, "proposals")
	for name, count := range map[string]int{
		"drafts": len(drafts), "cards": len(cards),
		"teams": len(teams), "proposals": len(proposals),
	} {
		if count != 5 {
			t.Errorf("%s: got %d items, want 5", name, count)
		}
	}

	draftIDs := make(map[int]bool)
	topics := make(map[string]bool)
	for _, draft := range drafts {
		if draft.ID <= 0 || draftIDs[draft.ID] {
			t.Errorf("draft ID must be positive and unique: %d", draft.ID)
		}
		draftIDs[draft.ID] = true
		topics[draft.Topic] = true
		if err := ValidateQuestionsRequest(QuestionsRequest{Description: draft.Description}); err != nil {
			t.Errorf("draft %d: invalid description: %v", draft.ID, err)
		}
	}
	for _, topic := range []string{"logistics", "education", "retail", "healthcare administration", "analytics"} {
		if !topics[topic] {
			t.Errorf("drafts lack required topic %q", topic)
		}
	}

	cardIDs := make(map[int]bool)
	for _, item := range cards {
		if item.ID <= 0 || cardIDs[item.ID] {
			t.Errorf("card ID must be positive and unique: %d", item.ID)
		}
		cardIDs[item.ID] = true
	}
	teamIDs := make(map[int]bool)
	for _, team := range teams {
		if team.ID <= 0 || teamIDs[team.ID] {
			t.Errorf("team ID must be positive and unique: %d", team.ID)
		}
		teamIDs[team.ID] = true
		requireFixtureStrings(t, "team name", team.Name)
		requireFixtureStrings(t, "team interests", team.Interests...)
		requireFixtureStrings(t, "team skills", team.Skills...)
		requireFixtureStrings(t, "team technologies", team.Technologies...)
	}
	pairs := make(map[[2]int]bool)
	for _, proposal := range proposals {
		if !cardIDs[proposal.TaskID] || !teamIDs[proposal.TeamID] {
			t.Errorf("proposal has dangling task/team reference: %d/%d", proposal.TaskID, proposal.TeamID)
		}
		pair := [2]int{proposal.TaskID, proposal.TeamID}
		if pairs[pair] {
			t.Errorf("duplicate synthetic proposal for task/team %v", pair)
		}
		pairs[pair] = true
		requireFixtureStrings(t, "proposal", proposal.Idea, proposal.Plan, proposal.Deadline, proposal.PrototypeURL)
		parsed, err := url.Parse(proposal.PrototypeURL)
		if err != nil || parsed.Scheme != "https" || parsed.Host != "example.com" || parsed.User != nil || parsed.Path == "" {
			t.Errorf("proposal prototype must be a synthetic HTTPS example.com path: %q", proposal.PrototypeURL)
		}
	}
}

func TestCompleteCardFixturesHaveEvidence(t *testing.T) {
	for _, item := range readFixtures[cardFixture](t, "cards") {
		t.Run(fmt.Sprintf("card_%d", item.ID), func(t *testing.T) {
			source, err := ParseCardRequest(item.Source)
			if err != nil {
				t.Fatalf("invalid source request: %v", err)
			}
			if err := ValidateCardRequest(source); err != nil {
				t.Fatalf("source violates request limits: %v", err)
			}
			card, err := ValidateCard(item.Card)
			if err != nil {
				t.Fatalf("card violates response contract: %v", err)
			}
			if err := CheckGrounding(source, card); err != nil {
				t.Errorf("important factual anchors lack evidence: %v", err)
			}
			var fields map[string]string
			if err := json.Unmarshal(item.Card, &fields); err != nil {
				t.Fatal(err)
			}
			if len(fields) != 11 {
				t.Fatalf("got %d card fields, want the 11 named fields", len(fields))
			}
			var evidence strings.Builder
			evidence.WriteString(source.Description)
			for _, answer := range source.Answers {
				evidence.WriteByte('\n')
				evidence.WriteString(answer.Answer)
			}
			for field, value := range fields {
				requireFixtureStrings(t, field, value)
				// These curated fixture fields intentionally quote their sources.
				// Production responses allow faithful paraphrases; this is not a
				// general word-matching rule for evaluating real model output.
				if field != "title" && field != "topic" && !strings.Contains(evidence.String(), value) {
					t.Errorf("curated factual field %q is not quoted from description or answers", field)
				}
			}
		})
	}
}

func TestTaskCardSchemaMatchesGoContract(t *testing.T) {
	var schema struct {
		Type       string `json:"type"`
		Properties map[string]struct {
			Type string `json:"type"`
		} `json:"properties"`
		Required             []string `json:"required"`
		AdditionalProperties *bool    `json:"additionalProperties"`
	}
	if err := json.Unmarshal(cardSchema, &schema); err != nil {
		t.Fatal(err)
	}
	if schema.Type != "object" || schema.AdditionalProperties == nil || *schema.AdditionalProperties {
		t.Fatal("card schema must explicitly forbid additional properties on an object")
	}
	want := []string{
		"title", "context", "need", "users", "data", "constraints",
		"expected_result", "success_criteria", "contact", "interaction_format", "topic",
	}
	raw, err := json.Marshal(TaskCard{})
	if err != nil {
		t.Fatal(err)
	}
	goFields, err := strictObject(raw)
	if err != nil || !exactKeys(goFields, want...) {
		t.Fatalf("TaskCard JSON differs from the named contract: %v", err)
	}
	if len(schema.Properties) != len(want) || len(schema.Required) != len(want) {
		t.Fatalf("schema property/required count must be %d", len(want))
	}
	required := make(map[string]bool)
	for _, field := range schema.Required {
		if required[field] {
			t.Errorf("duplicate schema required field %q", field)
		}
		required[field] = true
	}
	for _, field := range want {
		if !required[field] || schema.Properties[field].Type != "string" {
			t.Errorf("schema field %q must be required and string-typed", field)
		}
		if _, err := jsonString(goFields[field]); err != nil {
			t.Errorf("Go field %q must marshal as string: %v", field, err)
		}
	}
}

func TestEvaluationFixturesAreValidInputsWithReviewCriteria(t *testing.T) {
	cases := readFixtures[evaluationFixture](t, "evaluation_cases")
	if len(cases) < 11 {
		t.Fatalf("got %d evaluation cases, want coverage of at least 11 scenarios", len(cases))
	}
	seen := make(map[string]bool)
	for _, item := range cases {
		t.Run(item.ID, func(t *testing.T) {
			if strings.TrimSpace(item.ID) == "" || seen[item.ID] {
				t.Errorf("evaluation ID must be nonempty and unique: %q", item.ID)
			}
			seen[item.ID] = true
			if err := ValidateCardRequest(item.CardRequest); err != nil {
				t.Errorf("evaluation source is not a valid request: %v", err)
			}
			raw, err := json.Marshal(item.CardRequest)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ParseCardRequest(raw); err != nil {
				t.Errorf("evaluation description/answers cannot be sent to card endpoint: %v", err)
			}
			requireFixtureStrings(t, "expectations", item.Expectations...)
			requireFixtureStrings(t, "forbidden additions", item.ForbiddenAdditions...)
		})
	}
	// This validates the evaluation corpus, not real AI quality: no provider
	// has been called and none of these semantic expectations is auto-scored.
}
