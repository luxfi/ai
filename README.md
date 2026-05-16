# Lux AI

Mine AI tokens and chat with AI models on the Lux AI network.

## Overview

Lux AI is a specialized AI compute network that:
- **AI Mining**: Earn AI coins by contributing GPU compute for AI inference
- **AI Chat**: Chat with AI models running on the decentralized network
- **Q-Chain Integration**: Shared attestations and quantum security via Q-Chain
- **Cross-Network**: AI proofs can mint on any teleport-supported network (Lux, Hanzo, Zoo)
- **OpenAI-Compatible API**: Drop-in replacement for OpenAI API

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Q-Chain                                 │
│  (Shared attestations, quantum security, AI proof generation)   │
└──────────────┬────────────────┬────────────────┬────────────────┘
               │                │                │
       ┌───────▼───────┐ ┌──────▼──────┐ ┌───────▼───────┐
       │   Lux AI      │ │  Hanzo AI   │ │    Zoo AI     │
       │   Network     │ │  Network    │ │   Network     │
       └───────────────┘ └─────────────┘ └───────────────┘
               │                │                │
               └────────────────┼────────────────┘
                                │
                    ┌───────────▼───────────┐
                    │   Teleport Bridge     │
                    │  (Exit to any chain)  │
                    └───────────────────────┘
```

## Components

### lux-ai

Unified node binary that:
- Provides OpenAI-compatible chat API (`/v1/chat/completions`)
- Coordinates miners and distributes tasks
- Runs inference on GPU/CPU
- Generates AI proofs for Q-Chain attestation
- Tracks rewards and task completion

### lux-desktop

Tauri-based desktop application with:
- AI chat interface
- Miner status and control
- Wallet integration
- Model selection

## Quick Start

### Run Lux AI

```bash
# Build and run
make run

# Or manually
go build -o bin/lux-ai ./cmd/lux-ai
./bin/lux-ai -port 9090
```

### Run the Desktop App

```bash
# Development mode
cd desktop
pnpm install
pnpm tauri:dev

# Build for production
make build-desktop
```

## API Reference

### Chat Completion (OpenAI-compatible)

```bash
curl -X POST http://localhost:9090/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "zen-mini-0.5b",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

### List Models

```bash
curl http://localhost:9090/v1/models
```

### Miner Registration

```bash
curl -X POST http://localhost:9090/api/miners/register \
  -H "Content-Type: application/json" \
  -d '{
    "id": "miner-001",
    "wallet_address": "0x...",
    "endpoint": "http://localhost:8888",
    "gpu_enabled": true
  }'
```

### Stats

```bash
curl http://localhost:9090/api/stats
```

## Available Models

| Model | Parameters | Context | Capabilities |
|-------|-----------|---------|--------------|
| zen-coder-1.5b | 1.5B | 32K | Code, Chat |
| zen-mini-0.5b | 0.5B | 8K | Chat |
| qwen3-8b | 8B | 128K | Chat, Code, Reasoning |

## Earnings Model

Miners earn AI coins for:
- **Inference Tasks**: Variable rate based on model size and tokens
- **Uptime Bonus**: 10% bonus for 99.9% uptime
- **Speed Bonus**: 5% bonus for sub-100ms latency

AI proofs are generated and attested on Q-Chain, then can be minted on:
- **Lux Network**: Native rewards + ecosystem incentives
- **Hanzo Network**: Native rewards + ecosystem incentives  
- **Zoo Network**: Native rewards + ecosystem incentives

## Development

### Prerequisites

- Go 1.24+
- Node.js 20+
- pnpm
- Rust (for Tauri)

### Build

```bash
make build
```

### Run Tests

```bash
make test
```

## License

Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
