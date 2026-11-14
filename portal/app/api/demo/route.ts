import { randomBytes } from 'node:crypto';
import { NextResponse, type NextRequest } from 'next/server';
import { demoConfig } from '@/lib/demo/config';
import { dispatchDemo } from '@/lib/demo/github';
import { admit, release } from '@/lib/demo/limits';
import { DemoRequest, eventsKey, RECORD_TTL, recordKey, type DemoRecord } from '@/lib/demo/request';
import { SESSION_COOKIE, verifySession } from '@/lib/demo/session';

export async function POST(req: NextRequest) {
  const cfg = demoConfig();
  if (!cfg.enabled) return NextResponse.json({ error: 'The live demo is not configured.' }, { status: 503 });

  // Same-origin only; the session cookie is SameSite=Lax as a second line.
  const origin = req.headers.get('origin');
  if (!origin || new URL(origin).host !== req.nextUrl.host) {
    return NextResponse.json({ error: 'Cross-origin request rejected.' }, { status: 403 });
  }

  const session = await verifySession(req.cookies.get(SESSION_COOKIE)?.value, cfg.sessionSecret);
  if (!session) return NextResponse.json({ error: 'Sign in with GitHub first.' }, { status: 401 });

  const parsed = DemoRequest.safeParse(await req.json().catch(() => null));
  if (!parsed.success) return NextResponse.json({ error: 'Invalid request.' }, { status: 400 });

  const id = randomBytes(8).toString('hex');
  const admission = await admit(cfg.store, session.id, id);
  if (!admission.ok) {
    const messages = {
      busy: 'Another demo is running right now.',
      user: 'You can start one demo every 4 hours.',
      daily: 'Today’s demo budget is used up.',
    };
    return NextResponse.json(
      { error: messages[admission.reason], retryAfterSeconds: admission.retryAfterSeconds },
      { status: 429, headers: { 'retry-after': String(admission.retryAfterSeconds) } },
    );
  }

  const record: DemoRecord = { id, user: session.login, request: parsed.data, createdAt: new Date().toISOString() };
  await cfg.store.set(recordKey(id), record, { ex: RECORD_TTL });
  await cfg.store.push(eventsKey(id), { phase: 'queued', message: 'Waiting for a GitHub-hosted runner', at: record.createdAt }, RECORD_TTL);

  try {
    await dispatchDemo(id, parsed.data);
  } catch (err) {
    console.error(err);
    await release(cfg.store, id);
    await cfg.store.del(`demo:user:${session.id}`);
    return NextResponse.json({ error: 'Could not start the workflow. Try again later.' }, { status: 502 });
  }
  return NextResponse.json({ id }, { status: 202 });
}
