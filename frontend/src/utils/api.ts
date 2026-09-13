function token(): string {
  return (import.meta.env.VITE_API_TOKEN as string | undefined) || '';
}

export function apiHeaders(extra?: HeadersInit): Headers {
  const h = new Headers(extra);
  const t = token();
  if (t) {
    h.set('Authorization', `Bearer ${t}`);
  }
  return h;
}

export function authedWSURL(base: string): string {
  const t = token();
  if (!t) return base;
  const u = new URL(base, window.location.origin);
  u.searchParams.set('access_token', t);
  return u.toString();
}
