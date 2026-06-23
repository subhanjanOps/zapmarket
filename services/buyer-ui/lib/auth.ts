export interface BuyerUser {
  id: string;
  email: string;
  role: string;
}

export async function getUser(): Promise<BuyerUser | null> {
  try {
    const res = await fetch("/api/auth/me", { cache: "no-store" });
    if (!res.ok) return null;
    const { user } = await res.json();
    return user ?? null;
  } catch {
    return null;
  }
}

export async function logout(): Promise<void> {
  await fetch("/api/auth/logout", { method: "POST", cache: "no-store" });
}
