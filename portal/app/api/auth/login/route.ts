import { randomBytes } from 'node:crypto';
import { NextResponse, type NextRequest } from 'next/server';
import { canonicalOrigin, demoConfig } from '@/lib/demo/config';

export async function GET(req: NextRequest) {
  if (!demoConfig().enabled) return NextResponse.redirect(new URL('/demo', req.url));
  const origin = canonicalOrigin(req.nextUrl.origin);
  if (origin !== req.nextUrl.origin) return NextResponse.redirect(new URL('/api/auth/login', origin));
  const state = randomBytes(16).toString('hex');
  const url = new URL('https://github.com/login/oauth/authorize');
  url.searchParams.set('client_id', process.env.GITHUB_OAUTH_CLIENT_ID!);
  url.searchParams.set('redirect_uri', new URL('/api/auth/callback', origin).toString());
  url.searchParams.set('state', state);
  url.searchParams.set('allow_signup', 'false');
  // No scopes: we only read the public profile to rate-limit per account.
  const res = NextResponse.redirect(url);
  res.cookies.set('gantry_oauth_state', state, { httpOnly: true, secure: true, sameSite: 'lax', path: '/api/auth', maxAge: 600 });
  return res;
}
