import { NextResponse, type NextRequest } from 'next/server';
import { SESSION_COOKIE } from '@/lib/demo/session';

export async function POST(req: NextRequest) {
  const res = NextResponse.redirect(new URL('/demo', req.url), 303);
  res.cookies.delete(SESSION_COOKIE);
  return res;
}
