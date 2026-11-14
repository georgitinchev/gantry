import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';
import { repoUrl } from './site';
import { Mark } from '@/components/mark';

export function baseOptions(): BaseLayoutProps {
  return {
    nav: {
      title: (
        <span className="inline-flex items-center gap-2 font-semibold tracking-tight">
          <Mark className="size-5" /> gantry
        </span>
      ),
    },
    githubUrl: repoUrl,
    links: [
      { text: 'Catalog', url: '/catalog' },
      { text: 'Status', url: '/status' },
      { text: 'Live demo', url: '/demo' },
      { text: 'Docs', url: '/docs' },
    ],
  };
}
