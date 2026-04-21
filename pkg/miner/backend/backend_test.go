// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package backend_test

import (
	"context"
	"testing"

	"github.com/luxfi/ai/pkg/miner/backend"
	"github.com/luxfi/ai/pkg/miner/backend/noop"
)

// TestInterfaceContract ensures the canonical in-tree backends implement
// backend.InferenceBackend. If this test stops compiling, a backend has
// drifted from the interface.
func TestInterfaceContract(t *testing.T) {
	var _ backend.InferenceBackend = noop.New()
}

// TestCapabilitiesShape exercises the Capabilities reporting path end-to-end
// for a concrete backend. The point is to lock in the JSON-visible shape;
// any rename breaks the wire contract for API consumers.
func TestCapabilitiesShape(t *testing.T) {
	caps := noop.New().Capabilities()
	if !caps.Chat || !caps.Inference || !caps.Embedding {
		t.Fatalf("noop backend should advertise all three capabilities: %+v", caps)
	}
	if caps.EmbeddingDims == 0 {
		t.Fatal("noop backend should advertise a fixed embedding dimensionality")
	}
}

// TestRequestResponseRoundTrip sanity-checks that every request/response pair
// travels through the interface with no surprise mutations.
func TestRequestResponseRoundTrip(t *testing.T) {
	ctx := context.Background()
	b := noop.New()

	chat, err := b.Chat(ctx, backend.ChatRequest{
		Model:    "test-model",
		Messages: []backend.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if chat.Model != "test-model" {
		t.Errorf("Chat: model not preserved, got %q", chat.Model)
	}

	inf, err := b.Inference(ctx, backend.InferenceRequest{
		Model:  "inf-model",
		Prompt: "hello",
	})
	if err != nil {
		t.Fatalf("Inference: %v", err)
	}
	if inf.Model != "inf-model" {
		t.Errorf("Inference: model not preserved, got %q", inf.Model)
	}

	emb, err := b.Embed(ctx, backend.EmbedRequest{
		Model: "emb-model",
		Text:  "some text",
	})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if emb.Model != "emb-model" {
		t.Errorf("Embed: model not preserved, got %q", emb.Model)
	}
}
