import 'server-only';
import sample from '@/data/snapshot.sample.json';

export type Check = { name: string; status: 'pass' | 'fail' };
export type Resource = {
  kind: string;
  name: string;
  namespace: string | null;
  ready: boolean;
  synced: boolean;
  created: string;
  spec: Record<string, unknown>;
  status: Record<string, unknown> | null;
  composed: { kind: string; name: string }[];
};
export type Snapshot = {
  generatedAt: string;
  commit: string;
  runUrl: string | null;
  bootstrapSeconds: number;
  versions: Record<string, string>;
  checks: Check[];
  resources: Resource[];
  source: 'live' | 'sample';
};

// The nightly workflow publishes snapshot.json to the orphan `status` branch.
const STATUS_URL =
  process.env.STATUS_URL ?? 'https://raw.githubusercontent.com/georgitinchev/gantry/status/snapshot.json';

export async function getSnapshot(): Promise<Snapshot> {
  try {
    const res = await fetch(STATUS_URL, { next: { revalidate: 300 } });
    if (res.ok) return { ...((await res.json()) as Omit<Snapshot, 'source'>), source: 'live' };
  } catch {
    // fall through to the bundled sample
  }
  return { ...(sample as Omit<Snapshot, 'source'>), source: 'sample' };
}

export function overall(s: Snapshot) {
  const failed = s.checks.filter((c) => c.status !== 'pass').length;
  const notReady = s.resources.filter((r) => !r.ready).length;
  return { ok: failed === 0 && notReady === 0 && s.checks.length > 0, failed, notReady };
}
