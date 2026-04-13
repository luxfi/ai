// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package cli provides the "ai" subcommand for the lux CLI.
// It exposes AI operations: chat, completion, model listing, and agent execution
// against the Lux AI network (api.hanzo.ai gateway or local lux-ai node).
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

const defaultEndpoint = "http://localhost:9090"

// NewCmd returns the "ai" command tree for the lux CLI.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai",
		Short: "AI operations (chat, agents, models)",
		Long: `The ai command provides tools for interacting with the Lux AI network,
including chat completion, model listing, and agent execution.

Connects to a local lux-ai node or the Hanzo AI gateway (api.hanzo.ai).

ENDPOINTS:

  Local:    http://localhost:9090 (default, via lux ai serve)
  Gateway:  https://api.hanzo.ai/v1

ENVIRONMENT:

  LUX_AI_ENDPOINT   Override the default AI endpoint
  LUX_AI_API_KEY    API key for gateway authentication`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newChatCmd())
	cmd.AddCommand(newCompleteCmd())
	cmd.AddCommand(newModelsCmd())
	cmd.AddCommand(newAgentCmd())

	return cmd
}

func endpoint() string {
	if ep := os.Getenv("LUX_AI_ENDPOINT"); ep != "" {
		return ep
	}
	return defaultEndpoint
}

func apiKey() string {
	return os.Getenv("LUX_AI_API_KEY")
}

func newChatCmd() *cobra.Command {
	var (
		model  string
		system string
	)
	cmd := &cobra.Command{
		Use:   "chat [message]",
		Short: "Send a chat message",
		Long: `Send a chat message to the AI model and print the response.

Examples:
  lux ai chat "What is the Lux network?"
  lux ai chat --model qwen3-8b "Explain post-quantum cryptography"
  lux ai chat --system "You are a blockchain expert" "What is BFT?"`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			messages := []map[string]string{}
			if system != "" {
				messages = append(messages, map[string]string{"role": "system", "content": system})
			}
			messages = append(messages, map[string]string{"role": "user", "content": args[0]})

			body := map[string]interface{}{
				"model":    model,
				"messages": messages,
			}
			data, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("marshal request: %w", err)
			}

			req, err := http.NewRequest("POST", endpoint()+"/v1/chat/completions", strings.NewReader(string(data)))
			if err != nil {
				return fmt.Errorf("create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")
			if key := apiKey(); key != "" {
				req.Header.Set("Authorization", "Bearer "+key)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				respBody, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
			}

			var result struct {
				Choices []struct {
					Message struct {
						Content string `json:"content"`
					} `json:"message"`
				} `json:"choices"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}

			if len(result.Choices) > 0 {
				fmt.Println(result.Choices[0].Message.Content)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&model, "model", "qwen3-8b", "Model to use")
	cmd.Flags().StringVar(&system, "system", "", "System prompt")
	return cmd
}

func newCompleteCmd() *cobra.Command {
	var (
		model     string
		maxTokens int
	)
	cmd := &cobra.Command{
		Use:   "complete [prompt]",
		Short: "Text completion",
		Long: `Generate a text completion from a prompt.

Examples:
  lux ai complete "The Lux blockchain uses"
  lux ai complete --model zen-coder-1.5b --max-tokens 256 "func main() {"`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			messages := []map[string]string{
				{"role": "user", "content": args[0]},
			}
			body := map[string]interface{}{
				"model":    model,
				"messages": messages,
			}
			if maxTokens > 0 {
				body["max_tokens"] = maxTokens
			}
			data, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("marshal request: %w", err)
			}

			req, err := http.NewRequest("POST", endpoint()+"/v1/chat/completions", strings.NewReader(string(data)))
			if err != nil {
				return fmt.Errorf("create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")
			if key := apiKey(); key != "" {
				req.Header.Set("Authorization", "Bearer "+key)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				respBody, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
			}

			var result struct {
				Choices []struct {
					Message struct {
						Content string `json:"content"`
					} `json:"message"`
				} `json:"choices"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}

			if len(result.Choices) > 0 {
				fmt.Println(result.Choices[0].Message.Content)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&model, "model", "qwen3-8b", "Model to use")
	cmd.Flags().IntVar(&maxTokens, "max-tokens", 0, "Maximum tokens to generate (0 = model default)")
	return cmd
}

func newModelsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "models",
		Short: "List available models",
		Long: `List all models available on the connected AI endpoint.

Examples:
  lux ai models
  LUX_AI_ENDPOINT=https://api.hanzo.ai lux ai models`,
		RunE: func(_ *cobra.Command, _ []string) error {
			req, err := http.NewRequest("GET", endpoint()+"/v1/models", nil)
			if err != nil {
				return fmt.Errorf("create request: %w", err)
			}
			if key := apiKey(); key != "" {
				req.Header.Set("Authorization", "Bearer "+key)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				respBody, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
			}

			var result struct {
				Data []struct {
					ID      string `json:"id"`
					OwnedBy string `json:"owned_by"`
				} `json:"data"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tOWNER")
			for _, m := range result.Data {
				fmt.Fprintf(w, "%s\t%s\n", m.ID, m.OwnedBy)
			}
			return w.Flush()
		},
	}
}

func newAgentCmd() *cobra.Command {
	var (
		model string
	)
	cmd := &cobra.Command{
		Use:   "agent [task]",
		Short: "Run an AI agent",
		Long: `Run an AI agent that executes a task using the connected AI endpoint.

The agent sends the task as a system-prompted chat request with
tool-use capabilities when supported by the model.

Examples:
  lux ai agent "Deploy a new EVM chain on devnet"
  lux ai agent --model qwen3-8b "Analyze the validator set"`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			messages := []map[string]string{
				{"role": "system", "content": "You are a Lux blockchain operations agent. Execute the requested task step by step."},
				{"role": "user", "content": args[0]},
			}
			body := map[string]interface{}{
				"model":    model,
				"messages": messages,
			}
			data, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("marshal request: %w", err)
			}

			req, err := http.NewRequest("POST", endpoint()+"/v1/chat/completions", strings.NewReader(string(data)))
			if err != nil {
				return fmt.Errorf("create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")
			if key := apiKey(); key != "" {
				req.Header.Set("Authorization", "Bearer "+key)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				respBody, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
			}

			var result struct {
				Choices []struct {
					Message struct {
						Content string `json:"content"`
					} `json:"message"`
				} `json:"choices"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}

			if len(result.Choices) > 0 {
				fmt.Println(result.Choices[0].Message.Content)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&model, "model", "qwen3-8b", "Model to use")
	return cmd
}
