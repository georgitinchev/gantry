import { NextResponse } from 'next/server';
import { demoConfig } from '@/lib/demo/config';
import { release } from '@/lib/demo/limits';
import { DEMO_ID, DemoEvent, eventsKey, RECORD_TTL, recordKey } from '@/lib/demo/request';
import { verify } from '@/lib/demo/signature';

/** Progress callbacks from the demo workflow, HMAC-signed with DEMO_CALLBACK_SECRET. */
export async function POST(req: Request, { params }: { params: Promise<{ id: string }> }) {
  const cfg = demoConfig();
  const { id } = await params;
  if (!cfg.enabled || !DEMO_ID.test(id)) return NextResponse.json({ error: 'not found' }, { status: 404 });

  const body = await req.text();
  if (body.length > 16_384) return NextResponse.json({ error: 'too large' }, { status: 413 });
  if (!verify(cfg.callbackSecret, req.headers.get('x-gantry-timestamp'), req.headers.get('x-gantry-signature'), body)) {
    return NextResponse.json({ error: 'bad signature' }, { status: 401 });
  }
  if (!(await cfg.store.get(recordKey(id)))) return NextResponse.json({ error: 'not found' }, { status: 404 });

  let json: unknown;
  try {
    json = JSON.parse(body);
  } catch {
    return NextResponse.json({ error: 'invalid event' }, { status: 400 });
  }
  const event = DemoEvent.safeParse(json);
  if (!event.success) return NextResponse.json({ error: 'invalid event' }, { status: 400 });

  await cfg.store.push(eventsKey(id), { ...event.data, at: new Date().toISOString() }, RECORD_TTL);
  if (event.data.phase === 'done' || event.data.phase === 'failed') await release(cfg.store, id);
  return NextResponse.json({ ok: true });
}
