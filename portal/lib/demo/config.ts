import 'server-only';
import { upstashStore } from './store';

/** The live demo is optional: every piece must be configured or it stays off. */
export function demoConfig() {
  const env = process.env;
  const store = upstashStore();
  const ready =
    !!store &&
    !!env.GITHUB_OAUTH_CLIENT_ID &&
    !!env.GITHUB_OAUTH_CLIENT_SECRET &&
    !!env.SESSION_SECRET &&
    env.SESSION_SECRET.length >= 32 &&
    !!env.DEMO_CALLBACK_SECRET &&
    !!env.GITHUB_APP_ID &&
    !!env.GITHUB_APP_INSTALLATION_ID &&
    !!env.GITHUB_APP_PRIVATE_KEY;
  return ready
    ? { enabled: true as const, store: store!, sessionSecret: env.SESSION_SECRET!, callbackSecret: env.DEMO_CALLBACK_SECRET! }
    : { enabled: false as const };
}

/**
 * The one origin registered as the OAuth callback. Sign-ins that start on any
 * other host (preview URLs, the auto-generated *.vercel.app alias) are bounced
 * here first so the redirect_uri always matches and cookies land on one host.
 */
export function canonicalOrigin(requestOrigin: string): string {
  const configured = process.env.PORTAL_URL?.replace(/\/+$/, '');
  return configured && /^https:\/\/[a-z0-9.-]+$/.test(configured) ? configured : requestOrigin;
}
