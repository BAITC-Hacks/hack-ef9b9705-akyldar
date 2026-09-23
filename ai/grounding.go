package ai

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

var anchorPatterns = []*regexp.Regexp{
	regexp.MustCompile(`[\p{N}]+(?:[.,][\p{N}]+)*`),
	regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`),
	regexp.MustCompile(`(?i)https?://[^\s<>"\\]+`),
	regexp.MustCompile(`@[A-Za-z0-9_]{3,}`),
}

// This intentionally small set detects some invented technology names.
// It is not a ban on words absent from the source, and cannot prove grounding.
var technologyPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bexcel\b`),
	regexp.MustCompile(`(?i)\bpython\b`),
	regexp.MustCompile(`(?i)\breact(?:js)?\b`),
	regexp.MustCompile(`(?i)\bpostgres(?:ql)?\b`),
	regexp.MustCompile(`(?i)\b(?:golang|go)\b`),
	regexp.MustCompile(`(?i)\bopenai\b`),
	regexp.MustCompile(`(?i)\b(?:javascript|typescript)\b`),
	regexp.MustCompile(`(?i)\b(?:mysql|mongodb)\b`),
	regexp.MustCompile(`(?i)\b(?:docker|kubernetes)\b`),
}

func normalizeAnchor(s string) string {
	s = strings.TrimRight(s, ".,;:!?)]}")
	return strings.ReplaceAll(s, ",", ".")
}

// CheckGrounding checks literal numbers and contact anchors plus selected
// technology families. Questions are excluded from evidence. It cannot detect
// changed relations, facts expressed in words, negation loss or all injections.
// A false rejection safely leads to a manual fallback; acceptance still needs
// human review against description and answers.
func CheckGrounding(req CardRequest, card TaskCard) error {
	var evidence strings.Builder
	evidence.WriteString(req.Description)
	for _, pair := range req.Answers {
		evidence.WriteByte('\n')
		evidence.WriteString(pair.Answer)
	}
	raw, _ := json.Marshal(card)
	var fields map[string]string
	_ = json.Unmarshal(raw, &fields)
	source := evidence.String()
	for _, pattern := range anchorPatterns {
		allowed := map[string]bool{}
		for _, match := range pattern.FindAllString(source, -1) {
			allowed[normalizeAnchor(match)] = true
		}
		for _, value := range fields {
			for _, match := range pattern.FindAllString(value, -1) {
				if !allowed[normalizeAnchor(match)] {
					return errors.New("unsupported factual anchor")
				}
			}
		}
	}
	for _, pattern := range technologyPatterns {
		if pattern.MatchString(source) {
			continue
		}
		for _, value := range fields {
			if pattern.MatchString(value) {
				return errors.New("unsupported technology")
			}
		}
	}
	return nil
}
