import 'server-only';
import { Redis } from '@upstash/redis';

/** The handful of Redis operations the demo needs; swappable for tests. */
export interface Store {
  get<T>(key: string): Promise<T | null>;
  set(key: string, value: unknown, opts: { ex: number; nx?: boolean }): Promise<boolean>;
  del(key: string): Promise<void>;
  incr(key: string, ttlSeconds: number): Promise<number>;
  ttl(key: string): Promise<number>;
  push(key: string, value: unknown, ttlSeconds: number): Promise<void>;
  list<T>(key: string): Promise<T[]>;
}

export function upstashStore(): Store | null {
  const url = process.env.UPSTASH_REDIS_REST_URL ?? process.env.KV_REST_API_URL;
  const token = process.env.UPSTASH_REDIS_REST_TOKEN ?? process.env.KV_REST_API_TOKEN;
  if (!url || !token) return null;
  const r = new Redis({ url, token });
  return {
    get: (k) => r.get(k),
    async set(k, v, { ex, nx }) {
      const res = nx ? await r.set(k, v, { ex, nx: true }) : await r.set(k, v, { ex });
      return res === 'OK';
    },
    async del(k) {
      await r.del(k);
    },
    async incr(k, ttl) {
      const n = await r.incr(k);
      if (n === 1) await r.expire(k, ttl);
      return n;
    },
    ttl: (k) => r.ttl(k),
    async push(k, v, ttl) {
      await r.rpush(k, v);
      await r.expire(k, ttl);
    },
    list: (k) => r.lrange(k, 0, -1),
  };
}

/** In-memory store for tests and local development. */
export function memoryStore(now = () => Date.now()): Store {
  const data = new Map<string, { v: unknown; exp: number }>();
  const live = (k: string) => {
    const e = data.get(k);
    if (e && e.exp <= now()) data.delete(k);
    return data.get(k);
  };
  return {
    async get<T>(k: string) {
      return (live(k)?.v as T) ?? null;
    },
    async set(k, v, { ex, nx }) {
      if (nx && live(k)) return false;
      data.set(k, { v, exp: now() + ex * 1000 });
      return true;
    },
    async del(k) {
      data.delete(k);
    },
    async incr(k, ttl) {
      const e = live(k);
      const n = ((e?.v as number) ?? 0) + 1;
      data.set(k, { v: n, exp: e?.exp ?? now() + ttl * 1000 });
      return n;
    },
    async ttl(k) {
      const e = live(k);
      return e ? Math.ceil((e.exp - now()) / 1000) : -2;
    },
    async push(k, v, ttl) {
      const e = live(k);
      data.set(k, { v: [...((e?.v as unknown[]) ?? []), v], exp: now() + ttl * 1000 });
    },
    async list<T>(k: string) {
      return ((live(k)?.v as T[]) ?? []) as T[];
    },
  };
}
