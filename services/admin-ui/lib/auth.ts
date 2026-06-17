"use client";

const KEY = "gw_admin_token";

export const saveToken = (t: string) => localStorage.setItem(KEY, t);
export const getToken = (): string | null => localStorage.getItem(KEY);
export const clearToken = () => localStorage.removeItem(KEY);
