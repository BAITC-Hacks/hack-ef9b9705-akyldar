package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

// MaxProviderResponseBytes bounds the entire upstream JSON envelope, not only
// the generated text. Limits are enforced even when Content-Length is absent.
const MaxProviderResponseBytes = 256 << 10

// ProviderError exposes only a stable diagnostic code. Upstream bodies, request
// data, URLs and credentials must never be included in this error.
type ProviderError struct {
	Code       string
	Retryable  bool
	StatusCode int
}

func (e *ProviderError) Error() string { return e.Code }

// OpenAIProvider makes a single Responses API request per Generate call. The
// service owns retry count and the overall operation deadline.
type OpenAIProvider struct {
	apiKey   string
	model    string
	endpoint string
	client   *http.Client
}

func NewOpenAIProvider(cfg Config) *OpenAIProvider {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &OpenAIProvider{
		apiKey:   strings.TrimSpace(cfg.APIKey),
		model:    strings.TrimSpace(cfg.Model),
		endpoint: "https://api.openai.com/v1/responses",
		client: &http.Client{
			Timeout: timeout,
			// A redirect would create another request and could disclose the key.
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

type openAIRequest struct {
	Model           string               `json:"model"`
	Instructions    string               `json:"instructions"`
	Input           []openAIInputMessage `json:"input"`
	Text            openAITextFormat     `json:"text"`
	Store           bool                 `json:"store"`
	MaxOutputTokens int                  `json:"max_output_tokens"`
}

type openAIInputMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAITextFormat struct {
	Format openAIJSONSchema `json:"format"`
}

type openAIJSONSchema struct {
	Type   string          `json:"type"`
	Name   string          `json:"name"`
	Strict bool            `json:"strict"`
	Schema json.RawMessage `json:"schema"`
}

type openAIResponse struct {
	Status            string          `json:"status"`
	Error             json.RawMessage `json:"error"`
	IncompleteDetails json.RawMessage `json:"incomplete_details"`
	Output            []struct {
		Type    string `json:"type"`
		Status  string `json:"status"`
		Content []struct {
			Type    string `json:"type"`
			Text    string `json:"text"`
			Refusal string `json:"refusal"`
		} `json:"content"`
	} `json:"output"`
}

func (p *OpenAIProvider) Generate(ctx context.Context, input GenerationRequest) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.apiKey == "" {
		return nil, &ProviderError{Code: "missing_api_key"}
	}
	if p.model == "" {
		return nil, &ProviderError{Code: "missing_model"}
	}
	if strings.ContainsAny(p.apiKey, "\r\n") || strings.TrimSpace(input.Name) == "" ||
		strings.TrimSpace(input.SystemPrompt) == "" || !providerJSONObject(input.Input) || !providerJSONObject(input.Schema) {
		return nil, &ProviderError{Code: "provider_invalid_request"}
	}

	payload, err := json.Marshal(openAIRequest{
		Model:        p.model,
		Instructions: input.SystemPrompt,
		// Description and answers remain data in a separate user message.
		Input: []openAIInputMessage{{Role: "user", Content: string(input.Input)}},
		Text: openAITextFormat{Format: openAIJSONSchema{
			Type: "json_schema", Name: input.Name, Strict: true, Schema: input.Schema,
		}},
		Store:           false,
		MaxOutputTokens: 2400,
	})
	if err != nil {
		return nil, &ProviderError{Code: "provider_invalid_request"}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, &ProviderError{Code: "provider_invalid_request"}
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, providerConnectionError(ctx, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, providerStatusError(resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxProviderResponseBytes+1))
	if err != nil {
		return nil, providerConnectionError(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(body) > MaxProviderResponseBytes {
		return nil, &ProviderError{Code: "provider_response_too_large", Retryable: true}
	}
	var response openAIResponse
	if !utf8.Valid(body) || json.Unmarshal(body, &response) != nil {
		return nil, &ProviderError{Code: "invalid_output", Retryable: true}
	}
	if providerNonNull(response.Error) {
		return nil, &ProviderError{Code: "provider_error", Retryable: true}
	}
	if response.Status == "" {
		return nil, &ProviderError{Code: "invalid_output", Retryable: true}
	}
	if response.Status != "completed" || providerNonNull(response.IncompleteDetails) {
		return nil, &ProviderError{Code: "provider_incomplete"}
	}

	var output strings.Builder
	for _, item := range response.Output {
		if item.Type == "refusal" {
			return nil, &ProviderError{Code: "provider_refusal"}
		}
		if item.Type != "message" {
			continue
		}
		if item.Status != "" && item.Status != "completed" {
			return nil, &ProviderError{Code: "provider_incomplete"}
		}
		for _, content := range item.Content {
			if content.Type == "refusal" || content.Refusal != "" {
				return nil, &ProviderError{Code: "provider_refusal"}
			}
			if content.Type == "output_text" {
				output.WriteString(content.Text)
			}
		}
	}
	if strings.TrimSpace(output.String()) == "" {
		return nil, &ProviderError{Code: "invalid_output", Retryable: true}
	}
	// The service applies the business JSON schema and semantic checks.
	return []byte(output.String()), nil
}

func providerJSONObject(value []byte) bool {
	value = bytes.TrimSpace(value)
	return len(value) > 0 && value[0] == '{' && utf8.Valid(value) && json.Valid(value)
}

func providerNonNull(value json.RawMessage) bool {
	value = bytes.TrimSpace(value)
	return len(value) > 0 && !bytes.Equal(value, []byte("null"))
}

func providerConnectionError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &ProviderError{Code: "provider_timeout", Retryable: true}
	}
	return &ProviderError{Code: "provider_unavailable", Retryable: true}
}

func providerStatusError(status int) *ProviderError {
	err := &ProviderError{Code: "provider_http_error", StatusCode: status}
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		err.Code = "provider_auth"
	case status == http.StatusTooManyRequests:
		err.Code, err.Retryable = "provider_rate_limit", true
	case status >= 500 && status <= 599:
		err.Code, err.Retryable = "provider_unavailable", true
	case status >= 300 && status <= 399:
		err.Code = "provider_redirect"
	}
	return err
}
