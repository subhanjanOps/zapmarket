"use client";
import { getRegistry, RegistryInstance } from "@/lib/api";
import { useDataFetch } from "@/lib/hooks";
import { Wifi } from "lucide-react";
import { SkeletonTableCard } from "@/app/components/Skeleton";

/** Only allow http/https URLs to prevent javascript: protocol injection. */
function safeAddr(addr: string): string | null {
  try {
    const url = new URL(addr);
    return url.protocol === "http:" || url.protocol === "https:" ? addr : null;
  } catch {
    return null;
  }
}

export default function RegistryPage() {
  const { data: instances, loading, error, refresh } = useDataFetch<RegistryInstance[]>(
    getRegistry,
    { pollMs: 15_000 },
  );

  const rows = instances ?? [];
  const grouped = rows.reduce<Record<string, RegistryInstance[]>>((acc, inst) => {
    (acc[inst.service] ??= []).push(inst);
    return acc;
  }, {});
  const services     = Object.keys(grouped).sort();
  const totalHealthy = rows.filter((i) => i.healthy).length;

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">Service Registry</h1>
          <p className="page-subtitle">
            {rows.length} instance{rows.length !== 1 ? "s" : ""} — {totalHealthy} healthy
          </p>
        </div>
        <button className="btn btn-ghost" onClick={refresh}>Refresh</button>
      </div>

      {error && <p style={{ color: "var(--danger)", fontSize: "0.8125rem", marginBottom: "1rem" }}>{error}</p>}

      {loading && rows.length === 0 ? (
        <div style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
          <SkeletonTableCard cols={4} rows={3} />
          <SkeletonTableCard cols={4} rows={2} />
        </div>
      ) : services.length === 0 ? (
        <div className="card">
          <div className="empty-state">
            <p className="empty-state-title">No live instances</p>
            <p className="empty-state-body">Services must call registry.Heartbeat() to appear here</p>
          </div>
        </div>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
          {services.map((svc) => {
            const insts        = grouped[svc];
            const healthyCount = insts.filter((i) => i.healthy).length;
            return (
              <div key={svc} className="card" style={{ padding: 0, overflow: "hidden" }}>
                <div style={{
                  display: "flex",
                  alignItems: "center",
                  gap: "0.5rem",
                  padding: "0.75rem 1rem",
                  borderBottom: "1px solid var(--border)",
                  background: "var(--surface2)",
                }}>
                  <Wifi size={14} style={{ color: "var(--success)" }} />
                  <span style={{ fontSize: "0.875rem", fontWeight: 500, color: "var(--text)" }}>{svc}</span>
                  <div style={{ marginLeft: "auto", display: "flex", gap: "0.375rem" }}>
                    <span className="badge badge-green">{healthyCount} healthy</span>
                    {insts.length - healthyCount > 0 && (
                      <span className="badge badge-red">{insts.length - healthyCount} down</span>
                    )}
                  </div>
                </div>
                <table>
                  <thead>
                    <tr>
                      <th>Health</th><th>Instance ID</th><th>Address</th><th>Started</th>
                    </tr>
                  </thead>
                  <tbody>
                    {insts.map((inst) => {
                      const href = safeAddr(inst.addr);
                      return (
                        <tr key={inst.instance_id}>
                          <td>
                            <span className={`badge ${inst.healthy ? "badge-green" : "badge-red"}`}>
                              {inst.healthy ? "up" : "down"}
                            </span>
                          </td>
                          <td>
                            <span className="mono" style={{ fontSize: "0.8125rem", color: "var(--muted)" }}>
                              {inst.instance_id.slice(0, 12)}…
                            </span>
                          </td>
                          <td>
                            {href ? (
                              <a href={href} target="_blank" rel="noopener noreferrer" className="mono" style={{ fontSize: "0.8125rem", color: "var(--accent)" }}>
                                {inst.addr}
                              </a>
                            ) : (
                              <span className="mono" style={{ fontSize: "0.8125rem", color: "var(--muted)" }}>{inst.addr}</span>
                            )}
                          </td>
                          <td style={{ color: "var(--muted)", fontSize: "0.75rem" }}>
                            {new Date(inst.started_at).toLocaleString()}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
