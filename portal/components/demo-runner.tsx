'use client';

import { useCallback, useEffect, useState } from 'react';
import type { DemoEvent, DemoRecord, DemoRequest } from '@/lib/demo/request';
import { StatusChip } from './ui';

type User = { login: string; avatar: string } | null;

const PHASES: { id: DemoEvent['phase']; label: string }[] = [
  { id: 'queued', label: 'Runner queued' },
  { id: 'cluster', label: 'kind cluster' },
  { id: 'platform', label: 'Platform installed' },
  { id: 'lint', label: 'Request linted' },
  { id: 'apply', label: 'Request applied' },
  { id: 'reconcile', label: 'Reconciled' },
  { id: 'verify', label: 'Verified' },
];

export function DemoRunner({ enabled, user }: { enabled: boolean; user: User }) {
  const [req, setReq] = useState<DemoRequest>({ size: 'small', exposure: 'internal', postgres: true, version: '17' });
  const [runId, setRunId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [starting, setStarting] = useState(false);
  const [run, setRun] = useState<{ record: DemoRecord; events: DemoEvent[] } | null>(null);
  const [expired, setExpired] = useState(false);

  useEffect(() => {
    const id = new URLSearchParams(window.location.search).get('run');
    if (id && /^[a-f0-9]{16}$/.test(id)) setRunId(id);
  }, []);

  const poll = useCallback(async (id: string) => {
    const res = await fetch(`/api/demo/${id}`, { cache: 'no-store' });
    if (res.ok) setRun(await res.json());
    else if (res.status === 404) setExpired(true);
  }, []);

  useEffect(() => {
    if (!runId) return;
    poll(runId);
    const finished = run?.events.some((e) => e.phase === 'done' || e.phase === 'failed');
    if (finished || expired) return;
    const t = setInterval(() => poll(runId), 4000);
    return () => clearInterval(t);
  }, [runId, poll, run?.events, expired]);

  const start = async () => {
    setStarting(true);
    setError(null);
    const res = await fetch('/api/demo', { method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify(req) });
    const body = await res.json().catch(() => ({}));
    setStarting(false);
    if (!res.ok) {
      const wait = body.retryAfterSeconds ? ` Try again in ${Math.ceil(body.retryAfterSeconds / 60)} min.` : '';
      setError(`${body.error ?? 'Something went wrong.'}${wait}`);
      return;
    }
    setRunId(body.id);
    window.history.replaceState(null, '', `/demo?run=${body.id}`);
  };

  if (runId && expired) {
    return (
      <div className="mt-10 rounded-lg border border-dashed border-ink/40 p-6">
        <p className="font-medium">This run has expired.</p>
        <p className="mt-1 text-sm text-ink-2">
          Portal run pages are kept for 24 hours. Every run&apos;s full log stays on{' '}
          <a className="underline underline-offset-4" href="https://github.com/georgitinchev/gantry/actions/workflows/demo.yml">GitHub Actions</a> for 90 days.
        </p>
      </div>
    );
  }
  if (runId) return <RunView run={run} />;

  if (!enabled) {
    return (
      <div className="mt-10 rounded-lg border border-dashed border-ink/40 p-6">
        <p className="font-medium">The live demo is switched off on this deployment.</p>
        <p className="mt-1 text-sm text-ink-2">
          The same pipeline runs on every pull request and every night — see the <a className="underline underline-offset-4" href="/status">latest verified run</a>.
        </p>
      </div>
    );
  }

  return (
    <div className="mt-10 grid grid-cols-1 gap-8 lg:grid-cols-[1fr_320px]">
      <div className="grid gap-6 rounded-lg border border-ink bg-card p-6">
        <Choice label="Environment size" value={req.size} options={['small', 'medium']} onChange={(size) => setReq({ ...req, size })} />
        <Choice label="Exposure" value={req.exposure} options={['internal', 'public']} onChange={(exposure) => setReq({ ...req, exposure })} />
        <Choice
          label="Also request a PostgreSQL database (CloudNativePG, inside the throwaway cluster)"
          value={req.postgres ? 'yes' : 'no'}
          options={['yes', 'no']}
          onChange={(v) => setReq({ ...req, postgres: v === 'yes' })}
        />
        {req.postgres && <Choice label="Postgres version" value={req.version} options={['16', '17', '18']} onChange={(version) => setReq({ ...req, version })} />}
      </div>
      <div className="grid content-start gap-3">
        {user ? (
          <>
            <div className="flex items-center gap-3 text-sm">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={user.avatar} alt="" className="size-8 rounded-full border border-line" />
              <span>
                Signed in as <strong>{user.login}</strong>
              </span>
            </div>
            <button
              type="button"
              disabled={starting}
              onClick={start}
              className="rounded-md bg-ink px-4 py-2.5 text-sm font-medium text-paper hover:bg-ink/85 disabled:opacity-60"
            >
              {starting ? 'Starting…' : 'Run it on a fresh cluster'}
            </button>
            <form action="/api/auth/logout" method="post">
              <button className="annot hover:!text-ink">sign out</button>
            </form>
          </>
        ) : (
          <a href="/api/auth/login" className="rounded-md bg-ink px-4 py-2.5 text-center text-sm font-medium text-paper hover:bg-ink/85">
            Sign in with GitHub to run
          </a>
        )}
        {error && <p className="text-sm text-drift">{error}</p>}
      </div>
    </div>
  );
}

function RunView({ run }: { run: { record: DemoRecord; events: DemoEvent[] } | null }) {
  if (!run) return <p className="mt-10 text-ink-2">Loading run…</p>;
  const last = run.events.at(-1);
  const reached = new Set(run.events.map((e) => e.phase));
  const failed = reached.has('failed');
  const done = reached.has('done');
  const resources = [...run.events].reverse().find((e) => e.resources)?.resources ?? [];
  const checks = [...run.events].reverse().find((e) => e.checks)?.checks ?? [];
  const runUrl = run.events.find((e) => e.runUrl)?.runUrl;

  return (
    <div className="mt-10 grid grid-cols-1 gap-8 lg:grid-cols-[300px_1fr]">
      <ol className="grid content-start gap-2">
        {PHASES.map((p) => {
          const isDone = reached.has(p.id) && (last?.phase !== p.id || done);
          const active = last?.phase === p.id && !done && !failed;
          return (
            <li key={p.id} data-state={isDone ? 'ready' : 'desired'} className="resource flex items-center justify-between rounded-md px-3 py-2 text-sm">
              <span>{p.label}</span>
              {active && <span className="annot animate-pulse">running</span>}
              {isDone && <span aria-label="done" className="size-2 rounded-full bg-signal" />}
            </li>
          );
        })}
      </ol>
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-3">
          <StatusChip ok={!failed}>{failed ? 'failed' : done ? 'verified' : 'in progress'}</StatusChip>
          <span className="annot">
            run {run.record.id} · by {run.record.user}
          </span>
          {runUrl && (
            <a className="annot underline underline-offset-4 hover:!text-ink" href={runUrl}>
              actions log
            </a>
          )}
        </div>
        <p className="mt-4 text-lg">{last?.message}</p>
        {resources.length > 0 && (
          <ul className="mt-6 grid grid-cols-1 gap-2 sm:grid-cols-2">
            {resources.map((r) => (
              <li key={`${r.kind}/${r.name}`} data-state={r.ready ? 'ready' : 'desired'} className="resource rounded-md px-3 py-2">
                <span className="block font-mono text-[0.625rem] uppercase tracking-wider text-ink-2">{r.kind}</span>
                <span className="block truncate text-sm">{r.name}</span>
              </li>
            ))}
          </ul>
        )}
        {checks.length > 0 && (
          <ul className="mt-6 divide-y divide-line rounded-lg border border-line">
            {checks.map((c) => (
              <li key={c.name} className="flex items-center justify-between gap-3 px-4 py-2.5 text-sm">
                <span>{c.name}</span>
                <StatusChip ok={c.status === 'pass'}>{c.status}</StatusChip>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

function Choice<T extends string>({ label, value, options, onChange }: { label: string; value: T; options: T[]; onChange: (v: T) => void }) {
  return (
    <fieldset>
      <legend className="mb-2 text-sm font-medium">{label}</legend>
      <div className="flex flex-wrap gap-2">
        {options.map((o) => (
          <button
            key={o}
            type="button"
            aria-pressed={value === o}
            onClick={() => onChange(o)}
            className={`rounded-md px-3 py-1.5 font-mono text-sm ${value === o ? 'border border-ink bg-ink text-paper' : 'border border-dashed border-ink/40 hover:border-solid'}`}
          >
            {o}
          </button>
        ))}
      </div>
    </fieldset>
  );
}
