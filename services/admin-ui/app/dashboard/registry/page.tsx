"use client";
import { useEffect, useState } from "react";
import { getToken } from "@/lib/auth";
import { getRegistry, RegistryInstance } from "@/lib/api";
import { RefreshCw, Wifi } from "lucide-react";

export default function RegistryPage() {
  const [instances, setInstances] = useState<RegistryInstance[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    let cancelled = false;

    async function fetchData() {
      const token = getToken();
      if (!token) return;
      setLoading(true);
      try {
        const data = await getRegistry(token);
        if (!cancelled) {
          setInstances(data);
          setLastUpdated(new Date());
          setError("");
        }
      } catch (e: unknown) {
        if (!cancelled) setError(e instanceof Error ? e.message : "Failed to load registry");
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    fetchData();
    const interval = setInterval(fetchData, 15_000);
    return () => {
      cancelled = true;
      clearInterval(interval);
    };
  }, [refreshKey]);

  const grouped = instances.reduce<Record<string, RegistryInstance[]>>((acc, inst) => {
    (acc[inst.service] ??= []).push(inst);
    return acc;
  }, {});
  const services = Object.keys(grouped).sort();

  const totalHealthy = instances.filter((i) => i.healthy).length;

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "2rem" }}>
        <div>
          <h1 style={{ fontSize: "1.125rem", fontWeight: 600, color: "var(--text)", margin: 0 }}>Service Registry</h1>
          <p style={{ fontSize: "0.8125rem", color: "var(--muted)", margin: "0.25rem 0 0" }}>
            {instances.length} instance{instances.length !== 1 ? "s" : ""} — {totalHealthy} healthy
            {lastUpdated && <span> · {lastUpdated.toLocaleTimeString()}</span>}
          </p>
        </div>
        <button className="btn btn-ghost" onClick={() => setRefreshKey((k) => k + 1)}>
          <RefreshCw size={14} />
        </button>
      </div>

      {error && <p style={{ color: "var(--danger)", fontSize: "0.8125rem", marginBottom: "1rem" }}>{error}</p>}

      {loading && instances.length === 0 ? (
        <p style={{ color: "var(--muted)", fontSize: "0.8125rem" }}>Loading…</p>
      ) : services.length === 0 ? (
        <div className="card" style={{ textAlign: "center", padding: "3rem" }}>
          <p style={{ color: "var(--muted)" }}>No live instances found.</p>
          <p style={{ fontSize: "0.75rem", color: "var(--muted)", marginTop: "0.5rem" }}>
            Services must call registry.Heartbeat() to appear here.
          </p>
        </div>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
          {services.map((svc) => {
            const insts = grouped[svc];
            const healthyCount = insts.filter((i) => i.healthy).length;
            return (
              <div key={svc} className="card" style={{ padding: 0, overflow: "hidden" }}>
                <div style={{
                  display: "flex", alignItems: "center", gap: "0.5rem",
                  padding: "0.75rem 1rem", borderBottom: "1px solid var(--border)",
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
                      <th>Health</th>
                      <th>Instance ID</th>
                      <th>Address</th>
                      <th>Started</th>
                    </tr>
                  </thead>
                  <tbody>
                    {insts.map((inst) => (
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
                          <a
                            href={inst.addr}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="mono"
                            style={{ fontSize: "0.8125rem", color: "var(--accent)" }}
                          >
                            {inst.addr}
                          </a>
                        </td>
                        <td style={{ color: "var(--muted)", fontSize: "0.75rem" }}>
                          {new Date(inst.started_at).toLocaleString()}
                        </td>
                      </tr>
                    ))}
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
