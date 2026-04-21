// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package noop provides a deterministic in-process InferenceBackend. It
// preserves the placeholder behaviour that was inlined into
// pkg/miner/miner.go before the backend interface landed, so existing tests
// and downstream consumers that rely on "some response comes back" keep
// working with zero configuration.
//
// This backend performs no real inference. Use it for local dev, CI, and as
// the safe default when an operator hasn't configured a real engine.
package noop

import (
	"context"
	"fmt"

	"github.com/luxfi/ai/pkg/miner/backend"
)

// DefaultEmbeddingDims matches the placeholder dim used by the pre-refactor
// embedding stub in pkg/miner/miner.go (len(embedding) == 384).
const DefaultEmbeddingDims = 384

// Backend is a deterministic mock implementing backend.InferenceBackend.
type Backend struct {
	// EmbeddingDims controls the length of the zero-vector returned by
	// Embed. Defaults to DefaultEmbeddingDims when zero.
	EmbeddingDims int
}

// New returns a new noop backend with default embedding dimensionality.
func New() *Backend {
	return &Backend{EmbeddingDims: DefaultEmbeddingDims}
}

// Name implements backend.InferenceBackend.
func (*Backend) Name() string { return "noop" }

// Capabilities implements backend.InferenceBackend.
func (b *Backend) Capabilities() backend.Capabilities {
	dims := b.EmbeddingDims
	if dims == 0 {
		dims = DefaultEmbeddingDims
	}
	return backend.Capabilities{
		Chat:          true,
		Inference:     true,
		Embedding:     true,
		EmbeddingDims: dims,
	}
}

// Chat returns a fixed assistant message. Identical to the string returned by
// the previous inline stub in miner.runChat.
func (b *Backend) Chat(_ context.Context, req backend.ChatRequest) (backend.ChatResponse, error) {
	return backend.ChatResponse{
		Role:    "assistant",
		Content: "I'm an AI assistant running on the Lux network.",
		Model:   req.Model,
	}, nil
}

// Inference echoes the prompt, matching the previous inline stub in
// miner.runInference.
func (b *Backend) Inference(_ context.Context, req backend.InferenceRequest) (backend.InferenceResponse, error) {
	return backend.InferenceResponse{
		Text:   fmt.Sprintf("Response to: %s", req.Prompt),
		Tokens: 10,
		Model:  req.Model,
	}, nil
}

// Embed returns a zero vector of EmbeddingDims length — byte-for-byte identical
// to the pre-refactor miner.runEmbedding placeholder.
func (b *Backend) Embed(_ context.Context, req backend.EmbedRequest) (backend.EmbedResponse, error) {
	dims := b.EmbeddingDims
	if dims == 0 {
		dims = DefaultEmbeddingDims
	}
	vec := make([]float64, dims)
	return backend.EmbedResponse{
		Embedding: vec,
		Model:     req.Model,
	}, nil
}
