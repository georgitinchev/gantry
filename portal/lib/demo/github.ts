import 'server-only';
import { importPKCS8, SignJWT } from 'jose';
import { createPrivateKey } from 'node:crypto';
import { site } from '@/lib/site';
import type { DemoRequest } from './request';

/**
 * Dispatches the demo workflow as a GitHub App installation, whose only
 * repository permission is `actions: write` on this repository.
 */
export async function dispatchDemo(demoId: string, req: DemoRequest): Promise<void> {
  const token = await installationToken();
  const res = await fetch(`https://api.github.com/repos/${site.repo}/actions/workflows/demo.yml/dispatches`, {
    method: 'POST',
    headers: { authorization: `Bearer ${token}`, accept: 'application/vnd.github+json', 'x-github-api-version': '2022-11-28' },
    body: JSON.stringify({
      ref: site.branch,
      inputs: { demo_id: demoId, size: req.size, exposure: req.exposure, postgres: String(req.postgres), version: req.version },
    }),
  });
  if (!res.ok) throw new Error(`workflow dispatch failed: ${res.status} ${await res.text()}`);
}

async function installationToken(): Promise<string> {
  const appId = process.env.GITHUB_APP_ID!;
  const installationId = process.env.GITHUB_APP_INSTALLATION_ID!;
  // GitHub issues PKCS#1 keys; jose wants PKCS#8.
  const pem = process.env.GITHUB_APP_PRIVATE_KEY!.replace(/\\n/g, '\n');
  const pkcs8 = createPrivateKey(pem).export({ type: 'pkcs8', format: 'pem' }).toString();
  const now = Math.floor(Date.now() / 1000);
  const jwt = await new SignJWT({})
    .setProtectedHeader({ alg: 'RS256' })
    .setIssuer(appId)
    .setIssuedAt(now - 60)
    .setExpirationTime(now + 9 * 60)
    .sign(await importPKCS8(pkcs8, 'RS256'));
  const res = await fetch(`https://api.github.com/app/installations/${installationId}/access_tokens`, {
    method: 'POST',
    headers: { authorization: `Bearer ${jwt}`, accept: 'application/vnd.github+json' },
    body: JSON.stringify({ repositories: [site.repo.split('/')[1]], permissions: { actions: 'write' } }),
  });
  if (!res.ok) throw new Error(`installation token failed: ${res.status}`);
  return ((await res.json()) as { token: string }).token;
}
