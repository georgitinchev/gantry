import { NextResponse } from 'next/server';
import { demoConfig } from '@/lib/demo/config';
import { DEMO_ID, eventsKey, recordKey, type DemoEvent, type DemoRecord } from '@/lib/demo/request';

export async function GET(_req: Request, { params }: { params: Promise<{ id: string }> }) {
  const cfg = demoConfig();
  const { id } = await params;
  if (!cfg.enabled || !DEMO_ID.test(id)) return NextResponse.json({ error: 'not found' }, { status: 404 });
  const record = await cfg.store.get<DemoRecord>(recordKey(id));
  if (!record) return NextResponse.json({ error: 'not found' }, { status: 404 });
  const events = await cfg.store.list<DemoEvent>(eventsKey(id));
  return NextResponse.json({ record, events }, { headers: { 'cache-control': 'no-store' } });
}
