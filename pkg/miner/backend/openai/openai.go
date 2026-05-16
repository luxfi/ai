// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package openai provides an InferenceBackend that speaks the OpenAI HTTP
// API. Because llama.cpp (server mode), vllm, ollama, and LocalAI all expose
// an OpenAI-compatible `/v1/chat/completions` + `/v1/embeddings` surface,
// this single adapter covers every local engine most operators run. Point
// BaseURL at `http://localhost:8080/v1` (llama.cpp) or `http://localhost:11434/v1`
// (ollama) and everything works.
//
// This adapter uses only the Go standard library — no SDK dependency.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/luxfi/ai/pkg/miner/backend"
)

const (
	// DefaultBaseURL targets the OpenAI public API. Override for local
	// engines (llama.cpp, vllm, ollama, LocalAI).
	DefaultBaseURL = "https://api.openai.com/v1"

	// DefaultTimeout is the per-request HTTP timeout used when the caller
	// does not supply an HTTPClient.
	DefaultTimeout = 60 * time.Second
)

// Config configures an OpenAI-compatible backend.
type Config struct {
	// BaseURL is the API root, e.g. "http://localhost:8080/v1". Trailing
	// slash is tolerated. Defaults to DefaultBaseURL.
	BaseURL string
	// APIKey is sent as "Authorization: Bearer <key>". Empty is fine for
	// local engines that do not check auth.
	APIKey string
	// Model is the default model name for requests whose own Model field
	// is empty.
	Model string
	// EmbeddingModel overrides Model for embedding requests (e.g.
	// "text-embedding-3-small" vs "gpt-4o-mini" for chat).
	EmbeddingModel string
	// HTTPClient is optional. When nil, a client with DefaultTimeout is
	// used.
	HTTPClient *http.Client
}

// Backend is the OpenAI-compatible InferenceBackend.
type Backend struct {
	cfg    Config
	client *http.Client

	// skipLegacyCompletions is latched to true the first time the
	// server tells us /completions is not available (HTTP 404 or 405).
	// Subsequent Inference calls go straight to the Chat fallback,
	// avoiding a useless round-trip on every task.
	skipLegacyCompletions atomic.Bool
}

// New returns a backend configured against cfg. If cfg.BaseURL is empty,
// DefaultBaseURL is used.
func New(cfg Config) *Backend {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	c := cfg.HTTPClient
	if c == nil {
		c = &http.Client{Timeout: DefaultTimeout}
	}
	return &Backend{cfg: cfg, client: c}
}

// StatusError reports a non-2xx HTTP response from the OpenAI-compatible
// endpoint. Callers can `errors.As` against it to react to specific
// status codes (e.g. 401 for re-auth, 429 for backoff).
type StatusError struct {
	// StatusCode is the HTTP response status code.
	StatusCode int
	// APIMessage is the structured `error.message` field returned in the
	// JSON body, if the server provided one. Empty for non-JSON errors.
	APIMessage string
	// RawBody is the verbatim response body. Used only in the fallback
	// rendering when the server didn't return a structured error.
	RawBody string
}

// Error matches the legacy non-typed error format so log scrapers and
// metric labels don't change when a caller upgrades.
func (e *StatusError) Error() string {
	if e.APIMessage != "" {
		return fmt.Sprintf("openai: %s (status %d)", e.APIMessage, e.StatusCode)
	}
	return fmt.Sprintf("openai: status %d: %s", e.StatusCode, strings.TrimSpace(e.RawBody))
}

// Name implements backend.InferenceBackend.
func (*Backend) Name() string { return "openai" }

// Capabilities implements backend.InferenceBackend.
func (*Backend) Capabilities() backend.Capabilities {
	return backend.Capabilities{
		Chat:      true,
		Inference: true,
		Embedding: true,
		// EmbeddingDims deliberately 0 — varies per model.
	}
}

// --- chat ---

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model     string        `json:"model"`
	Messages  []chatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens,omitempty"`
}

type chatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type chatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type chatCompletionResponse struct {
	ID      string                 `json:"id"`
	Model   string                 `json:"model"`
	Choices []chatCompletionChoice `json:"choices"`
	Usage   chatUsage              `json:"usage"`
}

// Chat implements backend.InferenceBackend.
func (b *Backend) Chat(ctx context.Context, req backend.ChatRequest) (backend.ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = b.cfg.Model
	}

	msgs := make([]chatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		msgs = append(msgs, chatMessage{Role: m.Role, Content: m.Content})
	}

	payload := chatCompletionRequest{
		Model:     model,
		Messages:  msgs,
		MaxTokens: req.MaxTokens,
	}

	var resp chatCompletionResponse
	if err := b.post(ctx, "/chat/completions", payload, &resp); err != nil {
		return backend.ChatResponse{}, err
	}

	if len(resp.Choices) == 0 {
		return backend.ChatResponse{}, errors.New("openai: chat response has no choices")
	}
	c := resp.Choices[0].Message
	return backend.ChatResponse{
		Role:    c.Role,
		Content: c.Content,
		Model:   resp.Model,
		Tokens:  resp.Usage.CompletionTokens,
	}, nil
}

// --- completion (legacy /completions endpoint) ---

type completionRequest struct {
	Model     string `json:"model"`
	Prompt    string `json:"prompt"`
	MaxTokens int    `json:"max_tokens,omitempty"`
}

type completionChoice struct {
	Index        int    `json:"index"`
	Text         string `json:"text"`
	FinishReason string `json:"finish_reason"`
}

type completionResponse struct {
	ID      string             `json:"id"`
	Model   string             `json:"model"`
	Choices []completionChoice `json:"choices"`
	Usage   chatUsage          `json:"usage"`
}

// Inference implements backend.InferenceBackend. It prefers the legacy
// /completions endpoint (still served by OpenAI, llama.cpp's `--legacy`,
// and older ollama/vllm builds). When the server returns 404/405 — modern
// vllm, recent ollama, LocalAI without the completions plugin — we fall
// back to /chat/completions with a single user message and remember the
// decision so subsequent tasks skip the /completions probe entirely.
//
// The fallback semantics match what callers had to implement by hand
// before: a one-message chat completion. Token counts and the response
// model name carry through unchanged.
func (b *Backend) Inference(ctx context.Context, req backend.InferenceRequest) (backend.InferenceResponse, error) {
	model := req.Model
	if model == "" {
		model = b.cfg.Model
	}

	if !b.skipLegacyCompletions.Load() {
		payload := completionRequest{
			Model:     model,
			Prompt:    req.Prompt,
			MaxTokens: req.MaxTokens,
		}
		var resp completionResponse
		err := b.post(ctx, "/completions", payload, &resp)
		switch {
		case err == nil:
			if len(resp.Choices) == 0 {
				return backend.InferenceResponse{}, errors.New("openai: completion response has no choices")
			}
			return backend.InferenceResponse{
				Text:   resp.Choices[0].Text,
				Tokens: resp.Usage.CompletionTokens,
				Model:  resp.Model,
			}, nil
		case isLegacyCompletionsMissing(err):
			b.skipLegacyCompletions.Store(true)
			// fall through to /chat/completions fallback
		default:
			return backend.InferenceResponse{}, err
		}
	}

	chat, err := b.Chat(ctx, backend.ChatRequest{
		Model:     model,
		Messages:  []backend.Message{{Role: "user", Content: req.Prompt}},
		MaxTokens: req.MaxTokens,
	})
	if err != nil {
		return backend.InferenceResponse{}, err
	}
	return backend.InferenceResponse{
		Text:   chat.Content,
		Tokens: chat.Tokens,
		Model:  chat.Model,
	}, nil
}

// isLegacyCompletionsMissing reports whether err indicates the server
// has no /completions endpoint. We treat 404 (Not Found) and 405 (Method
// Not Allowed) as missing — both shapes appear in the wild depending on
// the router. Any other status, including 4xx auth failures, is real.
func isLegacyCompletionsMissing(err error) bool {
	var se *StatusError
	if !errors.As(err, &se) {
		return false
	}
	return se.StatusCode == http.StatusNotFound || se.StatusCode == http.StatusMethodNotAllowed
}

// --- embeddings ---

type embeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embeddingDatum struct {
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

type embeddingResponse struct {
	Model string           `json:"model"`
	Data  []embeddingDatum `json:"data"`
}

// Embed implements backend.InferenceBackend.
func (b *Backend) Embed(ctx context.Context, req backend.EmbedRequest) (backend.EmbedResponse, error) {
	model := req.Model
	if model == "" {
		model = b.cfg.EmbeddingModel
	}
	if model == "" {
		model = b.cfg.Model
	}

	payload := embeddingRequest{Model: model, Input: req.Text}

	var resp embeddingResponse
	if err := b.post(ctx, "/embeddings", payload, &resp); err != nil {
		return backend.EmbedResponse{}, err
	}
	if len(resp.Data) == 0 {
		return backend.EmbedResponse{}, errors.New("openai: embedding response has no data")
	}
	return backend.EmbedResponse{
		Embedding: resp.Data[0].Embedding,
		Model:     resp.Model,
	}, nil
}

// --- HTTP plumbing ---

func (b *Backend) post(ctx context.Context, path string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("openai: encode request: %w", err)
	}

	url := b.cfg.BaseURL + path
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("openai: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if b.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+b.cfg.APIKey)
	}

	resp, err := b.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("openai: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("openai: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to surface a useful error message. OpenAI-compatible servers
		// return {"error": {"message": "..."}}; we fall back to the raw
		// body when parsing fails.
		var errEnv struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		statusErr := &StatusError{StatusCode: resp.StatusCode, RawBody: string(respBody)}
		if jerr := json.Unmarshal(respBody, &errEnv); jerr == nil && errEnv.Error.Message != "" {
			statusErr.APIMessage = errEnv.Error.Message
		}
		return statusErr
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("openai: decode response: %w", err)
	}
	return nil
}
