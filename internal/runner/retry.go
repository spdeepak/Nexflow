package runner

import (
	"context"
	"errors"
	"iter"
	"log/slog"

	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/model/openaimodel"
)

// defaultGenerateRetries is the number of times a single model generation is
// re-issued when the provider returns a response that contains no content.
const defaultGenerateRetries = 2

// retryingLLM wraps a model.LLM and transparently re-issues a generation that
// ends with an empty/no-content error before any content was yielded. Providers
// (e.g. Ollama's gemma4) intermittently return a response that translates to no
// text or tool parts; retrying the same call avoids failing the entire run over
// a transient empty output. Retrying only happens before anything was yielded,
// so no partial output is re-delivered.
type retryingLLM struct {
	inner   model.LLM
	retries int
}

// withGenerateRetry wraps model so its generations are retried on empty/no-content
// responses. m is returned unchanged when retries is non-positive or m is nil.
func withGenerateRetry(model model.LLM, retries int) model.LLM {
	if model == nil || retries <= 0 {
		return model
	}
	return &retryingLLM{inner: model, retries: retries}
}

func (m *retryingLLM) Name() string { return m.inner.Name() }

func (m *retryingLLM) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		for attempt := 0; attempt <= m.retries; attempt++ {
			yielded := 0
			shouldRetry := false
			for resp, err := range m.inner.GenerateContent(ctx, req, stream) {
				if err != nil && yielded == 0 && shouldRetryGeneration(err, attempt) {
					shouldRetry = true
					slog.WarnContext(ctx, "Retrying model generation after empty/no-content response",
						"model", m.Name(), "attempt", attempt+1, "error", err)
					break
				}
				yielded++
				if !yield(resp, err) {
					return
				}
			}
			if !shouldRetry {
				return
			}
		}
	}
}

// shouldRetryGeneration reports whether err is a retryable empty/no-content
// error and retries remain. Once any content has been yielded the attempt is
// never retried (the caller already received output).
func shouldRetryGeneration(err error, attempt int) bool {
	if attempt >= defaultGenerateRetries {
		return false
	}
	return errors.Is(err, openaimodel.ErrNoTextOrToolContent) || errors.Is(err, openaimodel.ErrNoOutputItems)
}
