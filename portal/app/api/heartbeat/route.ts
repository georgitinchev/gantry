import { NextResponse } from 'next/server';
import { demoConfig } from '@/lib/demo/config';
import { verify } from '@/lib/demo/signature';

/**
 * Nightly keepalive. Upstash archives free databases after 14 idle days, so the
 * nightly workflow writes one key here. Signed with DEMO_CALLBACK_SECRET so
 * nobody else can spend the command quota.
 */
export async function POST(req: Request) {
  const cfg = demoConfig();
  if (!cfg.enabled) return NextResponse.json({ error: 'demo not configured' }, { status: 503 });
  const body = await req.text();
  if (!verify(cfg.callbackSecret, req.headers.get('x-gantry-timestamp'), req.headers.get('x-gantry-signature'), body)) {
    return NextResponse.json({ error: 'bad signature' }, { status: 401 });
  }
  const at = new Date().toISOString();
  await cfg.store.set('demo:heartbeat', at, { ex: 3 * 24 * 60 * 60 });
  return NextResponse.json({ ok: true, at });
}
