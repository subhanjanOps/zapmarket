"use client";

import { createContext, useCallback, useContext, useEffect, useRef, useState } from "react";

// ── Module-level singletons (set by DialogProvider on mount) ─────────────────
// Fallback to browser native so anything that renders before the provider
// mounts still has a working implementation.

let _showAlert: (msg: string) => Promise<void> = (msg) => {
  // eslint-disable-next-line no-alert
  window.alert(msg);
  return Promise.resolve();
};
let _showConfirm: (msg: string) => Promise<boolean> = (msg) =>
  // eslint-disable-next-line no-alert
  Promise.resolve(window.confirm(msg));

export function showAlert(message: string): Promise<void> {
  return _showAlert(message);
}

export function showConfirm(message: string): Promise<boolean> {
  return _showConfirm(message);
}

// ── Internal context (optional — components can use module fns directly) ──────

type DialogEntry = {
  type: "alert" | "confirm";
  message: string;
  resolve: (v: boolean) => void;
};

const Ctx = createContext<{
  showAlert: (msg: string) => Promise<void>;
  showConfirm: (msg: string) => Promise<boolean>;
} | null>(null);

export function useDialog() {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useDialog must be used inside <DialogProvider>");
  return ctx;
}

// ── Provider ──────────────────────────────────────────────────────────────────

export function DialogProvider({ children }: { children: React.ReactNode }) {
  const [dialog, setDialog] = useState<DialogEntry | null>(null);
  const resolveRef = useRef<((v: boolean) => void) | null>(null);

  const alert = useCallback((message: string): Promise<void> => {
    return new Promise<void>((res) => {
      const resolve = (v: boolean) => { void v; res(); };
      resolveRef.current = resolve;
      setDialog({ type: "alert", message, resolve });
    });
  }, []);

  const confirm = useCallback((message: string): Promise<boolean> => {
    return new Promise<boolean>((res) => {
      resolveRef.current = res;
      setDialog({ type: "confirm", message, resolve: res });
    });
  }, []);

  // Wire module-level singletons to this provider instance
  useEffect(() => {
    _showAlert = alert;
    _showConfirm = confirm;
    return () => {
      _showAlert = (msg) => { window.alert(msg); return Promise.resolve(); };
      _showConfirm = (msg) => Promise.resolve(window.confirm(msg));
    };
  }, [alert, confirm]);

  // Keyboard: Escape → dismiss / cancel; Enter → confirm primary action
  useEffect(() => {
    if (!dialog) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") dismiss(false);
      if (e.key === "Enter") dismiss(dialog!.type === "alert" ? true : false);
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dialog]);

  function dismiss(value: boolean) {
    dialog?.resolve(value);
    setDialog(null);
  }

  return (
    <Ctx.Provider value={{ showAlert: alert, showConfirm: confirm }}>
      {children}
      {dialog && (
        <div
          style={{
            position: "fixed", inset: 0, zIndex: 9999,
            display: "flex", alignItems: "center", justifyContent: "center",
            background: "rgba(0,0,0,0.45)", backdropFilter: "blur(2px)",
            padding: "1rem", animation: "overlay-in 0.18s ease",
          }}
          onMouseDown={(e) => { if (e.target === e.currentTarget) dismiss(false); }}
        >
          <div
            role="dialog"
            aria-modal="true"
            className="dialog-card"
            style={{
              background: "var(--surface)",
              border: "1px solid var(--border)",
              borderRadius: 12,
              boxShadow: "0 24px 64px rgba(0,0,0,0.3)",
              maxWidth: 420,
              width: "100%",
              padding: "1.75rem",
              display: "flex",
              flexDirection: "column",
              gap: "1.25rem",
            }}
          >
            {/* Icon + title */}
            <div style={{ display: "flex", alignItems: "center", gap: "0.75rem" }}>
              <div
                style={{
                  width: 36, height: 36, borderRadius: "50%", flexShrink: 0,
                  display: "flex", alignItems: "center", justifyContent: "center",
                  background: dialog.type === "confirm"
                    ? "color-mix(in srgb, var(--danger) 12%, transparent)"
                    : "color-mix(in srgb, var(--accent) 12%, transparent)",
                  color: dialog.type === "confirm" ? "var(--danger)" : "var(--accent)",
                  fontSize: "1.0625rem",
                }}
              >
                {dialog.type === "confirm" ? "⚠" : "ℹ"}
              </div>
              <span style={{ fontWeight: 600, fontSize: "0.9375rem", color: "var(--text)" }}>
                {dialog.type === "confirm" ? "Confirm action" : "Notice"}
              </span>
            </div>

            {/* Message */}
            <p style={{
              margin: 0, fontSize: "0.875rem", color: "var(--text-2)",
              lineHeight: 1.55, whiteSpace: "pre-wrap",
            }}>
              {dialog.message}
            </p>

            {/* Actions */}
            <div style={{ display: "flex", gap: "0.5rem", justifyContent: "flex-end" }}>
              {dialog.type === "confirm" && (
                <button
                  className="btn btn-ghost"
                  style={{ fontSize: "0.875rem", padding: "0.5rem 1.125rem" }}
                  onClick={() => dismiss(false)}
                  autoFocus
                >
                  Cancel
                </button>
              )}
              <button
                className={`btn ${dialog.type === "confirm" ? "btn-danger" : "btn-primary"}`}
                style={{ fontSize: "0.875rem", padding: "0.5rem 1.125rem" }}
                onClick={() => dismiss(true)}
                autoFocus={dialog.type === "alert"}
              >
                {dialog.type === "confirm" ? "Confirm" : "OK"}
              </button>
            </div>
          </div>
        </div>
      )}
    </Ctx.Provider>
  );
}
