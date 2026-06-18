"use client";

const KEY = "seller_token";

export const saveToken = (t: string) => localStorage.setItem(KEY, t);
export const getToken = (): string | null => localStorage.getItem(KEY);
export const clearToken = () => localStorage.removeItem(KEY);
