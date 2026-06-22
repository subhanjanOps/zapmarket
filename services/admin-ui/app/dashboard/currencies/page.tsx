"use client";
import { useState } from "react";
import { getCurrencies, toggleCurrency, Currency } from "@/lib/api";
import { useDataFetch } from "@/lib/hooks";
import { SkeletonTableCard } from "@/app/components/Skeleton";

export default function CurrenciesPage() {
  const { data, loading, error: fetchError, refresh } = useDataFetch<Currency[]>(getCurrencies);
  const currencies = data ?? [];
  const [actionError, setActionError] = useState("");
  const [togglingCode, setTogglingCode] = useState<string | null>(null);

  const error = actionError || fetchError;

  async function handleToggle(c: Currency) {
    setTogglingCode(c.code);
    setActionError("");
    try {
      await toggleCurrency(c.code, !c.enabled);
      refresh();
    } catch (e: unknown) {
      setActionError(e instanceof Error ? e.message : String(e));
    } finally {
      setTogglingCode(null);
    }
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">Currencies</h1>
          <p className="page-subtitle">
            {currencies.filter((c) => c.enabled).length} of {currencies.length} enabled
          </p>
        </div>
        <button className="btn btn-ghost" onClick={refresh}>Refresh</button>
      </div>

      {error && (
        <p style={{ color: "var(--danger)", fontSize: "0.8125rem", marginBottom: "1rem" }}>{error}</p>
      )}

      {loading && currencies.length === 0 ? (
        <SkeletonTableCard cols={5} rows={10} />
      ) : null}

      <div
        className="card"
        style={{
          padding: 0,
          display: loading && currencies.length === 0 ? "none" : undefined,
          overflowX: "auto",
        }}
      >
        <table style={{ minWidth: "32rem" }}>
          <thead>
            <tr>
              <th style={{ width: "4rem" }}>Flag</th>
              <th style={{ width: "6rem" }}>Code</th>
              <th>Name</th>
              <th style={{ width: "7rem" }}>Decimals</th>
              <th style={{ width: "9rem" }}>Status</th>
            </tr>
          </thead>
          <tbody>
            {currencies.length === 0 ? (
              <tr>
                <td colSpan={5}>
                  <div className="empty-state">
                    <p className="empty-state-title">No currencies found</p>
                    <p className="empty-state-body">Run migrations to seed the currencies table</p>
                  </div>
                </td>
              </tr>
            ) : (
              currencies.map((c) => (
                <tr key={c.code} data-status={c.enabled ? "ok" : "off"}>
                  <td style={{ fontSize: "1.25rem", textAlign: "center" }}>{c.flag}</td>
                  <td className="mono" style={{ fontWeight: 600 }}>{c.code}</td>
                  <td style={{ color: "var(--text-2)" }}>{c.name}</td>
                  <td style={{ color: "var(--muted)", textAlign: "center" }}>{c.decimals}</td>
                  <td>
                    <button
                      onClick={() => handleToggle(c)}
                      disabled={togglingCode === c.code}
                      style={{ background: "none", border: "none", cursor: togglingCode === c.code ? "wait" : "pointer", padding: 0 }}
                      title={c.enabled ? "Click to disable" : "Click to enable"}
                    >
                      <span className={`badge ${c.enabled ? "badge-green" : "badge-red"}`}>
                        {togglingCode === c.code ? "…" : c.enabled ? "enabled" : "disabled"}
                      </span>
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
