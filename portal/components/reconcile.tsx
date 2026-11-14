'use client';

import { useEffect, useRef, useState } from 'react';

type Item = { kind: string; name: string };

/**
 * The site's signature visual: a request on the left, the resources it
 * composes on the right. Each starts dashed (desired) and turns solid
 * (actual) as it reconciles. Data comes from a real snapshot.
 */
export function Reconcile({ request, items }: { request: string; items: Item[] }) {
  const [ready, setReady] = useState(0);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (reduce) {
      setReady(items.length);
      return;
    }
    let timer: ReturnType<typeof setInterval> | undefined;
    const io = new IntersectionObserver(([entry]) => {
      if (!entry.isIntersecting || timer) return;
      timer = setInterval(() => {
        setReady((n) => {
          if (n >= items.length) {
            clearInterval(timer);
            return n;
          }
          return n + 1;
        });
      }, 380);
    }, { threshold: 0.15 });
    if (ref.current) io.observe(ref.current);
    return () => {
      io.disconnect();
      clearInterval(timer);
    };
  }, [items.length]);

  const done = ready >= items.length;

  return (
    <div ref={ref} className="grid grid-cols-1 gap-4 md:grid-cols-[minmax(0,1fr)_auto_minmax(0,1.1fr)] md:items-center">
      <div className="min-w-0 rounded-lg border border-ink bg-card shadow-[4px_4px_0_0_var(--ink)]">
        <div className="flex items-center justify-between border-b border-line px-3 py-2">
          <span className="annot">requests/checkout/appenvironment.yaml</span>
        </div>
        <pre className="overflow-x-auto p-3 font-mono text-[0.75rem] leading-relaxed text-ink">{request}</pre>
      </div>

      <div aria-hidden="true" className="hidden flex-col items-center gap-1 md:flex">
        <span className="annot">reconcile</span>
        <svg width="56" height="12" viewBox="0 0 56 12" fill="none">
          <path d="M0 6h50" stroke="currentColor" strokeWidth="1.5" strokeDasharray="4 3" />
          <path d="M48 1l6 5-6 5" stroke="currentColor" strokeWidth="1.5" />
        </svg>
      </div>

      <div className="min-w-0">
        <div className="mb-2 flex items-center justify-between">
          <span className="annot">composed resources</span>
          <span className={`annot ${done ? '!text-signal' : ''}`} aria-live="polite">
            {done ? `Ready · ${items.length}/${items.length}` : `Creating · ${ready}/${items.length}`}
          </span>
        </div>
        <ul className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          {items.map((it, i) => {
            const isReady = i < ready;
            return (
              <li key={`${it.kind}/${it.name}`} data-state={isReady ? 'ready' : 'desired'} className="resource flex items-center justify-between gap-2 rounded-md px-3 py-2">
                <span className="min-w-0">
                  <span className="block font-mono text-[0.625rem] uppercase tracking-wider text-ink-2">{it.kind}</span>
                  <span className="block truncate text-sm">{it.name}</span>
                </span>
                <span aria-hidden="true" className={`size-2 shrink-0 rounded-full transition-colors ${isReady ? 'bg-signal' : 'border border-dashed border-ink/50'}`} />
              </li>
            );
          })}
        </ul>
      </div>
    </div>
  );
}
