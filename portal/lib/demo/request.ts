import { z } from 'zod';

/** Everything a visitor can choose. The workflow re-validates with `choice` inputs. */
export const DemoRequest = z
  .object({
    size: z.enum(['small', 'medium']),
    exposure: z.enum(['internal', 'public']),
    postgres: z.boolean(),
    version: z.enum(['16', '17', '18']),
  })
  .strict();
export type DemoRequest = z.infer<typeof DemoRequest>;

export const DemoEvent = z
  .object({
    phase: z.enum(['queued', 'cluster', 'platform', 'lint', 'apply', 'reconcile', 'verify', 'done', 'failed']),
    message: z.string().max(300),
    resources: z
      .array(z.object({ kind: z.string().max(60), name: z.string().max(80), ready: z.boolean() }))
      .max(30)
      .optional(),
    checks: z.array(z.object({ name: z.string().max(120), status: z.enum(['pass', 'fail']) })).max(30).optional(),
    runUrl: z.url().max(300).optional(),
  })
  .strict();
export type DemoEvent = z.infer<typeof DemoEvent> & { at: string };

export type DemoRecord = {
  id: string;
  user: string;
  request: DemoRequest;
  createdAt: string;
};

export const DEMO_ID = /^[a-f0-9]{16}$/;
export const RECORD_TTL = 24 * 60 * 60;
export const recordKey = (id: string) => `demo:run:${id}`;
export const eventsKey = (id: string) => `demo:events:${id}`;
