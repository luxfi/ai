// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package backend defines a pluggable inference-engine interface used by the
// miner. Backends translate miner task inputs into model outputs; this package
// deliberately imports nothing outside the stdlib so adapters (llama.cpp,
// vllm, ollama, remote OpenAI-compatible endpoints, etc.) can be dropped in
// without bloating the miner binary.
//
// The interface is intentionally minimal — it mirrors what the miner actually
// does in runInference, runChat, and runEmbedding.
package backend

import "context"

// Message is a single chat turn. Shape matches OpenAI chat messages and the
// miner's internal message type.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is a multi-turn chat prompt.
type ChatRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

// ChatResponse is the assistant's reply.
type ChatResponse struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Model   string `json:"model"`
	Tokens  int    `json:"tokens,omitempty"`
}

// InferenceRequest is a single-prompt completion request.
type InferenceRequest struct {
	Model     string `json:"model"`
	Prompt    string `json:"prompt"`
	MaxTokens int    `json:"max_tokens,omitempty"`
}

// InferenceResponse is the completion output.
type InferenceResponse struct {
	Text   string `json:"text"`
	Tokens int    `json:"tokens"`
	Model  string `json:"model"`
}

// EmbedRequest asks for a vector embedding of a piece of text.
type EmbedRequest struct {
	Model string `json:"model"`
	Text  string `json:"text"`
}

// EmbedResponse carries the embedding vector.
type EmbedResponse struct {
	Embedding []float64 `json:"embedding"`
	Model     string    `json:"model"`
}

// Capabilities reports what a backend can do. Consumers use this to pick a
// backend or to skip tasks a backend cannot serve.
type Capabilities struct {
	Chat      bool `json:"chat"`
	Inference bool `json:"inference"`
	Embedding bool `json:"embedding"`
	// EmbeddingDims, when non-zero, declares a fixed output dimensionality for
	// embeddings; 0 means the backend decides per-request.
	EmbeddingDims int `json:"embedding_dims,omitempty"`
}

// InferenceBackend is the pluggable compute layer for the miner.
//
// Implementations must be safe for concurrent use — the miner's task worker
// pool may invoke any method from multiple goroutines.
type InferenceBackend interface {
	// Name returns a short identifier ("noop", "openai", ...). Used in logs
	// and config matching.
	Name() string

	// Capabilities reports what this backend supports.
	Capabilities() Capabilities

	// Chat runs a multi-turn chat completion.
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)

	// Inference runs a single-prompt completion.
	Inference(ctx context.Context, req InferenceRequest) (InferenceResponse, error)

	// Embed produces an embedding vector for the given text.
	Embed(ctx context.Context, req EmbedRequest) (EmbedResponse, error)
}
