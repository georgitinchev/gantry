import type { Metadata } from 'next';
import { StatusChip } from '@/components/ui';
import { getSnapshot, overall, runbookFor } from '@/lib/snapshot';
import Link from 'next/link';
import { repoUrl } from '@/lib/site';
import { ago } from '@/lib/time';

export const revalidate = 300;
export const metadata: Metadata = { title: 'Status', description: 'What the last platform run built and verified.' };

export default async function StatusPage() {
  const s = await getSnapshot();
  const state = overall(s);

  return (
    <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-12 sm:px-6">
      <p className="annot">Platform status</p>
      <div className="mt-2 flex flex-wrap items-end justify-between gap-4">
        <h1 className="text-3xl font-semibold tracking-tight sm:text-4xl">
          {state.ok ? 'Every check passed.' : `${state.failed + state.notReady} item(s) need attention.`}
        </h1>
        <StatusChip ok={state.ok}>{state.ok ? 'verified' : 'attention'}</StatusChip>
      </div>
      <p className="mt-3 max-w-2xl text-ink-2">
        Built from scratch on an ephemeral kind cluster {ago(s.generatedAt)}
        {s.bootstrapSeconds > 0 && <> · platform bootstrap took {Math.round(s.bootstrapSeconds / 60 * 10) / 10} min</>}.
        {s.source === 'sample' && ' Showing a recorded run until the nightly job publishes a fresh snapshot.'}
      </p>

      <dl className="mt-8 grid gap-px overflow-hidden rounded-lg border border-line bg-line sm:grid-cols-3">
        <Meta label="Commit">
          <a className="font-mono underline decoration-line underline-offset-4 hover:decoration-ink" href={`${repoUrl}/commit/${s.commit}`}>
            {s.commit.slice(0, 7)}
          </a>
        </Meta>
        <Meta label="Run">{s.runUrl ? <a className="underline decoration-line underline-offset-4 hover:decoration-ink" href={s.runUrl}>GitHub Actions log</a> : 'local run'}</Meta>
        <Meta label="Generated">
          <time dateTime={s.generatedAt}>{new Date(s.generatedAt).toUTCString().replace(' GMT', ' UTC')}</time>
        </Meta>
      </dl>

      <div className="mt-12 grid gap-10 lg:grid-cols-[1fr_1.3fr]">
        <section>
          <h2 className="annot">Checks</h2>
          <ul className="mt-3 divide-y divide-line rounded-lg border border-line">
            {s.checks.map((c) => (
              <li key={c.name} className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
                <span>{c.name}</span>
                <StatusChip ok={c.status === 'pass'}>{c.status}</StatusChip>
              </li>
            ))}
          </ul>

          {s.drills && s.drills.length > 0 && (
            <>
              <h2 className="annot mt-10">Failure drills</h2>
              <p className="mt-1 text-sm text-ink-2">Faults injected on purpose; each alert must fire and link to a runbook.</p>
              <ul className="mt-3 divide-y divide-line rounded-lg border border-line">
                {s.drills.map((d) => (
                  <li key={d.name} className="grid gap-1 px-4 py-3 text-sm">
                    <div className="flex items-center justify-between gap-3">
                      <span>{d.name}</span>
                      <StatusChip ok={d.status === 'pass'}>{d.status === 'pass' ? `${d.seconds}s` : 'fail'}</StatusChip>
                    </div>
                    {!d.alert && d.note && <span className="text-xs text-ink-2">{d.note}</span>}
                    {d.alert && (
                      <Link href={runbookFor(d.alert)} className="annot w-fit underline decoration-line underline-offset-4 hover:!text-ink">
                        {d.alert} → runbook
                      </Link>
                    )}
                  </li>
                ))}
              </ul>
            </>
          )}

          <h2 className="annot mt-10">Versions</h2>
          <dl className="mt-3 divide-y divide-line rounded-lg border border-line">
            {Object.entries(s.versions).map(([k, v]) => (
              <div key={k} className="flex justify-between px-4 py-2.5 text-sm">
                <dt className="text-ink-2">{k}</dt>
                <dd className="font-mono">{v}</dd>
              </div>
            ))}
          </dl>
        </section>

        <section>
          <h2 className="annot">Resources</h2>
          <ul className="mt-3 grid gap-4">
            {s.resources.map((r) => (
              <li key={`${r.kind}/${r.namespace}/${r.name}`} data-state={r.ready ? 'ready' : 'drift'} className="resource rounded-lg p-4">
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <div>
                    <p className="font-mono text-[0.625rem] uppercase tracking-wider text-ink-2">{r.kind}</p>
                    <p className="text-lg font-medium">
                      {r.namespace && <span className="text-ink-2">{r.namespace}/</span>}
                      {r.name}
                    </p>
                  </div>
                  <StatusChip ok={r.ready}>{r.ready ? 'ready' : 'not ready'}</StatusChip>
                </div>
                <dl className="mt-3 flex flex-wrap gap-x-6 gap-y-1 text-sm">
                  {Object.entries(r.spec).filter(([k]) => k !== 'crossplane').map(([k, v]) => (
                    <div key={k} className="flex gap-1.5">
                      <dt className="text-ink-2">{k}</dt>
                      <dd className="font-mono text-[0.8125rem]">{Array.isArray(v) ? v.join(', ') : String(v)}</dd>
                    </div>
                  ))}
                </dl>
                <ul className="mt-4 flex flex-wrap gap-1.5">
                  {r.composed.map((c) => (
                    <li key={`${c.kind}/${c.name}`} className="rounded border border-line bg-paper px-2 py-0.5 font-mono text-[0.6875rem]">
                      {c.kind}/{c.name}
                    </li>
                  ))}
                </ul>
              </li>
            ))}
          </ul>
        </section>
      </div>
    </main>
  );
}

function Meta({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="bg-paper px-4 py-3">
      <dt className="annot">{label}</dt>
      <dd className="mt-1 text-sm">{children}</dd>
    </div>
  );
}
