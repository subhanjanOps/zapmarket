const GW = process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";

export { GW };

export async function apiFetch<T>(path: string, opts?: RequestInit): Promise<T | null> {
  try {
    const r = await fetch(`${GW}${path}`, opts);
    if (!r.ok) return null;
    return r.json() as Promise<T>;
  } catch {
    return null;
  }
}

/** Decode the display name from a JWT payload (no verification — display only). */
export function decodeJwtUser(token: string): { name: string | null; email: string | null } {
  try {
    const payload = JSON.parse(Buffer.from(token.split(".")[1], "base64").toString());
    return {
      name: payload.full_name ?? payload.name ?? null,
      email: payload.email ?? null,
    };
  } catch {
    return { name: null, email: null };
  }
}

/** Returns a buyer-friendly order number from a UUID. */
export function friendlyOrderId(id: string): string {
  return `#ZM-${id.slice(-8).toUpperCase()}`;
}
