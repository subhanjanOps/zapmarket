"use client";
import { useEffect, useState } from "react";
import { getToken } from "@/lib/auth";
import { getStats, Stats } from "@/lib/api";

function StatCard({ label, value, sub, accent }: {
  label: string;
  value: string | number;
  sub?: string;
  accent?: boolean;
}) {
  return (
    <div className="card" style={{ padding: "1.25rem 1.375rem" }}>
      <div style={{
        fontSize: "0.6875rem",
        fontWeight: 500,
        color: "var(--muted)",
        textTransform: "uppercase",
        letterSpacing: "0.07em",
        marginBottom: "0.625rem",
      }}>
        {label}
      </div>
      <div style={{
        fontFamily: '"Roboto Mono", monospace',
        fontSize: "2rem",
        fontWeight: 500,
        color: accent ? "var(--accent)" : "var(--text)",
        lineHeight: 1.1,
        letterSpacing: "-0.02em",
      }}>
        {value}
      </div>
      {sub && (
        <div style={{ fontSize: "0.75rem", color: "var(--muted)", marginTop: "0.3rem" }}>
          {sub}
        </div>
      )}
    </div>
  );
}

function StatusBadge({ code }: { code: number }) {
  const cls = code < 300 ? "badge-green" : code < 400 ? "badge-blue" : code < 500 ? "badge-yellow" : "badge-red";
  return <span className={`badge ${cls}`}>{code}</span>;
}

export default function OverviewPage() {
  const [stats, setStats] = useState<Stats | null>(null);
  const [error, setError] = useState("");
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    let cancelled = false;
    const token = getToken();
    if (!token) return;
    getStats(token)
      .then((s) => { if (!cancelled) { setStats(s); setError(""); } })
      .catch((e) => { if (!cancelled) setError(e.message); });
    return () => { cancelled = true; };
  }, [refreshKey]);

  const topRpm = stats?.upstreams?.length
    ? Math.max(...stats.upstreams.map((u) => u.req_per_min))
    : null;

  const totalErrors = stats?.upstreams?.reduce((s, u) => s + u.errors, 0) ?? null;

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">Overview</h1>
          <p className="page-subtitle">Gateway health at a glance</p>
        </div>
        <button className="btn btn-ghost" onClick={() => setRefreshKey((k) => k + 1)}>
          Refresh
        </button>
      </div>

      {error && (
        <div style={{
          marginBottom: "1.25rem",
          padding: "0.625rem 1rem",
          borderRadius: "8px",
          background: "color-mix(in srgb, var(--danger) 8%, transparent)",
          border: "1px solid color-mix(in srgb, var(--danger) 25%, transparent)",
          fontSize: "0.8125rem",
          color: "var(--danger)",
        }}>
          {error}
        </div>
      )}

      {/* Stat cards */}
      <div style={{
        display: "grid",
        gridTemplateColumns: "repeat(auto-fill, minmax(11rem, 1fr))",
        gap: "1rem",
        marginBottom: "1.75rem",
      }}>
        <StatCard
          label="Active Routes"
          value={stats?.active_routes ?? "—"}
          sub={stats ? `${stats.total_routes} configured` : undefined}
        />
        <StatCard
          label="Live Instances"
          value={stats?.live_instances ?? "—"}
          sub={stats ? `${stats.live_services} services` : undefined}
        />
        <StatCard
          label="Top req / min"
          value={topRpm !== null ? topRpm.toFixed(1) : "—"}
          sub="sliding 60s window"
          accent={topRpm !== null && topRpm > 0}
        />
        <StatCard
          label="5xx Errors"
          value={totalErrors !== null ? totalErrors : "—"}
          sub="all upstreams"
          accent={!!totalErrors}
        />
      </div>

      {/* Upstream metrics */}
      {stats?.upstreams && stats.upstreams.length > 0 && (
        <div className="card" style={{ marginBottom: "1.5rem", padding: 0, overflow: "hidden" }}>
          <div className="card-header">
            <span className="card-title">Upstream Metrics</span>
          </div>
          <table>
            <thead>
              <tr>
                <th>Upstream</th>
                <th>Total</th>
                <th>req / min</th>
                <th>5xx</th>
                <th>Error Rate</th>
              </tr>
            </thead>
            <tbody>
              {stats.upstreams
                .slice()
                .sort((a, b) => b.req_per_min - a.req_per_min)
                .map((u) => (
                  <tr key={u.name}>
                    <td className="mono">{u.name}</td>
                    <td className="mono">{u.total.toLocaleString()}</td>
                    <td className="mono">{u.req_per_min.toFixed(1)}</td>
                    <td className="mono" style={{ color: u.errors > 0 ? "var(--danger)" : "var(--muted)" }}>
                      {u.errors}
                    </td>
                    <td>
                      <span className={`badge ${u.error_rate > 0.1 ? "badge-red" : u.error_rate > 0 ? "badge-yellow" : "badge-green"}`}>
                        {(u.error_rate * 100).toFixed(1)}%
                      </span>
                    </td>
                  </tr>
                ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Recent activity */}
      <div className="card" style={{ padding: 0, overflow: "hidden" }}>
        <div className="card-header">
          <span className="card-title">Recent Activity</span>
          <span style={{ fontSize: "0.75rem", color: "var(--muted)" }}>last 5 events</span>
        </div>
        {!stats?.recent_audit?.length ? (
          <div className="empty-state">
            <p className="empty-state-title">No activity yet</p>
            <p className="empty-state-body">Events appear as traffic flows through the gateway</p>
          </div>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Time</th>
                <th>Method</th>
                <th>Path</th>
                <th>Status</th>
                <th>Event</th>
                <th>IP</th>
              </tr>
            </thead>
            <tbody>
              {stats.recent_audit.map((e) => (
                <tr key={e.id}>
                  <td className="mono" style={{ color: "var(--muted)", fontSize: "0.75rem" }}>
                    {new Date(e.ts).toLocaleTimeString()}
                  </td>
                  <td><span className="badge badge-gray">{e.method}</span></td>
                  <td className="mono" style={{ color: "var(--accent)" }}>{e.path}</td>
                  <td><StatusBadge code={e.status_code} /></td>
                  <td className="mono" style={{ color: "var(--muted)", fontSize: "0.6875rem" }}>{e.event}</td>
                  <td className="mono" style={{ color: "var(--muted)", fontSize: "0.6875rem" }}>{e.ip}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
