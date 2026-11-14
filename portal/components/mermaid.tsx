'use client';

import { useEffect, useId, useState } from 'react';
import { useTheme } from 'next-themes';

// Renders ```mermaid blocks with the site's own tokens, in both themes.
export function Mermaid({ chart }: { chart: string }) {
  const id = useId().replace(/[^a-zA-Z0-9]/g, '');
  const { resolvedTheme } = useTheme();
  const [svg, setSvg] = useState('');

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const { default: mermaid } = await import('mermaid');
      const css = getComputedStyle(document.documentElement);
      const v = (name: string) => css.getPropertyValue(name).trim();
      mermaid.initialize({
        startOnLoad: false,
        securityLevel: 'strict',
        fontFamily: 'var(--font-instrument), ui-sans-serif, sans-serif',
        theme: 'base',
        themeVariables: {
          darkMode: resolvedTheme === 'dark',
          background: v('--paper'),
          primaryColor: v('--card'),
          primaryTextColor: v('--ink'),
          primaryBorderColor: v('--ink'),
          secondaryColor: v('--paper-2'),
          tertiaryColor: v('--paper-2'),
          lineColor: v('--ink-2'),
          textColor: v('--ink'),
          clusterBkg: v('--paper-2'),
          clusterBorder: v('--line'),
          actorBkg: v('--card'),
          actorBorder: v('--ink'),
          signalColor: v('--ink'),
          noteBkgColor: v('--paper-2'),
          fontSize: '14px',
        },
      });
      const { svg } = await mermaid.render(`m${id}`, chart.replaceAll('\\n', '\n'));
      if (!cancelled) setSvg(svg);
    })().catch((err) => console.error('mermaid', err));
    return () => {
      cancelled = true;
    };
  }, [chart, id, resolvedTheme]);

  return (
    <div
      className="not-prose my-6 overflow-x-auto rounded-lg border border-line bg-card p-4 [&_svg]:mx-auto [&_svg]:h-auto [&_svg]:max-w-full"
      aria-label="diagram"
      dangerouslySetInnerHTML={{ __html: svg }}
    />
  );
}
