import { describe, expect, it } from 'vitest';
import { DemoEvent, DemoRequest } from '@/lib/demo/request';
import { signSession, verifySession } from '@/lib/demo/session';
import { sign, verify } from '@/lib/demo/signature';

const secret = 'x'.repeat(32);

describe('callback signatures', () => {
  const now = 1_800_000_000;
  const body = '{"phase":"done","message":"ok"}';

  it('accepts a valid signature', () => {
    expect(verify(secret, String(now), sign(secret, now, body), body, now)).toBe(true);
  });
  it('rejects a modified body', () => {
    expect(verify(secret, String(now), sign(secret, now, body), body + ' ', now)).toBe(false);
  });
  it('rejects the wrong secret', () => {
    expect(verify(secret, String(now), sign('y'.repeat(32), now, body), body, now)).toBe(false);
  });
  it('rejects replays outside five minutes', () => {
    expect(verify(secret, String(now), sign(secret, now, body), body, now + 301)).toBe(false);
  });
  it('rejects missing or malformed headers', () => {
    expect(verify(secret, null, 'sha256=00', body, now)).toBe(false);
    expect(verify(secret, 'abc', sign(secret, now, body), body, now)).toBe(false);
  });
});

describe('sessions', () => {
  it('round-trips', async () => {
    const token = await signSession({ id: 42, login: 'octo', avatar: 'a' }, secret);
    expect(await verifySession(token, secret)).toEqual({ id: 42, login: 'octo', avatar: 'a' });
  });
  it('rejects a token signed with another secret', async () => {
    const token = await signSession({ id: 42, login: 'octo', avatar: 'a' }, 'z'.repeat(32));
    expect(await verifySession(token, secret)).toBeNull();
  });
  it('rejects garbage', async () => {
    expect(await verifySession('not.a.jwt', secret)).toBeNull();
    expect(await verifySession(undefined, secret)).toBeNull();
  });
});

describe('demo input', () => {
  it('accepts the catalog choices', () => {
    expect(DemoRequest.safeParse({ size: 'small', exposure: 'public', postgres: true, version: '17' }).success).toBe(true);
  });
  it('rejects anything outside the choices or extra keys', () => {
    expect(DemoRequest.safeParse({ size: 'large', exposure: 'public', postgres: true, version: '17' }).success).toBe(false);
    expect(DemoRequest.safeParse({ size: 'small', exposure: 'public', postgres: true, version: '17', image: 'x' }).success).toBe(false);
    expect(DemoRequest.safeParse({ size: 'small; rm -rf /', exposure: 'public', postgres: true, version: '17' }).success).toBe(false);
  });
  it('bounds callback events', () => {
    expect(DemoEvent.safeParse({ phase: 'apply', message: 'x'.repeat(301) }).success).toBe(false);
    expect(DemoEvent.safeParse({ phase: 'pwn', message: 'x' }).success).toBe(false);
  });
});
