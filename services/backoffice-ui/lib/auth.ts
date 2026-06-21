"use client";

// bo_auth_hint is a non-httpOnly cookie set by the login BFF alongside the
// httpOnly bo_token. It lets client components check login state without
// accessing the actual JWT (which is inaccessible to JS).
export const isAuthenticated = (): boolean => {
  if (typeof document === "undefined") return false;
  return document.cookie.split(";").some((c) => c.trim().startsWith("bo_auth_hint="));
};

export const clearToken = async (): Promise<void> => {
  await fetch("/api/auth/logout", { method: "POST", cache: "no-store" });
};
