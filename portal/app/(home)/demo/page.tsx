import type { Metadata } from 'next';
import { cookies } from 'next/headers';
import { DemoRunner } from '@/components/demo-runner';
import { demoConfig } from '@/lib/demo/config';
import { LIMITS } from '@/lib/demo/limits';
import { SESSION_COOKIE, verifySession } from '@/lib/demo/session';

export const dynamic = 'force-dynamic';
export const metadata: Metadata = {
  title: 'Live demo',
  description: 'Watch a real request reconcile on a fresh Kubernetes cluster.',
};

const errors: Record<string, string> = {
  state: 'Sign-in expired or was tampered with. Please try again.',
  token: 'GitHub did not complete the sign-in.',
  profile: 'Could not read your GitHub profile.',
  disabled: 'The live demo is not configured on this deployment.',
};

export default async function DemoPage({ searchParams }: { searchParams: Promise<{ error?: string }> }) {
  const cfg = demoConfig();
  const { error } = await searchParams;
  const session = cfg.enabled ? await verifySession((await cookies()).get(SESSION_COOKIE)?.value, cfg.sessionSecret) : null;

  return (
    <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-12 sm:px-6">
      <p className="annot">Live demo</p>
      <h1 className="mt-2 max-w-3xl text-3xl font-semibold tracking-tight sm:text-4xl">
        Build the platform from nothing, then reconcile your request on it.
      </h1>
      <p className="mt-3 max-w-2xl text-ink-2">
        Pressing run starts a GitHub Actions job that creates a kind cluster, installs Crossplane, CloudNativePG and Kyverno,
        lints your request, applies it and verifies the result. It takes about five minutes and you can watch every step.
      </p>
      <ul className="mt-4 flex flex-wrap gap-x-6 gap-y-1 text-sm text-ink-2">
        <li>· one run per GitHub account every {LIMITS.perUserSeconds / 3600} h</li>
        <li>· one run at a time</li>
        <li>· {LIMITS.dailyRuns} runs per day in total</li>
        <li>· sign-in reads your public profile only</li>
      </ul>
      {error && errors[error] && <p className="mt-6 rounded-md border border-drift px-4 py-3 text-sm">{errors[error]}</p>}
      <DemoRunner enabled={cfg.enabled} user={session ? { login: session.login, avatar: session.avatar } : null} />
    </main>
  );
}
