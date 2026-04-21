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

// Inference implements backend.InferenceBackend by calling /completions. Most
// OpenAI-compatible local servers support this path; when they don't, the
// caller should route inference tasks through Chat with a single user message
// instead.
func (b *Backend) Inference(ctx context.Context, req backend.InferenceRequest) (backend.InferenceResponse, error) {
	model := req.Model
	if model == "" {
		model = b.cfg.Model
	}

	payload := completionRequest{
		Model:     model,
		Prompt:    req.Prompt,
		MaxTokens: req.MaxTokens,
	}

	var resp completionResponse
	if err := b.post(ctx, "/completions", payload, &resp); err != nil {
		return backend.InferenceResponse{}, err
	}

	if len(resp.Choices) == 0 {
		return backend.InferenceResponse{}, errors.New("openai: completion response has no choices")
	}
	return backend.InferenceResponse{
		Text:   resp.Choices[0].Text,
		Tokens: resp.Usage.CompletionTokens,
		Model:  resp.Model,
	}, nil
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
		if jerr := json.Unmarshal(respBody, &errEnv); jerr == nil && errEnv.Error.Message != "" {
			return fmt.Errorf("openai: %s (status %d)", errEnv.Error.Message, resp.StatusCode)
		}
		return fmt.Errorf("openai: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("openai: decode response: %w", err)
	}
	return nil
}
