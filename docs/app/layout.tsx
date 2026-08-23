import './global.css';
import { RootProvider } from 'fumadocs-ui/provider';
import { ZenMono } from '@hanzo/font/mono';
import { ZenSans } from '@hanzo/font/sans';
import type { ReactNode } from 'react';

export const metadata = {
  title: {
    default: 'Lux AI - Decentralized AI Compute Network',
    template: '%s | Lux AI',
  },
  description: 'GPU mining for AI inference. Earn rewards by contributing compute to the decentralized AI network.',
  keywords: ['Lux', 'AI', 'GPU mining', 'decentralized', 'inference', 'OpenAI', 'compute', 'blockchain'],
  authors: [{ name: 'Lux Network' }],
  openGraph: {
    title: 'Lux AI - Decentralized AI Compute Network',
    description: 'GPU mining for AI inference. Earn rewards by contributing compute.',
    type: 'website',
  },
};

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" className={`${ZenSans.variable} ${ZenMono.variable}`} suppressHydrationWarning>
      <body className="min-h-svh bg-background font-sans antialiased">
        <RootProvider
          search={{
            enabled: true,
          }}
          theme={{
            enabled: true,
            defaultTheme: 'dark',
            storageKey: 'lux-ai-theme',
          }}
        >
          <div className="relative flex min-h-svh flex-col bg-background">
            {children}
          </div>
        </RootProvider>
      </body>
    </html>
  );
}
