"use client";

import { useCallback, useEffect, useRef, useState } from "react";

type ToastType = "loading" | "success" | "error";

type ToastItem = {
  id: number;
  message: string;
  type: ToastType;
  duration: number | undefined;
};

let _nextId = 0;
let _add: ((item: ToastItem) => void) | null = null;
let _remove: ((id: number) => void) | null = null;

/** Shows a toast and returns a function that dismisses it. */
export function showToast(
  message: string,
  type: ToastType = "success",
  duration?: number,
): () => void {
  const id = ++_nextId;
  const defaultDuration = type === "loading" ? undefined : type === "error" ? 6000 : 4000;
  _add?.({ id, message, type, duration: duration ?? defaultDuration });
  return () => _remove?.(id);
}

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const timers = useRef<Map<number, ReturnType<typeof setTimeout>>>(new Map());

  const remove = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
    const t = timers.current.get(id);
    if (t) { clearTimeout(t); timers.current.delete(id); }
  }, []);

  const add = useCallback((item: ToastItem) => {
    setToasts((prev) => [...prev, item]);
    if (item.duration) {
      const t = setTimeout(() => remove(item.id), item.duration);
      timers.current.set(item.id, t);
    }
  }, [remove]);

  useEffect(() => {
    _add = add;
    _remove = remove;
    return () => { _add = null; _remove = null; };
  }, [add, remove]);

  if (toasts.length === 0) return <>{children}</>;

  return (
    <>
      {children}
      <div style={{
        position: "fixed", bottom: "1.5rem", right: "1.5rem",
        zIndex: 10000, display: "flex", flexDirection: "column",
        gap: "0.5rem", alignItems: "flex-end", pointerEvents: "none",
      }}>
        {toasts.map((t) => (
          <div key={t.id} className="toast-card" style={{
            pointerEvents: "auto",
            background: "var(--surface)",
            border: `1px solid ${
              t.type === "error" ? "var(--danger)"
              : t.type === "success" ? "color-mix(in srgb, var(--success) 40%, var(--border))"
              : "var(--border)"
            }`,
            borderRadius: 10,
            boxShadow: "0 8px 32px rgba(0,0,0,0.2)",
            padding: "0.7rem 0.875rem",
            display: "flex", alignItems: "center", gap: "0.625rem",
            minWidth: 220, maxWidth: 340,
            fontSize: "0.8125rem", color: "var(--text)",
          }}>
            {t.type === "loading" && <Spinner />}
            {t.type === "success" && (
              <span style={{ color: "var(--success)", fontSize: "1rem", lineHeight: 1, flexShrink: 0 }}>✓</span>
            )}
            {t.type === "error" && (
              <span style={{ color: "var(--danger)", fontSize: "1rem", lineHeight: 1, flexShrink: 0 }}>✕</span>
            )}
            <span style={{ flex: 1, lineHeight: 1.4 }}>{t.message}</span>
            <button
              onClick={() => remove(t.id)}
              style={{
                background: "none", border: "none", cursor: "pointer",
                color: "var(--muted)", padding: "0 0.125rem",
                fontSize: "1rem", lineHeight: 1, flexShrink: 0,
              }}
              aria-label="Dismiss"
            >
              ×
            </button>
          </div>
        ))}
      </div>
    </>
  );
}

function Spinner() {
  return (
    <span style={{
      display: "inline-block", width: 14, height: 14, flexShrink: 0,
      border: "2px solid var(--border)", borderTopColor: "var(--accent)",
      borderRadius: "50%", animation: "toast-spin 0.65s linear infinite",
    }} />
  );
}
