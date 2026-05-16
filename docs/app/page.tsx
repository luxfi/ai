import Link from 'next/link';
import { Logo } from '../components/logo';
import {
  Cpu,
  Zap,
  Shield,
  DollarSign,
  Server,
  Code,
  ArrowRight,
  GitBranch,
  Activity,
  Lock,
} from 'lucide-react';

export default function HomePage() {
  return (
    <div className="min-h-screen">
      {/* Header */}
      <header className="border-b border-border">
        <div className="container mx-auto px-4 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Logo size={32} />
            <span className="font-bold text-xl">Lux AI</span>
          </div>
          <nav className="flex items-center gap-6 text-sm">
            <Link href="/docs" className="text-muted-foreground hover:text-foreground transition-colors">
              Documentation
            </Link>
            <Link href="/docs/mining" className="text-muted-foreground hover:text-foreground transition-colors">
              Mining
            </Link>
            <Link href="/docs/api" className="text-muted-foreground hover:text-foreground transition-colors">
              API
            </Link>
            <a
              href="https://github.com/luxfi/ai"
              target="_blank"
              rel="noopener noreferrer"
              className="text-muted-foreground hover:text-foreground transition-colors"
            >
              GitHub
            </a>
          </nav>
        </div>
      </header>

      {/* Hero */}
      <section className="py-24 px-4 relative overflow-hidden">
        <div className="absolute inset-0 bg-gradient-to-b from-primary/5 to-transparent pointer-events-none" />
        <div className="container mx-auto max-w-4xl text-center relative">
          <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full border border-primary/20 bg-primary/5 text-sm text-primary mb-8">
            <Activity className="size-4" />
            Decentralized AI Compute Network
          </div>
          <h1 className="text-5xl md:text-6xl font-bold mb-6 tracking-tight">
            GPU Mining for{' '}
            <span className="text-gradient-cyan">AI Inference</span>
          </h1>
          <p className="text-xl text-muted-foreground mb-8 max-w-2xl mx-auto">
            Contribute your GPU power to the decentralized AI network.
            Earn rewards by running AI inference workloads with OpenAI-compatible APIs.
          </p>
          <div className="flex items-center justify-center gap-4 flex-wrap">
            <Link
              href="/docs/getting-started"
              className="inline-flex items-center gap-2 rounded-lg bg-primary text-primary-foreground px-6 py-3 font-medium hover:opacity-90 transition-opacity"
            >
              <Cpu className="size-4" />
              Start Mining
            </Link>
            <Link
              href="/docs/api"
              className="inline-flex items-center gap-2 rounded-lg border border-border px-6 py-3 font-medium hover:bg-accent transition-colors"
            >
              <Code className="size-4" />
              API Reference
            </Link>
          </div>
        </div>
      </section>

      {/* Key Features */}
      <section className="py-20 px-4 border-y border-border bg-card">
        <div className="container mx-auto max-w-6xl">
          <div className="grid md:grid-cols-4 gap-8">
            <div className="text-center">
              <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center mx-auto mb-4">
                <Cpu className="size-8 text-primary" />
              </div>
              <h3 className="font-semibold mb-2">GPU Mining</h3>
              <p className="text-sm text-muted-foreground">
                Mine with any modern GPU. NVIDIA, AMD, and Apple Silicon supported.
              </p>
            </div>
            <div className="text-center">
              <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center mx-auto mb-4">
                <Code className="size-8 text-primary" />
              </div>
              <h3 className="font-semibold mb-2">OpenAI Compatible</h3>
              <p className="text-sm text-muted-foreground">
                Drop-in replacement for OpenAI APIs. Use existing SDKs and tools.
              </p>
            </div>
            <div className="text-center">
              <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center mx-auto mb-4">
                <Shield className="size-8 text-primary" />
              </div>
              <h3 className="font-semibold mb-2">Attestation System</h3>
              <p className="text-sm text-muted-foreground">
                Cryptographic proof of inference quality. Trust scores for miners.
              </p>
            </div>
            <div className="text-center">
              <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center mx-auto mb-4">
                <DollarSign className="size-8 text-primary" />
              </div>
              <h3 className="font-semibold mb-2">Earn Rewards</h3>
              <p className="text-sm text-muted-foreground">
                Get paid in LUX tokens for successful inference completions.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* How It Works */}
      <section className="py-20 px-4">
        <div className="container mx-auto max-w-6xl">
          <h2 className="text-3xl font-bold text-center mb-4">How It Works</h2>
          <p className="text-center text-muted-foreground mb-12 max-w-2xl mx-auto">
            A decentralized network where GPU miners serve AI inference requests
            and earn rewards based on quality and performance.
          </p>
          <div className="grid md:grid-cols-3 gap-8">
            <div className="p-6 rounded-lg border border-border bg-card">
              <div className="w-10 h-10 rounded-lg bg-primary text-primary-foreground flex items-center justify-center font-bold mb-4">
                1
              </div>
              <h3 className="text-lg font-semibold mb-2">Register Your GPU</h3>
              <p className="text-muted-foreground text-sm">
                Install the Lux AI miner software and register your GPU with the network.
                Stake LUX tokens to participate in mining.
              </p>
            </div>
            <div className="p-6 rounded-lg border border-border bg-card">
              <div className="w-10 h-10 rounded-lg bg-primary text-primary-foreground flex items-center justify-center font-bold mb-4">
                2
              </div>
              <h3 className="text-lg font-semibold mb-2">Process Requests</h3>
              <p className="text-muted-foreground text-sm">
                Receive AI inference requests from the network. Run LLM completions,
                embeddings, and other AI workloads on your hardware.
              </p>
            </div>
            <div className="p-6 rounded-lg border border-border bg-card">
              <div className="w-10 h-10 rounded-lg bg-primary text-primary-foreground flex items-center justify-center font-bold mb-4">
                3
              </div>
              <h3 className="text-lg font-semibold mb-2">Earn Rewards</h3>
              <p className="text-muted-foreground text-sm">
                Get attestations for completed work. Earn LUX tokens proportional
                to your contributions and trust score.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Architecture */}
      <section className="py-20 px-4 border-t border-border bg-card">
        <div className="container mx-auto max-w-6xl">
          <h2 className="text-3xl font-bold text-center mb-4">Network Architecture</h2>
          <p className="text-center text-muted-foreground mb-12 max-w-2xl mx-auto">
            Built on the Lux Network with cryptographic attestations and decentralized coordination.
          </p>
          <div className="grid md:grid-cols-2 gap-8">
            <div className="space-y-6">
              <div className="flex gap-4">
                <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
                  <Server className="size-5 text-primary" />
                </div>
                <div>
                  <h3 className="font-semibold mb-1">Decentralized Inference</h3>
                  <p className="text-sm text-muted-foreground">
                    No central servers. Requests are routed to available miners based on
                    capacity, trust scores, and latency requirements.
                  </p>
                </div>
              </div>
              <div className="flex gap-4">
                <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
                  <Lock className="size-5 text-primary" />
                </div>
                <div>
                  <h3 className="font-semibold mb-1">Attestation Protocol</h3>
                  <p className="text-sm text-muted-foreground">
                    Cryptographic proofs verify inference quality. Challenge-response
                    mechanisms detect fraudulent or low-quality outputs.
                  </p>
                </div>
              </div>
              <div className="flex gap-4">
                <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
                  <GitBranch className="size-5 text-primary" />
                </div>
                <div>
                  <h3 className="font-semibold mb-1">Multi-Model Support</h3>
                  <p className="text-sm text-muted-foreground">
                    Run any open model - Llama, Mistral, Qwen, and more.
                    Miners choose which models to serve based on hardware.
                  </p>
                </div>
              </div>
              <div className="flex gap-4">
                <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
                  <Zap className="size-5 text-primary" />
                </div>
                <div>
                  <h3 className="font-semibold mb-1">Low Latency Routing</h3>
                  <p className="text-sm text-muted-foreground">
                    Geographic and performance-based routing ensures fast responses.
                    Edge nodes minimize round-trip times.
                  </p>
                </div>
              </div>
            </div>
            <div className="rounded-lg border border-border bg-background p-6">
              <pre className="text-xs overflow-x-auto">
{`// Example: OpenAI-compatible API call
const response = await fetch('https://api.lux.ai/v1/chat/completions', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer YOUR_API_KEY',
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    model: 'llama-3.3-70b',
    messages: [
      { role: 'user', content: 'Hello, world!' }
    ],
    temperature: 0.7,
  }),
});

const data = await response.json();
console.log(data.choices[0].message.content);`}
              </pre>
            </div>
          </div>
        </div>
      </section>

      {/* Quick Links */}
      <section className="py-20 px-4 border-t border-border">
        <div className="container mx-auto max-w-6xl">
          <h2 className="text-3xl font-bold text-center mb-12">Quick Links</h2>
          <div className="grid md:grid-cols-3 gap-4">
            <Link
              href="/docs/getting-started"
              className="p-6 rounded-lg border border-border hover:border-primary/50 hover:bg-accent/50 transition-colors group"
            >
              <Cpu className="size-6 text-primary mb-3" />
              <h3 className="font-semibold mb-2 group-hover:text-primary transition-colors">
                Getting Started
              </h3>
              <p className="text-sm text-muted-foreground">
                Install the miner and start contributing compute in minutes.
              </p>
              <div className="flex items-center gap-1 text-sm text-primary mt-4">
                Read guide <ArrowRight className="size-4" />
              </div>
            </Link>
            <Link
              href="/docs/mining/hardware"
              className="p-6 rounded-lg border border-border hover:border-primary/50 hover:bg-accent/50 transition-colors group"
            >
              <Server className="size-6 text-primary mb-3" />
              <h3 className="font-semibold mb-2 group-hover:text-primary transition-colors">
                Hardware Requirements
              </h3>
              <p className="text-sm text-muted-foreground">
                GPU specifications and recommended configurations for mining.
              </p>
              <div className="flex items-center gap-1 text-sm text-primary mt-4">
                View specs <ArrowRight className="size-4" />
              </div>
            </Link>
            <Link
              href="/docs/api/openai"
              className="p-6 rounded-lg border border-border hover:border-primary/50 hover:bg-accent/50 transition-colors group"
            >
              <Code className="size-6 text-primary mb-3" />
              <h3 className="font-semibold mb-2 group-hover:text-primary transition-colors">
                API Reference
              </h3>
              <p className="text-sm text-muted-foreground">
                OpenAI-compatible API endpoints and usage examples.
              </p>
              <div className="flex items-center gap-1 text-sm text-primary mt-4">
                View API <ArrowRight className="size-4" />
              </div>
            </Link>
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="py-20 px-4 border-t border-border bg-card">
        <div className="container mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold mb-4">Ready to Start Mining?</h2>
          <p className="text-muted-foreground mb-8">
            Join the decentralized AI network and earn rewards for your GPU contributions.
          </p>
          <div className="flex items-center justify-center gap-4">
            <Link
              href="/docs/getting-started"
              className="inline-flex items-center gap-2 rounded-lg bg-primary text-primary-foreground px-6 py-3 font-medium hover:opacity-90 transition-opacity"
            >
              Get Started
              <ArrowRight className="size-4" />
            </Link>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-8 px-4 border-t border-border">
        <div className="container mx-auto flex items-center justify-between text-sm text-muted-foreground">
          <div className="flex items-center gap-2">
            <Logo size={20} />
            <span>Lux AI</span>
          </div>
          <div className="flex items-center gap-6">
            <a href="https://lux.network" target="_blank" rel="noopener noreferrer" className="hover:text-foreground">
              Lux Network
            </a>
            <a href="https://docs.lux.network" target="_blank" rel="noopener noreferrer" className="hover:text-foreground">
              Docs
            </a>
            <a href="https://github.com/luxfi/ai" target="_blank" rel="noopener noreferrer" className="hover:text-foreground">
              GitHub
            </a>
            <a href="https://discord.gg/lux" target="_blank" rel="noopener noreferrer" className="hover:text-foreground">
              Discord
            </a>
          </div>
        </div>
      </footer>
    </div>
  );
}
