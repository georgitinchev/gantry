import { createHmac, timingSafeEqual } from 'node:crypto';

/**
 * Callbacks from the demo workflow are signed with a shared secret:
 *   X-Gantry-Timestamp: unix seconds
 *   X-Gantry-Signature: sha256=hex(HMAC(secret, `${timestamp}.${body}`))
 * The timestamp bounds replay to a five-minute window.
 */
export function sign(secret: string, timestamp: number, body: string) {
  return 'sha256=' + createHmac('sha256', secret).update(`${timestamp}.${body}`).digest('hex');
}

export function verify(secret: string, timestamp: string | null, signature: string | null, body: string, nowSeconds = Date.now() / 1000) {
  if (!timestamp || !signature || !/^\d{9,11}$/.test(timestamp)) return false;
  if (Math.abs(nowSeconds - Number(timestamp)) > 300) return false;
  const expected = Buffer.from(sign(secret, Number(timestamp), body));
  const given = Buffer.from(signature);
  return expected.length === given.length && timingSafeEqual(expected, given);
}
