// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luxfi/ai/pkg/miner/backend"
)

func TestName(t *testing.T) {
	if got := New(Config{}).Name(); got != "openai" {
		t.Errorf("Name: got %q want %q", got, "openai")
	}
}

func TestChatHappyPath(t *testing.T) {
	var gotAuth, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl-xyz",
			"model": "gpt-test",
			"choices": [{
				"index": 0,
				"message": {"role": "assistant", "content": "hi there"},
				"finish_reason": "stop"
			}],
			"usage": {"prompt_tokens": 3, "completion_tokens": 5, "total_tokens": 8}
		}`))
	}))
	defer srv.Close()

	b := New(Config{BaseURL: srv.URL, APIKey: "sk-test", Model: "gpt-default"})
	resp, err := b.Chat(context.Background(), backend.ChatRequest{
		Model:    "gpt-test",
		Messages: []backend.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if gotPath != "/chat/completions" {
		t.Errorf("request path: got %q want %q", gotPath, "/chat/completions")
	}
	if gotAuth != "Bearer sk-test" {
		t.Errorf("auth header: got %q want %q", gotAuth, "Bearer sk-test")
	}
	if !strings.Contains(gotBody, `"model":"gpt-test"`) {
		t.Errorf("request body missing model: %s", gotBody)
	}
	if !strings.Contains(gotBody, `"role":"user"`) {
		t.Errorf("request body missing user message: %s", gotBody)
	}
	if resp.Content != "hi there" {
		t.Errorf("response content: got %q want %q", resp.Content, "hi there")
	}
	if resp.Tokens != 5 {
		t.Errorf("response tokens: got %d want 5", resp.Tokens)
	}
	if resp.Model != "gpt-test" {
		t.Errorf("response model: got %q want %q", resp.Model, "gpt-test")
	}
}

func TestChatDefaultsModelFromConfig(t *testing.T) {
	var sawModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		sawModel, _ = req["model"].(string)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer srv.Close()

	b := New(Config{BaseURL: srv.URL, Model: "configured-default"})
	_, err := b.Chat(context.Background(), backend.ChatRequest{})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if sawModel != "configured-default" {
		t.Errorf("model default fallback: got %q want %q", sawModel, "configured-default")
	}
}

func TestChatErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key","type":"auth_error"}}`))
	}))
	defer srv.Close()

	b := New(Config{BaseURL: srv.URL, APIKey: "bad"})
	_, err := b.Chat(context.Background(), backend.ChatRequest{Model: "m"})
	if err == nil {
		t.Fatal("expected error on 401")
	}
	if !strings.Contains(err.Error(), "invalid api key") {
		t.Errorf("error should surface server message, got: %v", err)
	}
}

func TestChatEmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()

	b := New(Config{BaseURL: srv.URL})
	_, err := b.Chat(context.Background(), backend.ChatRequest{Model: "m"})
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestInference(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/completions" {
			t.Errorf("path: got %q want /completions", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model": "inf-m",
			"choices": [{"text": "the answer"}],
			"usage": {"completion_tokens": 2, "total_tokens": 5}
		}`))
	}))
	defer srv.Close()

	b := New(Config{BaseURL: srv.URL})
	resp, err := b.Inference(context.Background(), backend.InferenceRequest{
		Model:  "inf-m",
		Prompt: "q?",
	})
	if err != nil {
		t.Fatalf("Inference: %v", err)
	}
	if resp.Text != "the answer" {
		t.Errorf("text: got %q", resp.Text)
	}
	if resp.Tokens != 2 {
		t.Errorf("tokens: got %d want 2", resp.Tokens)
	}
}

func TestEmbed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Errorf("path: got %q want /embeddings", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model": "emb-m",
			"data": [{"index": 0, "embedding": [0.1, 0.2, 0.3]}]
		}`))
	}))
	defer srv.Close()

	b := New(Config{BaseURL: srv.URL, EmbeddingModel: "emb-m"})
	resp, err := b.Embed(context.Background(), backend.EmbedRequest{Text: "hi"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(resp.Embedding) != 3 || resp.Embedding[2] != 0.3 {
		t.Errorf("embedding: got %v", resp.Embedding)
	}
	if resp.Model != "emb-m" {
		t.Errorf("model: got %q", resp.Model)
	}
}

func TestBaseURLTrailingSlash(t *testing.T) {
	b := New(Config{BaseURL: "http://example.com/v1/"})
	if b.cfg.BaseURL != "http://example.com/v1" {
		t.Errorf("trailing slash not stripped: %q", b.cfg.BaseURL)
	}
}

func TestNoAuthHeaderWhenAPIKeyEmpty(t *testing.T) {
	var sawAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer srv.Close()

	b := New(Config{BaseURL: srv.URL})
	_, _ = b.Chat(context.Background(), backend.ChatRequest{Model: "m"})
	if sawAuth != "" {
		t.Errorf("expected no Authorization header when APIKey empty, got %q", sawAuth)
	}
}

func TestCapabilities(t *testing.T) {
	caps := New(Config{}).Capabilities()
	if !caps.Chat || !caps.Inference || !caps.Embedding {
		t.Errorf("capabilities: %+v", caps)
	}
}
