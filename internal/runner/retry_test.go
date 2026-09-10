package runner

import (
	"context"
	"errors"
	"iter"
	"testing"

	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/model/openaimodel"
	"google.golang.org/genai"
)

type stubLLM struct {
	name     string
	calls    []bool // true: succeed, false: return ErrNoTextOrToolContent
	callIdx  int
	partials int // if >0, yield this many partial responses before the error
}

func (s *stubLLM) Name() string { return s.name }

func (s *stubLLM) GenerateContent(_ context.Context, _ *model.LLMRequest, _ bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		if s.callIdx >= len(s.calls) {
			yield(nil, errors.New("unexpected extra call"))
			return
		}
		call := s.calls[s.callIdx]
		s.callIdx++

		for i := 0; i < s.partials; i++ {
			if !yield(&model.LLMResponse{Partial: true}, nil) {
				return
			}
		}
		if !call {
			yield(nil, openaimodel.ErrNoTextOrToolContent)
			return
		}
		yield(&model.LLMResponse{Content: &genai.Content{Parts: []*genai.Part{{Text: "ok"}}}}, nil)
	}
}

func collect(t *testing.T, m model.LLM) (*model.LLMResponse, error) {
	t.Helper()
	var resp *model.LLMResponse
	var err error
	for r, e := range m.GenerateContent(context.Background(), &model.LLMRequest{}, true) {
		if e != nil {
			err = e
			break
		}
		resp = r
	}
	return resp, err
}

func TestWithGenerateRetryRetriesEmptyResponse(t *testing.T) {
	stub := &stubLLM{name: "stub", calls: []bool{false, false, true}}
	m := withGenerateRetry(stub, 2)

	resp, err := collect(t, m)
	if err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if stub.callIdx != 3 {
		t.Fatalf("expected 3 calls, got %d", stub.callIdx)
	}
	if resp.Content == nil || resp.Content.Parts[0].Text != "ok" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestWithGenerateRetryExhaustsRetries(t *testing.T) {
	stub := &stubLLM{name: "stub", calls: []bool{false, false, false}}
	m := withGenerateRetry(stub, 2)

	_, err := collect(t, m)
	if !errors.Is(err, openaimodel.ErrNoTextOrToolContent) {
		t.Fatalf("expected ErrNoTextOrToolContent after exhausting retries, got %v", err)
	}
	if stub.callIdx != 3 {
		t.Fatalf("expected 3 calls, got %d", stub.callIdx)
	}
}

func TestWithGenerateRetryDoesNotRetryAfterPartialOutput(t *testing.T) {
	stub := &stubLLM{name: "stub", calls: []bool{false}, partials: 1}
	m := withGenerateRetry(stub, 2)

	_, err := collect(t, m)
	if !errors.Is(err, openaimodel.ErrNoTextOrToolContent) {
		t.Fatalf("expected error not to be retried after partial output, got %v", err)
	}
	if stub.callIdx != 1 {
		t.Fatalf("expected single call (partial output delivered), got %d", stub.callIdx)
	}
}

func TestWithGenerateRetryReturnsNilWhenNoRetriesConfigured(t *testing.T) {
	stub := &stubLLM{name: "stub", calls: []bool{false, true}}
	if m := withGenerateRetry(stub, 0); m != stub {
		t.Fatal("expected original llm when retries is 0")
	}
	m := withGenerateRetry(nil, 2)
	if m != nil {
		t.Fatal("expected nil when wrapping nil llm")
	}
}
