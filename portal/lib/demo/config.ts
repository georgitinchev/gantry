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
