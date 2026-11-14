'use client';

import { useMemo, useState } from 'react';
import type { Field, Offering } from '@/lib/catalog';
import { filePath, toYaml, validate, type Draft } from '@/lib/request';
import { site } from '@/lib/site';

const defaultNames: Record<string, string> = { PostgresInstance: 'orders', VirtualMachine: 'worker' };
const defaultName = (o: Offering, team: string) => (o.kind === 'AppEnvironment' ? `${team}-dev` : (defaultNames[o.kind] ?? 'main'));

function initialDraft(o: Offering, team: string): Draft {
  const values: Record<string, string> = {};
  for (const f of o.fields) if (f.default !== undefined) values[f.name] = String(f.default);
  if (!values.size && o.fields.find((f) => f.name === 'size')) values.size = 'small';
  return { team, name: defaultName(o, team), namespace: `${team}-dev`, values };
}

export function CatalogBrowser({ offerings }: { offerings: Offering[] }) {
  const [active, setActive] = useState(0);
  const [team, setTeam] = useState('checkout');
  const [drafts, setDrafts] = useState(() => offerings.map((o) => initialDraft(o, 'checkout')));
  const [copied, setCopied] = useState(false);

  const o = offerings[active];
  const draft = { ...drafts[active], team };
  const errors = useMemo(() => validate(o, draft), [o, draft]);
  const yaml = useMemo(() => toYaml(o, draft), [o, draft]);
  const path = filePath(o, draft);
  const valid = Object.keys(errors).length === 0;

  const update = (patch: Partial<Draft>) =>
    setDrafts((ds) => ds.map((d, i) => (i === active ? { ...d, ...patch } : d)));
  const setValue = (name: string, value: string) => update({ values: { ...draft.values, [name]: value } });

  const changeTeam = (t: string) => {
    // keep derived names in step with the team while they still look derived
    setDrafts((ds) =>
      ds.map((d, i) => ({
        ...d,
        name: offerings[i].kind === 'AppEnvironment' && d.name === `${team}-dev` ? `${t}-dev` : d.name,
        namespace: d.namespace === `${team}-dev` ? `${t}-dev` : d.namespace,
      })),
    );
    setTeam(t);
  };

  // GitHub's "new file" page: visitors without write access get a fork + PR flow for free.
  const prUrl = `https://github.com/${site.repo}/new/${site.branch}?filename=${encodeURIComponent(path)}&value=${encodeURIComponent(yaml)}`;

  const copy = async () => {
    await navigator.clipboard.writeText(yaml);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <div className="mt-10 grid grid-cols-1 gap-8 lg:grid-cols-[260px_1fr]">
      <nav aria-label="Offerings" className="flex min-w-0 gap-2 overflow-x-auto lg:flex-col">
        {offerings.map((off, i) => (
          <button
            key={off.kind}
            type="button"
            onClick={() => setActive(i)}
            aria-current={i === active}
            data-state={i === active ? 'ready' : 'desired'}
            className="resource min-w-[200px] rounded-lg p-4 text-left lg:min-w-0"
          >
            <span className="block font-mono text-[0.625rem] uppercase tracking-wider text-ink-2">
              {off.kind} · {off.namespaced ? 'namespaced' : 'cluster'}
            </span>
            <span className="mt-1 block font-medium">{off.title}</span>
            <span className="mt-1 block text-sm text-ink-2">{off.blurb}</span>
          </button>
        ))}
      </nav>

      <div className="grid min-w-0 grid-cols-1 gap-6 xl:grid-cols-2">
        <form className="grid min-w-0 grid-cols-1 content-start gap-5" onSubmit={(e) => e.preventDefault()} noValidate>
          <p className="text-sm text-ink-2">
            <span className="annot">Backed by</span>
            <br />
            {o.backedBy}
          </p>
          <Input label="Team" hint="The directory your request lives in" value={team} onChange={changeTeam} error={errors.team} />
          <Input
            label="Name"
            hint={o.kind === 'AppEnvironment' ? 'Becomes the namespace' : 'metadata.name'}
            value={draft.name}
            onChange={(v) => update({ name: v })}
            error={errors.name}
          />
          {o.namespaced && (
            <Input label="Environment" hint="metadata.namespace" value={draft.namespace} onChange={(v) => update({ namespace: v })} error={errors.namespace} />
          )}
          {o.fields.filter((f) => f.name !== 'team').map((f) => (
            <FieldInput key={f.name} field={f} value={draft.values[f.name] ?? ''} onChange={(v) => setValue(f.name, v)} error={errors[f.name]} />
          ))}
        </form>

        <div className="grid min-w-0 grid-cols-1 content-start gap-3">
          <div className="rounded-lg border border-ink bg-card shadow-[4px_4px_0_0_var(--ink)]">
            <div className="flex items-center justify-between gap-2 border-b border-line px-3 py-2">
              <span className="annot truncate">{path}</span>
              <button type="button" onClick={copy} className="annot shrink-0 hover:!text-ink">
                {copied ? 'copied' : 'copy'}
              </button>
            </div>
            <pre className="overflow-x-auto p-3 font-mono text-[0.75rem] leading-relaxed">{yaml}</pre>
          </div>
          <a
            href={valid ? prUrl : undefined}
            aria-disabled={!valid}
            target="_blank"
            rel="noreferrer"
            className={`inline-flex items-center justify-center gap-2 rounded-md px-4 py-2.5 text-sm font-medium ${
              valid ? 'bg-ink text-paper hover:bg-ink/85' : 'cursor-not-allowed border border-dashed border-ink/40 text-ink-2'
            }`}
          >
            {valid ? 'Open as pull request on GitHub' : 'Fix the highlighted fields'}
          </a>
          <p className="text-xs text-ink-2">
            GitHub opens a pre-filled file. Without write access it forks the repo and proposes the change — the same CI
            gate runs on your PR.
          </p>
        </div>
      </div>
    </div>
  );
}

function Input({ label, hint, value, onChange, error }: { label: string; hint?: string; value: string; onChange: (v: string) => void; error?: string }) {
  return (
    <label className="grid gap-1.5">
      <span className="flex items-baseline justify-between gap-2">
        <span className="text-sm font-medium">{label}</span>
        {hint && <span className="annot normal-case tracking-normal">{hint}</span>}
      </span>
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        aria-invalid={!!error}
        spellCheck={false}
        className="rounded-md border border-line bg-card px-3 py-2 font-mono text-sm outline-none focus:border-ink aria-[invalid=true]:border-drift"
      />
      {error && <span className="text-xs text-drift">{error}</span>}
    </label>
  );
}

function FieldInput({ field, value, onChange, error }: { field: Field; value: string; onChange: (v: string) => void; error?: string }) {
  const label = `${field.name}${field.required ? '' : ' (optional)'}`;
  if (field.type === 'enum') {
    return (
      <fieldset className="grid gap-1.5">
        <legend className="mb-1.5 text-sm font-medium">{label}</legend>
        <div className="flex flex-wrap gap-2">
          {field.options!.map((opt) => (
            <button
              key={opt}
              type="button"
              onClick={() => onChange(opt)}
              aria-pressed={value === opt}
              className={`rounded-md px-3 py-1.5 font-mono text-sm ${
                value === opt ? 'border border-ink bg-ink text-paper' : 'border border-dashed border-ink/40 hover:border-solid'
              }`}
            >
              {opt}
            </button>
          ))}
        </div>
        {field.description && <span className="text-xs text-ink-2">{field.description}</span>}
        {error && <span className="text-xs text-drift">{error}</span>}
      </fieldset>
    );
  }
  const hint = field.type === 'integer' ? `${field.min ?? ''}–${field.max ?? ''}` : field.type === 'list' ? 'comma separated' : undefined;
  return (
    <div className="grid gap-1">
      <Input label={label} hint={hint} value={value} onChange={onChange} error={error} />
      {field.description && <span className="text-xs text-ink-2">{field.description}</span>}
    </div>
  );
}
