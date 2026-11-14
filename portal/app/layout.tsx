import './globals.css';
import type { Metadata } from 'next';
import type { ReactNode } from 'react';
import { Instrument_Sans, Martian_Mono } from 'next/font/google';
import { RootProvider } from 'fumadocs-ui/provider/next';
import { site } from '@/lib/site';

const sans = Instrument_Sans({ subsets: ['latin'], variable: '--font-instrument' });
const mono = Martian_Mono({ subsets: ['latin'], variable: '--font-martian' });

export const metadata: Metadata = {
  title: { default: `${site.name} — ${site.tagline}`, template: `%s · ${site.name}` },
  description:
    'A small, self-service internal developer platform: Crossplane APIs, a policy gate, and proof on a real cluster for every pull request.',
  authors: [{ name: site.author.name, url: site.author.url }],
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" className={`${sans.variable} ${mono.variable}`} suppressHydrationWarning>
      <body className="flex min-h-screen flex-col font-sans antialiased">
        <RootProvider>{children}</RootProvider>
      </body>
    </html>
  );
}
