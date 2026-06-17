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

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-semibold">Service Registry</h1>
          <p className="text-sm mt-1" style={{ color: "var(--muted)" }}>
            {instances.length} live instance{instances.length !== 1 ? "s" : ""} across{" "}
            {services.length} service{services.length !== 1 ? "s" : ""}
            {lastUpdated && <span className="ml-2">· updated {lastUpdated.toLocaleTimeString()}</span>}
          </p>
        </div>
        <button className="btn btn-ghost" onClick={() => setRefreshKey((k) => k + 1)}>
          <RefreshCw size={14} />
        </button>
      </div>

      {error && <p className="mb-4 text-sm" style={{ color: "var(--danger)" }}>{error}</p>}

      {loading && instances.length === 0 ? (
        <p className="text-sm" style={{ color: "var(--muted)" }}>Loading…</p>
      ) : services.length === 0 ? (
        <div className="card text-center py-12">
          <p style={{ color: "var(--muted)" }}>No live instances found.</p>
          <p className="text-xs mt-2" style={{ color: "var(--muted)" }}>
            Services must call registry.Heartbeat() to appear here.
          </p>
        </div>
      ) : (
        <div className="flex flex-col gap-4">
          {services.map((svc) => (
            <div key={svc} className="card p-0 overflow-hidden">
              <div
                className="flex items-center gap-2 px-4 py-3"
                style={{ borderBottom: "1px solid var(--border)", background: "var(--surface2)" }}
              >
                <Wifi size={14} style={{ color: "var(--success)" }} />
                <span className="font-medium text-sm">{svc}</span>
                <span className="badge badge-green ml-auto">
                  {grouped[svc].length} instance{grouped[svc].length !== 1 ? "s" : ""}
                </span>
              </div>
              <table>
                <thead>
                  <tr>
                    <th>Instance ID</th>
                    <th>Address</th>
                    <th>Started</th>
                  </tr>
                </thead>
                <tbody>
                  {grouped[svc].map((inst) => (
                    <tr key={inst.instance_id}>
                      <td>
                        <span className="mono" style={{ fontSize: "0.8125rem", color: "var(--muted)" }}>
                          {inst.instance_id}
                        </span>
                      </td>
                      <td>
                        <a
                          href={inst.addr}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="mono"
                          style={{ fontSize: "0.8125rem", color: "var(--accent-hover)" }}
                        >
                          {inst.addr}
                        </a>
                      </td>
                      <td className="text-xs" style={{ color: "var(--muted)" }}>
                        {new Date(inst.started_at).toLocaleString()}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
