import Link from 'next/link';
import type { ComponentProps, ReactNode } from 'react';

type ButtonProps = ComponentProps<typeof Link> & { variant?: 'solid' | 'outline' };

export function ButtonLink({ variant = 'solid', className = '', ...props }: ButtonProps) {
  const base =
    'inline-flex items-center gap-2 rounded-md px-4 py-2 text-sm font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-signal';
  const styles =
    variant === 'solid'
      ? 'bg-ink text-paper hover:bg-ink/85'
      : 'border border-dashed border-ink/50 text-ink hover:border-solid hover:border-ink';
  return <Link className={`${base} ${styles} ${className}`} {...props} />;
}

export function StatusChip({ ok, children }: { ok: boolean; children: ReactNode }) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 font-mono text-[0.6875rem] font-medium uppercase tracking-wider ${
        ok ? 'bg-signal-fill text-signal-ink' : 'bg-drift-fill text-ink'
      }`}
    >
      <span aria-hidden="true" className={`size-1.5 rounded-full ${ok ? 'bg-signal-ink' : 'bg-drift'}`} />
      {children}
    </span>
  );
}

export function Section({ label, title, children, className = '' }: { label: string; title: ReactNode; children: ReactNode; className?: string }) {
  return (
    <section className={`mx-auto w-full max-w-6xl px-4 py-16 sm:px-6 ${className}`}>
      <p className="annot">{label}</p>
      <h2 className="mt-2 max-w-2xl text-2xl font-semibold tracking-tight sm:text-3xl">{title}</h2>
      <div className="mt-8">{children}</div>
    </section>
  );
}
