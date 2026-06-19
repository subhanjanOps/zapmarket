"use client";

/** True when the non-httpOnly hint cookie is present, meaning the httpOnly JWT cookie is also set. */
export function isAuthenticated(): boolean {
  if (typeof document === "undefined") return false;
  return document.cookie.split(";").some((c) => c.trim().startsWith("gw_auth_hint="));
}

/** Clears both auth cookies via the logout API route and returns. */
export async function clearToken(): Promise<void> {
  await fetch("/api/auth/logout", { method: "POST" });
}
