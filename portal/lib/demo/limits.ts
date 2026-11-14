import type { Store } from './store';

export const LIMITS = {
  perUserSeconds: 4 * 60 * 60, // one run per GitHub user every 4 hours
  maxRunSeconds: 25 * 60, // global lock; a run that never reports back frees itself
  dailyRuns: 15, // across all users
};

export type Admission =
  | { ok: true }
  | { ok: false; reason: 'busy' | 'daily' | 'user'; retryAfterSeconds: number };

const keys = {
  active: 'demo:active',
  user: (id: number) => `demo:user:${id}`,
  day: (d: Date) => `demo:day:${d.toISOString().slice(0, 10)}`,
};

/**
 * Claims a demo slot for this user. Order matters: cheap read-only checks first,
 * then the per-user key, then the global lock, rolling back if the lock is lost.
 */
export async function admit(store: Store, userId: number, demoId: string, now = new Date()): Promise<Admission> {
  if (await store.get(keys.active)) {
    return { ok: false, reason: 'busy', retryAfterSeconds: Math.max(await store.ttl(keys.active), 30) };
  }
  if (!(await store.set(keys.user(userId), demoId, { ex: LIMITS.perUserSeconds, nx: true }))) {
    return { ok: false, reason: 'user', retryAfterSeconds: await store.ttl(keys.user(userId)) };
  }
  const today = await store.incr(keys.day(now), 2 * 24 * 60 * 60);
  if (today > LIMITS.dailyRuns) {
    await store.del(keys.user(userId));
    const midnight = new Date(now);
    midnight.setUTCHours(24, 0, 0, 0);
    return { ok: false, reason: 'daily', retryAfterSeconds: Math.ceil((midnight.getTime() - now.getTime()) / 1000) };
  }
  if (!(await store.set(keys.active, demoId, { ex: LIMITS.maxRunSeconds, nx: true }))) {
    await store.del(keys.user(userId));
    return { ok: false, reason: 'busy', retryAfterSeconds: Math.max(await store.ttl(keys.active), 30) };
  }
  return { ok: true };
}

/** Frees the global lock, but only if this demo still holds it. */
export async function release(store: Store, demoId: string) {
  if ((await store.get<string>(keys.active)) === demoId) await store.del(keys.active);
}
