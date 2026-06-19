"use client";
import { useState, useEffect } from "react";
import { getMetrics, UpstreamMetric } from "@/lib/api";
import { useDataFetch } from "@/lib/hooks";
import { SkeletonTableCard } from "@/app/components/Skeleton";
import { RefreshCw } from "lucide-react";

function MiniBar({ value, max, fillClass = "progress-fill-accent", suffix = "" }: {
  value: number;
  max: number;
  fillClass?: string;
  suffix?: string;
}) {
  const pct = max > 0 ? Math.min((value / max) * 100, 100) : 0;
  return (
    <div style={{ display: "flex", alignItems: "center", gap: "0.625rem" }}>
      <div className="mini-bar-track">
        <div className={`mini-bar-fill ${fillClass}`} style={{ width: `${pct}%` }} />
      </div>
      <span style={{ fontFamily: '"JetBrains Mono", monospace', fontSize: "0.75rem", color: "var(--text)", minWidth: "3rem", textAlign: "right", fontFeatureSettings: '"tnum"' }}>
        {value.toLocaleString()}{suffix}
      </span>
    </div>
  );
}

function KpiCard({ label, value, accent, danger, delay = 0 }: {
  label: string; value: string; accent?: boolean; danger?: boolean; delay?: number;
}) {
  return (
    <div
      className={`card animate-in ${danger ? "card-glow-danger" : accent ? "card-glow" : ""}`}
      style={{ padding: "1rem 1.25rem", animationDelay: `${delay}s` }}
    >
      <div style={{ fontFamily: '"JetBrains Mono", monospace', fontSize: "0.5rem", fontWeight: 700, color: "var(--muted)", textTransform: "uppercase", letterSpacing: "0.12em", marginBottom: "0.4rem" }}>
        {label}
      </div>
      <div style={{
        fontFamily: '"JetBrains Mono", monospace',
        fontSize: "1.625rem",
        fontWeight: 700,
        letterSpacing: "-0.03em",
        lineHeight: 1,
        color: danger ? "var(--danger)" : accent ? "var(--accent)" : "var(--text)",
        fontFeatureSettings: '"tnum"',
      }}>
        {value}
      </div>
    </div>
  );
}

export default function MetricsPage() {
  const { data: metrics, loading, error, refresh } = useDataFetch<UpstreamMetric[]>(getMetrics, { pollMs: 10_000 });
  const [lastUpdated, setLastUpdated] = useState("");
  const [refreshing, setRefreshing] = useState(false);

  useEffect(() => {
    if (metrics) setLastUpdated(new Date().toLocaleTimeString("en-GB", { hour12: false }));
  }, [metrics]);

  const rows        = metrics ?? [];
  const totalReqs   = rows.reduce((s, m) => s + m.total, 0);
  const totalErrors = rows.reduce((s, m) => s + m.errors, 0);
  const totalRpm    = rows.reduce((s, m) => s + m.req_per_min, 0);
  const maxTotal    = Math.max(...rows.map((m) => m.total), 1);
  const maxRpm      = Math.max(...rows.map((m) => m.req_per_min), 1);
  const sorted      = rows.slice().sort((a, b) => b.total - a.total);

  function handleRefresh() {
    setRefreshing(true);
    refresh();
    setTimeout(() => setRefreshing(false), 600);
  }

  return (
    <div>
      <div className="page-header animate-in">
        <div>
          <h1 className="page-title">Metrics</h1>
          <p className="page-subtitle" style={{ display: "flex", alignItems: "center", gap: "0.5rem", flexWrap: "wrap" }}>
            Per-upstream statistics
            {lastUpdated && (
              <span className="live-badge">
                <span className="live-dot" />
                {lastUpdated}
              </span>
            )}
          </p>
        </div>
        <button
          className="btn btn-ghost"
          onClick={handleRefresh}
          disabled={refreshing}
          style={{ display: "flex", alignItems: "center", gap: "0.4rem" }}
        >
          <RefreshCw size={13} style={{ transition: "transform 0.5s", transform: refreshing ? "rotate(360deg)" : "rotate(0deg)" }} />
          Refresh
        </button>
      </div>

      {error && <p style={{ color: "var(--danger)", fontSize: "0.8125rem", marginBottom: "1rem" }}>{error}</p>}

      {!loading && (
        <div style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: "1rem", marginBottom: "1.5rem" }}>
          <KpiCard label="Total Requests"   value={totalReqs.toLocaleString()}    delay={0}    />
          <KpiCard label="Req / Min"        value={totalRpm.toFixed(1)}           accent delay={0.04} />
          <KpiCard label="5xx Errors"       value={totalErrors.toLocaleString()}  danger={totalErrors > 0} delay={0.08} />
          <KpiCard label="Active Upstreams" value={String(rows.length)}           delay={0.12} />
        </div>
      )}

      {loading && rows.length === 0 ? (
        <SkeletonTableCard cols={5} rows={5} />
      ) : (
        <div className="card animate-in anim-d3" style={{ padding: 0, overflowX: "auto" }}>
          <table style={{ minWidth: "38rem" }}>
            <thead>
              <tr>
                <th style={{ width: "14rem" }}>Upstream</th>
                <th style={{ width: "32%" }}>Total Requests</th>
                <th style={{ width: "24%" }}>Req / Min</th>
                <th style={{ width: "7rem" }}>5xx Errors</th>
                <th style={{ width: "7rem" }}>Error Rate</th>
              </tr>
            </thead>
            <tbody>
              {sorted.length === 0 ? (
                <tr>
                  <td colSpan={5}>
                    <div className="empty-state">
                      <p className="empty-state-title">No traffic yet</p>
                      <p className="empty-state-body">Metrics appear as requests flow through the gateway</p>
                    </div>
                  </td>
                </tr>
              ) : sorted.map((m, i) => (
                <tr key={m.name} className="animate-in" style={{ animationDelay: `${0.14 + i * 0.04}s` }}>
                  <td className="mono" style={{ whiteSpace: "nowrap", fontWeight: 600 }}>{m.name}</td>
                  <td style={{ paddingRight: "1.5rem" }}>
                    <MiniBar value={m.total} max={maxTotal} fillClass="progress-fill-accent" />
                  </td>
                  <td style={{ paddingRight: "1.5rem" }}>
                    <MiniBar
                      value={parseFloat(m.req_per_min.toFixed(1))}
                      max={maxRpm}
                      fillClass={m.req_per_min > 0 ? "progress-fill-success" : "progress-fill-muted"}
                    />
                  </td>
                  <td>
                    <span style={{ color: m.errors > 0 ? "var(--danger)" : "var(--muted)", fontFamily: '"JetBrains Mono", monospace', fontSize: "0.8125rem", fontFeatureSettings: '"tnum"' }}>
                      {m.errors}
                    </span>
                  </td>
                  <td>
                    <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
                      <span className={`badge ${m.error_rate > 0.1 ? "badge-red" : m.error_rate > 0 ? "badge-yellow" : "badge-green"}`}>
                        {(m.error_rate * 100).toFixed(1)}%
                      </span>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
