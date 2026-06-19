"use client";
import { useEffect, useRef, useState } from "react";
import { getStats, Stats } from "@/lib/api";
import { useDataFetch } from "@/lib/hooks";
import { SkeletonStatCards, SkeletonTableCard } from "@/app/components/Skeleton";
import { RefreshCw } from "lucide-react";

/* ── count-up animation hook ────────────────────────────────────────────── */
function useCountUp(target: number | null, duration = 700) {
  const [value, setValue] = useState(0);
  const prev = useRef<number | null>(null);

  useEffect(() => {
    if (target === null) return;
    if (prev.current === target) return;
    prev.current = target;
    const from = 0;
    const start = performance.now();
    function step(now: number) {
      const t = Math.min((now - start) / duration, 1);
      const eased = 1 - Math.pow(1 - t, 3);
      setValue(Math.round(from + (target! - from) * eased));
      if (t < 1) requestAnimationFrame(step);
    }
    requestAnimationFrame(step);
  }, [target, duration]);

  return value;
}

/* ── stat card with count-up + mini progress bar ─────────────────────────── */
function StatCard({
  label, value, sub, accent, danger, progress, progressColor, delay = 0,
}: {
  label: string;
  value: number | string | null;
  sub?: string;
  accent?: boolean;
  danger?: boolean;
  progress?: number;  // 0-100
  progressColor?: string;
  delay?: number;
}) {
  const numTarget = typeof value === "number" ? value : null;
  const animated  = useCountUp(numTarget);
  const display   = numTarget !== null ? animated : value ?? "—";

  const glowClass = danger && numTarget ? "card card-glow-danger" : accent ? "card card-glow" : "card";

  return (
    <div
      className={`${glowClass} animate-in`}
      style={{ padding: "1.25rem 1.375rem", display: "flex", flexDirection: "column", gap: "0.5rem", animationDelay: `${delay}s` }}
    >
      <div style={{
        fontSize: "0.5625rem",
        fontWeight: 600,
        color: "var(--muted)",
        textTransform: "uppercase",
        letterSpacing: "0.09em",
      }}>
        {label}
      </div>

      <div style={{
        fontFamily: '"JetBrains Mono", monospace',
        fontSize: "2rem",
        fontWeight: 600,
        color: danger && numTarget ? "var(--danger)" : accent ? "var(--accent)" : "var(--text)",
        lineHeight: 1.1,
        letterSpacing: "-0.03em",
        fontFeatureSettings: '"tnum"',
      }}>
        {display}
      </div>

      {progress !== undefined && (
        <div className="progress-track" style={{ marginTop: "0.125rem" }}>
          <div
            className={`progress-fill ${progressColor ?? "progress-fill-accent"}`}
            style={{ width: `${Math.min(progress, 100)}%` }}
          />
        </div>
      )}

      {sub && (
        <div style={{ fontSize: "0.6875rem", color: "var(--muted)", letterSpacing: "0.01em" }}>{sub}</div>
      )}
    </div>
  );
}

function StatusBadge({ code }: { code: number }) {
  const cls = code < 300 ? "badge-green" : code < 400 ? "badge-blue" : code < 500 ? "badge-yellow" : "badge-red";
  return <span className={`badge ${cls}`}>{code}</span>;
}

/* ── upstream health bars ─────────────────────────────────────────────────── */
function UpstreamHealthBar({ name, total, rpm, errors, errorRate, maxTotal, delay = 0 }: {
  name: string; total: number; rpm: number; errors: number; errorRate: number; maxTotal: number; delay?: number;
}) {
  const pct = maxTotal > 0 ? (total / maxTotal) * 100 : 0;
  const fillColor = errorRate > 0.1 ? "progress-fill-danger" : errorRate > 0 ? "progress-fill-warning" : "progress-fill-success";

  return (
    <div className="animate-in" style={{ display: "flex", flexDirection: "column", gap: "0.375rem", animationDelay: `${delay}s` }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline" }}>
        <span style={{ fontFamily: '"JetBrains Mono", monospace', fontSize: "0.75rem", color: "var(--text)", fontWeight: 500 }}>{name}</span>
        <span style={{ fontFamily: '"JetBrains Mono", monospace', fontSize: "0.625rem", color: "var(--muted)" }}>
          {total.toLocaleString()} req · {rpm.toFixed(1)}/min
          {errors > 0 && <span style={{ color: "var(--danger)", marginLeft: "0.375rem" }}>{errors} err</span>}
        </span>
      </div>
      <div className="progress-track" style={{ height: "5px" }}>
        <div className={`progress-fill ${fillColor}`} style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}

export default function OverviewPage() {
  const { data: stats, loading, error, refresh } = useDataFetch<Stats>(getStats);
  const [refreshing, setRefreshing] = useState(false);

  const topRpm      = stats?.upstreams?.length ? Math.max(...stats.upstreams.map((u) => u.req_per_min)) : null;
  const totalErrors = stats?.upstreams?.reduce((s, u) => s + u.errors, 0) ?? null;
  const maxTotal    = stats?.upstreams?.length ? Math.max(...stats.upstreams.map((u) => u.total), 1) : 1;

  const activeRatio = stats ? (stats.active_routes / Math.max(stats.total_routes, 1)) * 100 : 0;

  async function handleRefresh() {
    setRefreshing(true);
    refresh();
    setTimeout(() => setRefreshing(false), 600);
  }

  return (
    <div>
      <div className="page-header animate-in">
        <div>
          <h1 className="page-title">Overview</h1>
          <p className="page-subtitle">Gateway health at a glance</p>
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

      {error && (
        <div className="animate-in" style={{
          marginBottom: "1.25rem", padding: "0.625rem 1rem", borderRadius: "8px",
          background: "color-mix(in srgb, var(--danger) 8%, transparent)",
          border: "1px solid color-mix(in srgb, var(--danger) 25%, transparent)",
          fontSize: "0.8125rem", color: "var(--danger)",
        }}>
          {error}
        </div>
      )}

      {loading && !stats ? (
        <>
          <SkeletonStatCards count={4} />
          <SkeletonTableCard cols={5} rows={4} />
        </>
      ) : (
        <>
          {/* ── KPI Cards ────────────────────────────────────────────────── */}
          <div style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fill, minmax(11rem, 1fr))",
            gap: "1rem",
            marginBottom: "1.75rem",
          }}>
            <StatCard
              label="Active Routes"
              value={stats?.active_routes ?? null}
              sub={stats ? `${stats.total_routes} configured` : undefined}
              progress={activeRatio}
              progressColor="progress-fill-success"
              delay={0}
            />
            <StatCard
              label="Live Instances"
              value={stats?.live_instances ?? null}
              sub={stats ? `${stats.live_services} services` : undefined}
              progress={stats ? Math.min((stats.live_instances / Math.max(stats.total_routes, 1)) * 100, 100) : 0}
              progressColor="progress-fill-accent"
              delay={0.04}
            />
            <StatCard
              label="Top req / min"
              value={topRpm !== null ? parseFloat(topRpm.toFixed(1)) : null}
              sub="sliding 60s window"
              accent={topRpm !== null && topRpm > 0}
              progress={topRpm ? Math.min((topRpm / 100) * 100, 100) : 0}
              progressColor="progress-fill-accent"
              delay={0.08}
            />
            <StatCard
              label="5xx Errors"
              value={totalErrors}
              sub="all upstreams"
              danger={!!totalErrors}
              progress={totalErrors ? Math.min(totalErrors * 10, 100) : 0}
              progressColor="progress-fill-danger"
              delay={0.12}
            />
          </div>

          {/* ── Upstream Traffic Visualization ────────────────────────────── */}
          {stats?.upstreams && stats.upstreams.length > 0 && (
            <div className="card animate-in anim-d3" style={{ marginBottom: "1.5rem", padding: "1.25rem 1.5rem" }}>
              <div style={{
                display: "flex", justifyContent: "space-between", alignItems: "center",
                marginBottom: "1.25rem",
              }}>
                <span className="card-title">Upstream Traffic</span>
                <span className="live-badge">
                  <span className="live-dot" />
                  live
                </span>
              </div>
              <div style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
                {stats.upstreams
                  .slice()
                  .sort((a, b) => b.total - a.total)
                  .map((u, i) => (
                    <UpstreamHealthBar
                      key={u.name}
                      name={u.name}
                      total={u.total}
                      rpm={u.req_per_min}
                      errors={u.errors}
                      errorRate={u.error_rate}
                      maxTotal={maxTotal}
                      delay={0.16 + i * 0.05}
                    />
                  ))}
              </div>
            </div>
          )}

          {/* ── Upstream Metrics Table ────────────────────────────────────── */}
          {stats?.upstreams && stats.upstreams.length > 0 && (
            <div className="card animate-in anim-d4" style={{ marginBottom: "1.5rem", padding: 0, overflow: "hidden" }}>
              <div className="card-header">
                <span className="card-title">Upstream Metrics</span>
              </div>
              <table>
                <thead>
                  <tr>
                    <th>Upstream</th><th>Total</th><th>req / min</th><th>5xx</th><th>Error Rate</th>
                  </tr>
                </thead>
                <tbody>
                  {stats.upstreams.slice().sort((a, b) => b.req_per_min - a.req_per_min).map((u) => (
                    <tr key={u.name}>
                      <td className="mono">{u.name}</td>
                      <td className="mono">{u.total.toLocaleString()}</td>
                      <td className="mono">{u.req_per_min.toFixed(1)}</td>
                      <td className="mono" style={{ color: u.errors > 0 ? "var(--danger)" : "var(--muted)" }}>{u.errors}</td>
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

          {/* ── Recent Activity ───────────────────────────────────────────── */}
          <div className="card animate-in anim-d5" style={{ padding: 0, overflow: "hidden" }}>
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
                    <th>Time</th><th>Method</th><th>Path</th><th>Status</th><th>Event</th><th>IP</th>
                  </tr>
                </thead>
                <tbody>
                  {stats.recent_audit.map((e, i) => (
                    <tr key={e.id} className="animate-in" style={{ animationDelay: `${0.24 + i * 0.04}s` }}>
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
        </>
      )}
    </div>
  );
}
