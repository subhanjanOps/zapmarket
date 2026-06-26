"use client";
import { useEffect } from "react";

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <div style={{ minHeight: "60vh", display: "flex", alignItems: "center", justifyContent: "center", padding: "4rem 1rem" }}>
      <div style={{ textAlign: "center", maxWidth: "24rem" }}>
        <h2 style={{ fontSize: "1.25rem", fontWeight: 700, marginBottom: "0.5rem" }}>Something went wrong</h2>
        <p style={{ color: "#6B7280", fontSize: "0.875rem", marginBottom: "1.5rem" }}>
          An unexpected error occurred. Please try again.
        </p>
        <button
          onClick={reset}
          style={{ padding: "0.5rem 1.25rem", background: "#111111", color: "#fff", border: "none", borderRadius: "0.375rem", cursor: "pointer", fontSize: "0.875rem" }}
        >
          Try again
        </button>
      </div>
    </div>
  );
}
