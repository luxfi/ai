// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/luxfi/ai/pkg/miner"
)

var (
	version = "0.1.0"
)

func main() {
	var (
		walletAddr  = flag.String("wallet", "", "Wallet address for rewards")
		nodeURL     = flag.String("node", "http://localhost:9650", "Lux node URL")
		apiPort     = flag.Int("port", 8888, "Local API port")
		gpuEnabled  = flag.Bool("gpu", true, "Enable GPU acceleration")
		modelDir    = flag.String("models", "./models", "Model directory")
		cacheSize   = flag.Int64("cache", 10*1024*1024*1024, "Cache size in bytes")
		showVersion = flag.Bool("version", false, "Show version")
	)

	flag.Parse()

	if *showVersion {
		fmt.Printf("lux-ai-miner %s\n", version)
		os.Exit(0)
	}

	if *walletAddr == "" {
		fmt.Fprintln(os.Stderr, "Error: wallet address required (-wallet)")
		flag.Usage()
		os.Exit(1)
	}

	config := miner.Config{
		WalletAddress: *walletAddr,
		NodeURL:       *nodeURL,
		GPUEnabled:    *gpuEnabled,
		MaxTasks:      10,
		CacheSize:     *cacheSize,
		ModelDir:      *modelDir,
		APIPort:       *apiPort,
	}

	m := miner.New(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
		_ = m.Stop()
	}()

	fmt.Printf("Starting Lux AI Miner %s\n", version)
	fmt.Printf("Wallet: %s\n", *walletAddr)
	fmt.Printf("Node: %s\n", *nodeURL)
	fmt.Printf("API Port: %d\n", *apiPort)
	fmt.Printf("GPU Enabled: %v\n", *gpuEnabled)

	if err := m.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting miner: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Miner started. Press Ctrl+C to stop.")

	// Wait for context cancellation
	<-ctx.Done()
	fmt.Println("Miner stopped.")
}
