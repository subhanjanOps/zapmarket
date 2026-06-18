"use client";

const KEY = "bo_token";

export const saveToken = (t: string) => {
  if (t && t !== "undefined" && t !== "null") localStorage.setItem(KEY, t);
};
export const getToken = (): string | null => {
  const t = localStorage.getItem(KEY);
  return t && t !== "undefined" && t !== "null" ? t : null;
};
export const clearToken = () => localStorage.removeItem(KEY);
