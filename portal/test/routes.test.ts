import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NextRequest } from 'next/server';
import { memoryStore, type Store } from '@/lib/demo/store';
import { signSession } from '@/lib/demo/session';
import { sign } from '@/lib/demo/signature';

const secret = 's'.repeat(32);
const callbackSecret = 'c'.repeat(32);
let store: Store;
const dispatched: string[] = [];

vi.mock('@/lib/demo/config', () => ({
  demoConfig: () => ({ enabled: true, store, sessionSecret: secret, callbackSecret }),
}));
vi.mock('@/lib/demo/github', () => ({
  dispatchDemo: async (id: string) => {
    dispatched.push(id);
  },
}));

const { POST: startDemo } = await import('@/app/api/demo/route');
const { GET: getDemo } = await import('@/app/api/demo/[id]/route');
const { POST: postEvent } = await import('@/app/api/demo/[id]/events/route');

const origin = 'https://gantry.example';
const body = { size: 'small', exposure: 'internal', postgres: true, version: '17' };

async function start(userId: number, init: { origin?: string | null; cookie?: string | null; body?: unknown } = {}) {
  const headers = new Headers({ 'content-type': 'application/json' });
  if (init.origin !== null) headers.set('origin', init.origin ?? origin);
  const cookie = init.cookie === undefined ? await signSession({ id: userId, login: `u${userId}`, avatar: '' }, secret) : init.cookie;
  if (cookie) headers.set('cookie', `gantry_session=${cookie}`);
  return startDemo(new NextRequest(`${origin}/api/demo`, { method: 'POST', headers, body: JSON.stringify(init.body ?? body) }));
}

function event(id: string, payload: object, opts: { secret?: string; ts?: number } = {}) {
  const raw = JSON.stringify(payload);
  const ts = opts.ts ?? Math.floor(Date.now() / 1000);
  return postEvent(
    new Request(`${origin}/api/demo/${id}/events`, {
      method: 'POST',
      headers: { 'x-gantry-timestamp': String(ts), 'x-gantry-signature': sign(opts.secret ?? callbackSecret, ts, raw) },
      body: raw,
    }),
    { params: Promise.resolve({ id }) },
  );
}

describe('demo API', () => {
  beforeEach(() => {
    store = memoryStore();
    dispatched.length = 0;
  });

  it('starts a run for a signed-in user and dispatches the workflow', async () => {
    const res = await start(1);
    expect(res.status).toBe(202);
    const { id } = await res.json();
    expect(id).toMatch(/^[a-f0-9]{16}$/);
    expect(dispatched).toEqual([id]);
  });

  it('requires a session, same origin and a valid body', async () => {
    expect((await start(1, { cookie: null })).status).toBe(401);
    expect((await start(1, { cookie: 'forged' })).status).toBe(401);
    expect((await start(1, { origin: 'https://evil.example' })).status).toBe(403);
    expect((await start(1, { origin: null })).status).toBe(403);
    expect((await start(1, { body: { ...body, size: 'large' } })).status).toBe(400);
    expect(dispatched).toHaveLength(0);
  });

  it('rate limits concurrent runs and repeat users', async () => {
    const first = await (await start(1)).json();
    const busy = await start(2);
    expect(busy.status).toBe(429);
    expect(busy.headers.get('retry-after')).toBeTruthy();

    // workflow finishes -> lock released, but user 1 still waits 4h
    expect((await event(first.id, { phase: 'done', message: 'ok' })).status).toBe(200);
    expect((await start(1)).status).toBe(429);
    expect((await start(2)).status).toBe(202);
  });

  it('accepts only correctly signed, well-formed events', async () => {
    const { id } = await (await start(1)).json();
    expect((await event(id, { phase: 'apply', message: 'x' }, { secret: 'wrong'.repeat(8) })).status).toBe(401);
    expect((await event(id, { phase: 'apply', message: 'x' }, { ts: 1_000_000_000 })).status).toBe(401);
    expect((await event(id, { phase: 'apply', message: 'x', extra: 1 })).status).toBe(400);
    expect((await event('ffffffffffffffff', { phase: 'apply', message: 'x' })).status).toBe(404);
    expect((await event(id, { phase: 'reconcile', message: '3/8 ready', resources: [{ kind: 'Namespace', name: 'demo', ready: true }] })).status).toBe(200);

    const res = await getDemo(new Request(`${origin}/api/demo/${id}`), { params: Promise.resolve({ id }) });
    const run = await res.json();
    expect(run.record.user).toBe('u1');
    expect(run.events.map((e: { phase: string }) => e.phase)).toEqual(['queued', 'reconcile']);
  });

  it('does not leak runs for malformed ids', async () => {
    const res = await getDemo(new Request(`${origin}/api/demo/../../etc`), { params: Promise.resolve({ id: '../../etc' }) });
    expect(res.status).toBe(404);
  });
});
