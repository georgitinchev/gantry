import { timingSafeEqual } from 'node:crypto';
import { NextResponse, type NextRequest } from 'next/server';
import { demoConfig } from '@/lib/demo/config';
import { SESSION_COOKIE, SESSION_TTL, signSession } from '@/lib/demo/session';

export async function GET(req: NextRequest) {
  const cfg = demoConfig();
  const fail = (why: string) => NextResponse.redirect(new URL(`/demo?error=${why}`, req.url));
  if (!cfg.enabled) return fail('disabled');

  const code = req.nextUrl.searchParams.get('code');
  const state = req.nextUrl.searchParams.get('state') ?? '';
  const expected = req.cookies.get('gantry_oauth_state')?.value ?? '';
  if (!code || !expected || state.length !== expected.length || !timingSafeEqual(Buffer.from(state), Buffer.from(expected))) {
    return fail('state');
  }

  const tokenRes = await fetch('https://github.com/login/oauth/access_token', {
    method: 'POST',
    headers: { accept: 'application/json', 'content-type': 'application/json' },
    body: JSON.stringify({
      client_id: process.env.GITHUB_OAUTH_CLIENT_ID,
      client_secret: process.env.GITHUB_OAUTH_CLIENT_SECRET,
      code,
      redirect_uri: new URL('/api/auth/callback', req.nextUrl.origin).toString(),
    }),
  });
  const { access_token } = (await tokenRes.json()) as { access_token?: string };
  if (!access_token) return fail('token');

  const userRes = await fetch('https://api.github.com/user', {
    headers: { authorization: `Bearer ${access_token}`, accept: 'application/vnd.github+json' },
  });
  if (!userRes.ok) return fail('profile');
  const user = (await userRes.json()) as { id: number; login: string; avatar_url: string };
  // The GitHub token is discarded here; the session only carries id, login and avatar.

  const res = NextResponse.redirect(new URL('/demo', req.url));
  res.cookies.set(SESSION_COOKIE, await signSession({ id: user.id, login: user.login, avatar: user.avatar_url }, cfg.sessionSecret), {
    httpOnly: true,
    secure: true,
    sameSite: 'lax',
    path: '/',
    maxAge: SESSION_TTL,
  });
  res.cookies.delete({ name: 'gantry_oauth_state', path: '/api/auth' });
  return res;
}
