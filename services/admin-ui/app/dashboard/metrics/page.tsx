"use client";
import { useEffect, useState } from "react";
import { getToken } from "@/lib/auth";
import { getMetrics, UpstreamMetric } from "@/lib/api";

export default function MetricsPage() {
  const [metrics, setMetrics] = useState<UpstreamMetric[]>([]);
  const [error, setError] = useState("");
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    let cancelled = false;
    const token = getToken();
    if (!token) return;
    getMetrics(token)
      .then((m) => { if (!cancelled) { setMetrics(m); setError(""); } })
      .catch((e) => { if (!cancelled) setError(e.message); });
    return () => { cancelled = true; };
  }, [refreshKey]);

  // auto-refresh every 10s
  useEffect(() => {
    const id = setInterval(() => setRefreshKey((k) => k + 1), 10_000);
    return () => clearInterval(id);
  }, []);

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "2rem" }}>
        <div>
          <h1 style={{ fontSize: "1.125rem", fontWeight: 600, color: "var(--text)", margin: 0 }}>Metrics</h1>
          <p style={{ fontSize: "0.8125rem", color: "var(--muted)", margin: "0.25rem 0 0" }}>
            Per-upstream request statistics — refreshes every 10s
          </p>
        </div>
        <button className="btn btn-ghost" onClick={() => setRefreshKey((k) => k + 1)}>Refresh</button>
      </div>

      {error && <p style={{ color: "var(--danger)", fontSize: "0.8125rem", marginBottom: "1rem" }}>{error}</p>}

      <div className="card" style={{ padding: 0 }}>
        <table>
          <thead>
            <tr>
              <th>Upstream</th>
              <th>Total Requests</th>
              <th>req / min</th>
              <th>5xx Errors</th>
              <th>Error Rate</th>
            </tr>
          </thead>
          <tbody>
            {metrics.length === 0 ? (
              <tr>
                <td colSpan={5} style={{ textAlign: "center", color: "var(--muted)", padding: "2rem" }}>
                  No traffic yet
                </td>
              </tr>
            ) : (
              metrics
                .slice()
                .sort((a, b) => b.total - a.total)
                .map((m) => (
                  <tr key={m.name}>
                    <td className="mono">{m.name}</td>
                    <td>{m.total.toLocaleString()}</td>
                    <td>
                      <span style={{ color: m.req_per_min > 0 ? "var(--text)" : "var(--muted)" }}>
                        {m.req_per_min.toFixed(1)}
                      </span>
                    </td>
                    <td>{m.errors > 0 ? <span style={{ color: "var(--danger)" }}>{m.errors}</span> : "0"}</td>
                    <td>
                      <span className={`badge ${m.error_rate > 0.1 ? "badge-red" : m.error_rate > 0 ? "badge-yellow" : "badge-green"}`}>
                        {(m.error_rate * 100).toFixed(1)}%
                      </span>
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
