import { stringify } from 'yaml';
import type { Field, Offering } from './catalog';

export type Draft = {
  team: string;
  name: string;
  namespace: string;
  values: Record<string, string>;
};

const TEAM = /^[a-z][a-z0-9-]{1,28}[a-z0-9]$/;
const DNS = /^[a-z0-9]([-a-z0-9]{0,61}[a-z0-9])?$/;

/** Mirrors the server-side checks so the form fails fast; gantry-lint stays authoritative. */
export function validate(o: Offering, d: Draft): Record<string, string> {
  const errors: Record<string, string> = {};
  if (!TEAM.test(d.team)) errors.team = 'lowercase letters, digits and dashes, 3–30 chars';
  if (!DNS.test(d.name)) errors.name = 'must be a DNS label (lowercase, digits, dashes)';
  else if (o.kind === 'AppEnvironment' && !d.name.startsWith(`${d.team}-`)) errors.name = `must start with "${d.team}-"`;
  if (o.namespaced && !d.namespace.startsWith(`${d.team}-`)) errors.namespace = `must be one of ${d.team}'s environments (${d.team}-…)`;
  for (const f of o.fields) {
    if (f.name === 'team') continue;
    const v = d.values[f.name] ?? '';
    if (!v) {
      if (f.required) errors[f.name] = 'required';
      continue;
    }
    if (f.type === 'integer') {
      const n = Number(v);
      if (!Number.isInteger(n)) errors[f.name] = 'must be a whole number';
      else if (f.min !== undefined && n < f.min) errors[f.name] = `minimum ${f.min}`;
      else if (f.max !== undefined && n > f.max) errors[f.name] = `maximum ${f.max}`;
    }
    if (f.type === 'string' && f.pattern && !new RegExp(f.pattern).test(v)) errors[f.name] = `must match ${f.pattern}`;
  }
  return errors;
}

export function toObject(o: Offering, d: Draft) {
  const spec: Record<string, unknown> = {};
  for (const f of o.fields) {
    if (f.name === 'team') {
      spec.team = d.team;
      continue;
    }
    const v = d.values[f.name];
    if (!v) continue;
    spec[f.name] = coerce(f, v);
  }
  return {
    apiVersion: o.apiVersion,
    kind: o.kind,
    metadata: o.namespaced ? { name: d.name, namespace: d.namespace } : { name: d.name },
    spec,
  };
}

function coerce(f: Field, v: string) {
  if (f.type === 'integer') return Number(v);
  if (f.type === 'list') return v.split(',').map((s) => s.trim()).filter(Boolean);
  return v;
}

export function toYaml(o: Offering, d: Draft) {
  return stringify(toObject(o, d), { lineWidth: 0 });
}

export function filePath(o: Offering, d: Draft) {
  const stem = o.namespaced ? `${o.filePrefix}-${d.name}` : o.filePrefix;
  return `requests/${d.team}/${stem}.yaml`;
}
