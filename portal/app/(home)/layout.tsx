import type { ReactNode } from 'react';
import { HomeLayout } from 'fumadocs-ui/layouts/home';
import { baseOptions } from '@/lib/layout.shared';
import { site } from '@/lib/site';

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <HomeLayout {...baseOptions()}>
      {children}
      <footer className="mt-auto border-t border-line">
        <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-3 px-4 py-6 text-sm text-ink-2 sm:px-6">
          <span>
            Built by{' '}
            <a className="text-ink underline decoration-line underline-offset-4 hover:decoration-ink" href={site.author.url}>
              {site.author.name}
            </a>{' '}
            · a portfolio project, not a production service
          </span>
          <span className="annot">Apache-2.0</span>
        </div>
      </footer>
    </HomeLayout>
  );
}
