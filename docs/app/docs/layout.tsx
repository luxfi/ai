import { DocsLayout } from 'fumadocs-ui/layouts/docs';
import type { ReactNode } from 'react';
import { Cpu, Code, DollarSign, Shield, BookOpen, ExternalLink, GitBranch } from 'lucide-react';
import { Logo } from '../../components/logo';
import { source } from '@/lib/source';

export default function Layout({ children }: { children: ReactNode }) {
  const pageTree = source.getPageTree();

  return (
    <DocsLayout
      tree={pageTree}
      nav={{
        title: (
          <div className="flex items-center gap-2">
            <Logo size={28} />
            <span className="font-bold">Lux AI</span>
          </div>
        ),
        transparentMode: 'top',
      }}
      sidebar={{
        defaultOpenLevel: 1,
        banner: (
          <div className="rounded-lg border border-primary/20 bg-primary/5 p-4">
            <div className="flex items-center gap-2 mb-2">
              <Cpu className="size-4 text-primary" />
              <span className="text-sm font-semibold">Quick Start</span>
            </div>
            <p className="text-xs text-muted-foreground mb-3">
              Get mining in under 5 minutes
            </p>
            <a
              href="/docs/getting-started"
              className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
            >
              Start mining <ExternalLink className="size-3" />
            </a>
          </div>
        ),
        footer: (
          <div className="flex flex-col gap-3 p-4 text-xs border-t border-border">
            <a
              href="https://github.com/luxfi/ai"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-2 text-muted-foreground hover:text-foreground transition-colors"
            >
              <GitBranch className="size-4" />
              GitHub Repository
            </a>
            <a
              href="https://discord.gg/lux"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-2 text-muted-foreground hover:text-foreground transition-colors"
            >
              <BookOpen className="size-4" />
              Discord Community
            </a>
            <a
              href="https://docs.lux.network"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-2 text-muted-foreground hover:text-foreground transition-colors"
            >
              <BookOpen className="size-4" />
              Network Docs
            </a>
          </div>
        ),
      }}
      links={[
        {
          text: 'Mining',
          url: '/docs/mining',
          icon: <Cpu className="size-4" />,
        },
        {
          text: 'API',
          url: '/docs/api',
          icon: <Code className="size-4" />,
        },
        {
          text: 'Rewards',
          url: '/docs/rewards',
          icon: <DollarSign className="size-4" />,
        },
        {
          text: 'Attestation',
          url: '/docs/attestation',
          icon: <Shield className="size-4" />,
        },
        {
          text: 'GitHub',
          url: 'https://github.com/luxfi/ai',
          icon: <ExternalLink className="size-4" />,
          external: true,
        },
      ]}
    >
      {children}
    </DocsLayout>
  );
}
