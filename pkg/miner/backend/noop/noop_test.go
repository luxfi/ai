// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package noop

import (
	"context"
	"testing"

	"github.com/luxfi/ai/pkg/miner/backend"
)

func TestName(t *testing.T) {
	if got := New().Name(); got != "noop" {
		t.Errorf("Name: got %q want %q", got, "noop")
	}
}

// TestChatMatchesLegacyStub pins the assistant content string so the
// miner-level behaviour is indistinguishable from the pre-refactor inline
// stub in miner.runChat.
func TestChatMatchesLegacyStub(t *testing.T) {
	b := New()
	resp, err := b.Chat(context.Background(), backend.ChatRequest{Model: "m"})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	const want = "I'm an AI assistant running on the Lux network."
	if resp.Content != want {
		t.Errorf("Chat content: got %q want %q", resp.Content, want)
	}
	if resp.Role != "assistant" {
		t.Errorf("Chat role: got %q want %q", resp.Role, "assistant")
	}
	if resp.Model != "m" {
		t.Errorf("Chat model: got %q want %q", resp.Model, "m")
	}
}

// TestInferenceMatchesLegacyStub pins the echo-prompt format used by the
// pre-refactor miner.runInference.
func TestInferenceMatchesLegacyStub(t *testing.T) {
	b := New()
	resp, err := b.Inference(context.Background(), backend.InferenceRequest{
		Model:  "m",
		Prompt: "what time is it?",
	})
	if err != nil {
		t.Fatalf("Inference: %v", err)
	}
	const want = "Response to: what time is it?"
	if resp.Text != want {
		t.Errorf("Inference text: got %q want %q", resp.Text, want)
	}
	if resp.Tokens != 10 {
		t.Errorf("Inference tokens: got %d want 10", resp.Tokens)
	}
}

// TestEmbedMatchesLegacyStub pins the zero-vector behaviour and the 384 dim.
func TestEmbedMatchesLegacyStub(t *testing.T) {
	b := New()
	resp, err := b.Embed(context.Background(), backend.EmbedRequest{Model: "m", Text: "x"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(resp.Embedding) != DefaultEmbeddingDims {
		t.Errorf("Embed dims: got %d want %d", len(resp.Embedding), DefaultEmbeddingDims)
	}
	for i, v := range resp.Embedding {
		if v != 0.0 {
			t.Errorf("Embed[%d]: got %v want 0.0", i, v)
			break
		}
	}
}

// TestCustomEmbeddingDims checks the dim knob (used by callers that want to
// mimic a real model's output shape in tests).
func TestCustomEmbeddingDims(t *testing.T) {
	b := &Backend{EmbeddingDims: 16}
	resp, err := b.Embed(context.Background(), backend.EmbedRequest{Text: "x"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(resp.Embedding) != 16 {
		t.Errorf("custom dims: got %d want 16", len(resp.Embedding))
	}
	if caps := b.Capabilities(); caps.EmbeddingDims != 16 {
		t.Errorf("Capabilities.EmbeddingDims: got %d want 16", caps.EmbeddingDims)
	}
}
