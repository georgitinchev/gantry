import { jwtVerify, SignJWT } from 'jose';

export type Session = { id: number; login: string; avatar: string };

export const SESSION_COOKIE = 'gantry_session';
export const SESSION_TTL = 8 * 60 * 60;

const key = (secret: string) => new TextEncoder().encode(secret);

export async function signSession(s: Session, secret: string): Promise<string> {
  return new SignJWT({ login: s.login, avatar: s.avatar })
    .setProtectedHeader({ alg: 'HS256' })
    .setSubject(String(s.id))
    .setIssuedAt()
    .setExpirationTime(`${SESSION_TTL}s`)
    .setIssuer('gantry-portal')
    .sign(key(secret));
}

export async function verifySession(token: string | undefined, secret: string): Promise<Session | null> {
  if (!token) return null;
  try {
    const { payload } = await jwtVerify(token, key(secret), { issuer: 'gantry-portal', algorithms: ['HS256'] });
    const id = Number(payload.sub);
    if (!Number.isSafeInteger(id) || typeof payload.login !== 'string') return null;
    return { id, login: payload.login, avatar: String(payload.avatar ?? '') };
  } catch {
    return null;
  }
}
