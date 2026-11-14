---
title: Deploy the portal
description: Vercel project, the optional live demo, and the secrets it needs.
---

The portal works with **no configuration**: catalog, docs and status render
without any secrets. The live demo switches itself on only when every value
below is set.

## 1. Vercel project

1. Import the repository in Vercel and set **Root Directory** to `portal`.
2. Framework preset: Next.js. No build overrides are needed.
3. Hobby plan is enough; the site is non-commercial.

## 2. Status snapshots

Nothing to configure. The nightly workflow pushes `snapshot.json` to the
`status` branch and the portal reads it from `raw.githubusercontent.com`,
revalidating every five minutes. Override with `STATUS_URL` if needed.

## 3. Live demo (optional)

| Variable | Where it comes from |
|---|---|
| `GITHUB_OAUTH_CLIENT_ID`, `GITHUB_OAUTH_CLIENT_SECRET` | GitHub → Settings → Developer settings → **OAuth Apps**. Callback URL: `https://<your-domain>/api/auth/callback` |
| `GITHUB_APP_ID`, `GITHUB_APP_INSTALLATION_ID`, `GITHUB_APP_PRIVATE_KEY` | A **GitHub App** with repository permission *Actions: Read and write* only, installed on this repository only |
| `UPSTASH_REDIS_REST_URL`, `UPSTASH_REDIS_REST_TOKEN` (or `KV_REST_API_URL`/`KV_REST_API_TOKEN`) | Vercel → Storage → **Upstash for Redis** (free tier) |
| `SESSION_SECRET` | `openssl rand -hex 32` |
| `DEMO_CALLBACK_SECRET` | `openssl rand -hex 32`, also stored as a repository secret |

On the repository:

```bash
gh secret set DEMO_CALLBACK_SECRET
gh variable set PORTAL_URL --body "https://<your-domain>"
```

## 4. Check it

1. Open `/demo`, sign in, start a run.
2. Watch **Actions → live-demo** and the portal update in step.
3. Start a second run straight away: it must be refused with a retry time.
