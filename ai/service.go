package ai

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type Service struct {
	provider      Provider
	timeout       time.Duration
	maxAttempts   int
	forceFallback bool
	retryDelay    time.Duration
}

// NewService with nil provider constructs the live OpenAI client. A missing
// credential or model will yield a visible fallback, without network traffic.
func NewService(cfg Config, provider Provider) (*Service, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if provider == nil {
		provider = NewOpenAIProvider(cfg)
	}
	return &Service{provider: provider, timeout: cfg.Timeout, maxAttempts: cfg.MaxAttempts, forceFallback: cfg.ForceFallback, retryDelay: 150 * time.Millisecond}, nil
}

func (s *Service) Questions(ctx context.Context, req QuestionsRequest) (QuestionsResponse, Mode, error) {
	if err := ValidateQuestionsRequest(req); err != nil {
		return QuestionsResponse{}, "", err
	}
	input, _ := json.Marshal(req)
	var response QuestionsResponse
	ok, err := s.generate(ctx, GenerationRequest{Name: "clarification_questions", SystemPrompt: questionsPrompt, Input: input, Schema: questionsSchema}, func(raw []byte) error {
		var err error
		response, err = ValidateQuestions(raw)
		return err
	})
	if err != nil {
		return QuestionsResponse{}, "", err
	}
	if ok {
		return response, ModeLive, nil
	}
	return FallbackQuestions(req.Description), ModeFallback, nil
}

func (s *Service) Card(ctx context.Context, req CardRequest) (TaskCard, Mode, error) {
	if err := ValidateCardRequest(req); err != nil {
		return TaskCard{}, "", err
	}
	input, _ := json.Marshal(req)
	var response TaskCard
	ok, err := s.generate(ctx, GenerationRequest{Name: "task_card", SystemPrompt: cardPrompt, Input: input, Schema: cardSchema}, func(raw []byte) error {
		var err error
		response, err = ValidateCard(raw)
		if err != nil {
			return err
		}
		return CheckGrounding(req, response)
	})
	if err != nil {
		return TaskCard{}, "", err
	}
	if ok {
		return response, ModeLive, nil
	}
	return FallbackCard(req), ModeFallback, nil
}

func (s *Service) generate(parent context.Context, req GenerationRequest, validate func([]byte) error) (bool, error) {
	if err := parent.Err(); err != nil {
		return false, err
	}
	if s.forceFallback {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(parent, s.timeout)
	defer cancel()
	for attempt := 0; attempt < s.maxAttempts; attempt++ {
		if err := parent.Err(); err != nil {
			return false, err
		}
		if ctx.Err() != nil {
			break
		}
		raw, err := s.provider.Generate(ctx, req)
		if parent.Err() != nil {
			return false, parent.Err()
		}
		if ctx.Err() != nil {
			break
		}
		if err == nil {
			if err = validate(raw); err == nil {
				return true, nil
			}
			// Invalid model outputs get at most one fresh attempt. Neither the
			// rejected content nor the error is inserted into the next prompt.
			err = &ProviderError{Code: "invalid_output", Retryable: true}
		}
		var providerErr *ProviderError
		if !errors.As(err, &providerErr) || !providerErr.Retryable || attempt+1 >= s.maxAttempts {
			break
		}
		timer := time.NewTimer(s.retryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
	if err := parent.Err(); err != nil {
		return false, err
	}
	return false, nil
}
