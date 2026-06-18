"use client";
import { useEffect, useState } from "react";
import { getToken } from "@/lib/auth";
import { getStats, Stats } from "@/lib/api";

function StatCard({ label, value, sub }: { label: string; value: string | number; sub?: string }) {
  return (
    <div className="card" style={{ minWidth: "10rem" }}>
      <div style={{ fontSize: "0.6875rem", color: "var(--muted)", textTransform: "uppercase", letterSpacing: "0.08em", fontFamily: '"JetBrains Mono", monospace', marginBottom: "0.5rem" }}>
        {label}
      </div>
      <div style={{ fontSize: "2rem", fontWeight: 600, color: "var(--text)", lineHeight: 1.1 }}>
        {value}
      </div>
      {sub && (
        <div style={{ fontSize: "0.75rem", color: "var(--muted)", marginTop: "0.25rem" }}>
          {sub}
        </div>
      )}
    </div>
  );
}

function statusBadge(code: number) {
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
      .then((s) => { if (!cancelled) setStats(s); })
      .catch((e) => { if (!cancelled) setError(e.message); });

    return () => { cancelled = true; };
  }, [refreshKey]);

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "2rem" }}>
        <div>
          <h1 style={{ fontSize: "1.125rem", fontWeight: 600, color: "var(--text)", margin: 0 }}>Overview</h1>
          <p style={{ fontSize: "0.8125rem", color: "var(--muted)", margin: "0.25rem 0 0" }}>
            Gateway health at a glance
          </p>
        </div>
        <button className="btn btn-ghost" onClick={() => setRefreshKey((k) => k + 1)}>
          Refresh
        </button>
      </div>

      {error && (
        <div style={{ color: "var(--danger)", fontSize: "0.8125rem", marginBottom: "1.5rem" }}>
          {error}
        </div>
      )}

      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(11rem, 1fr))", gap: "1rem", marginBottom: "2rem" }}>
        <StatCard label="Active Routes" value={stats?.active_routes ?? "—"} sub={`of ${stats?.total_routes ?? "—"} total`} />
        <StatCard label="Live Instances" value={stats?.live_instances ?? "—"} sub={`across ${stats?.live_services ?? "—"} services`} />
        <StatCard
          label="Top req / min"
          value={stats?.upstreams?.length
            ? Math.max(...stats.upstreams.map((u) => u.req_per_min)).toFixed(1)
            : "—"}
        />
        <StatCard
          label="Total 5xx"
          value={stats?.upstreams?.reduce((s, u) => s + u.errors, 0) ?? "—"}
        />
      </div>

      {stats?.upstreams && stats.upstreams.length > 0 && (
        <div className="card" style={{ marginBottom: "2rem", padding: 0 }}>
          <div style={{ padding: "1rem 1.25rem", borderBottom: "1px solid var(--border)" }}>
            <span style={{ fontSize: "0.8125rem", fontWeight: 500, color: "var(--text)" }}>Upstream Metrics</span>
          </div>
          <table>
            <thead>
              <tr>
                <th>Upstream</th>
                <th>Total</th>
                <th>req / min</th>
                <th>Errors</th>
                <th>Error Rate</th>
              </tr>
            </thead>
            <tbody>
              {stats.upstreams.map((u) => (
                <tr key={u.name}>
                  <td className="mono">{u.name}</td>
                  <td>{u.total.toLocaleString()}</td>
                  <td>{u.req_per_min.toFixed(1)}</td>
                  <td>{u.errors}</td>
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

      <div className="card" style={{ padding: 0 }}>
        <div style={{ padding: "1rem 1.25rem", borderBottom: "1px solid var(--border)" }}>
          <span style={{ fontSize: "0.8125rem", fontWeight: 500, color: "var(--text)" }}>Recent Activity</span>
        </div>
        {!stats?.recent_audit?.length ? (
          <div style={{ padding: "2rem", textAlign: "center", color: "var(--muted)", fontSize: "0.8125rem" }}>
            No recent activity
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
                  <td className="mono" style={{ color: "var(--muted)" }}>
                    {new Date(e.ts).toLocaleTimeString()}
                  </td>
                  <td><span className="badge badge-gray">{e.method}</span></td>
                  <td className="mono">{e.path}</td>
                  <td>{statusBadge(e.status_code)}</td>
                  <td className="mono" style={{ color: "var(--muted)", fontSize: "0.75rem" }}>{e.event}</td>
                  <td className="mono" style={{ color: "var(--muted)", fontSize: "0.75rem" }}>{e.ip}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
