import { describe, expect, it } from 'vitest';
import { admit, LIMITS, release } from '@/lib/demo/limits';
import { memoryStore } from '@/lib/demo/store';

describe('demo admission', () => {
  it('allows a first run and blocks a concurrent one', async () => {
    const store = memoryStore();
    expect(await admit(store, 1, 'a')).toEqual({ ok: true });
    const second = await admit(store, 2, 'b');
    expect(second).toMatchObject({ ok: false, reason: 'busy' });
  });

  it('does not charge the per-user slot when the platform is busy', async () => {
    const store = memoryStore();
    await admit(store, 1, 'a');
    await admit(store, 2, 'b'); // busy
    await release(store, 'a');
    expect(await admit(store, 2, 'c')).toEqual({ ok: true });
  });

  it('limits each user to one run per window, even after release', async () => {
    let now = 0;
    const store = memoryStore(() => now);
    await admit(store, 1, 'a', new Date(now));
    await release(store, 'a');
    expect(await admit(store, 1, 'b', new Date(now))).toMatchObject({ ok: false, reason: 'user' });
    now += LIMITS.perUserSeconds * 1000 + 1;
    expect(await admit(store, 1, 'c', new Date(now))).toEqual({ ok: true });
  });

  it('enforces the daily cap across users', async () => {
    const store = memoryStore();
    const day = new Date('2026-11-14T10:00:00Z');
    for (let u = 0; u < LIMITS.dailyRuns; u++) {
      expect(await admit(store, u, `r${u}`, day)).toEqual({ ok: true });
      await release(store, `r${u}`);
    }
    const over = await admit(store, 999, 'over', day);
    expect(over).toMatchObject({ ok: false, reason: 'daily' });
    // the rejected user keeps their slot for later
    expect(await store.get('demo:user:999')).toBeNull();
  });

  it('only the holder can release the lock', async () => {
    const store = memoryStore();
    await admit(store, 1, 'a');
    await release(store, 'someone-else');
    expect(await admit(store, 2, 'b')).toMatchObject({ ok: false, reason: 'busy' });
  });

  it('frees a stuck lock after the max run time', async () => {
    let now = 0;
    const store = memoryStore(() => now);
    await admit(store, 1, 'a', new Date(now));
    now += LIMITS.maxRunSeconds * 1000 + 1;
    expect(await admit(store, 2, 'b', new Date(now))).toEqual({ ok: true });
  });
});
