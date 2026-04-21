# pkg/miner/backend

Pluggable inference-engine seam for the Lux AI miner.

The miner used to inline three `// TODO: Integrate with actual ...` stubs for
chat, inference, and embedding. Those stubs are now hidden behind a small
interface so operators can point the miner at whichever engine they run
without touching the mining binary.

```go
type InferenceBackend interface {
    Name() string
    Capabilities() Capabilities
    Chat(ctx context.Context, req ChatRequest)          (ChatResponse, error)
    Inference(ctx context.Context, req InferenceRequest) (InferenceResponse, error)
    Embed(ctx context.Context, req EmbedRequest)         (EmbedResponse, error)
}
```

## Backends shipped in-tree

| Package                          | Name     | Use case                                              |
|----------------------------------|----------|-------------------------------------------------------|
| `pkg/miner/backend/noop`         | `noop`   | Deterministic mock. Default. Zero config, zero deps.  |
| `pkg/miner/backend/openai`       | `openai` | OpenAI-compatible HTTP adapter (stdlib `net/http`).   |

`noop` preserves the pre-refactor placeholder output (`"Response to: <prompt>"`,
`"I'm an AI assistant running on the Lux network."`, 384-dim zero-vector
embeddings) so existing tests and downstream consumers see no behaviour change.

`openai` works against any server that speaks the OpenAI HTTP dialect — which
happens to cover all the local engines operators actually run:

| Engine    | `OPENAI_API_BASE` target     | Notes                         |
|-----------|------------------------------|-------------------------------|
| llama.cpp | `http://localhost:8080/v1`   | `./server --port 8080`        |
| vllm      | `http://localhost:8000/v1`   | `vllm serve <model>`          |
| ollama    | `http://localhost:11434/v1`  | Native OpenAI compat endpoint |
| LocalAI   | `http://localhost:8080/v1`   | Drop-in OpenAI replacement    |
| OpenAI    | `https://api.openai.com/v1`  | The real thing                |

One adapter, five engines. No new Go deps.

## Wiring

Via `Config`:

```go
cfg := miner.DefaultConfig()
cfg.Backend = "openai"
cfg.OpenAIBase = "http://localhost:11434/v1"  // ollama
cfg.OpenAIModel = "llama3.1"
m := miner.New(cfg)
```

Or via `WithBackend` for fully custom plumbing (e.g. a direct MLX/CUDA binding
from your own `main`):

```go
m := miner.New(cfg).WithBackend(myBackend)
```

Unknown `Backend` values fall back to `noop` instead of failing — operator
typos show up in logs as `name=noop` without crash-looping the miner.

## Writing a new backend

Implement `InferenceBackend` in your own module and pass it via
`WithBackend`. Contract:

- All methods must be safe for concurrent use.
- `Capabilities()` should be cheap and pure.
- `Name()` should be a short, stable identifier (`"noop"`, `"openai"`, etc.).
- Return errors rather than panicking on upstream failures; the miner marks
  tasks failed and bumps `Stats.TasksFailed`.

## Why OpenAI-compatible instead of direct bindings

llama.cpp bindings pull in ~20 MB of C source. vllm is Python-only. MLX
bindings require CGo. The OpenAI HTTP contract lets one small Go adapter
cover every engine an operator would reasonably run, with zero new
dependencies in `go.mod`. A direct-binding backend can still be added in
a future PR for latency-sensitive deployments — the interface supports it.
