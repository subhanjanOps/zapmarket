"use client";
import { useEffect, useState } from "react";
import { getToken } from "@/lib/auth";
import { getBlocklist, blockIP, unblockIP, BlocklistEntry } from "@/lib/api";
import { ShieldOff, Plus, Trash2 } from "lucide-react";
import { SkeletonTableCard } from "@/app/components/Skeleton";

export default function BlocklistPage() {
  const [entries, setEntries] = useState<BlocklistEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [refreshKey, setRefreshKey] = useState(0);
  const [newIP, setNewIP] = useState("");
  const [reason, setReason] = useState("");
  const [adding, setAdding] = useState(false);

  useEffect(() => {
    let cancelled = false;
    const token = getToken();
    if (!token) return;
    setLoading(true);
    getBlocklist(token)
      .then((b) => { if (!cancelled) { setEntries(b); setError(""); } })
      .catch((e) => { if (!cancelled) setError(e.message); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [refreshKey]);

  const token = getToken()!;

  async function handleBlock() {
    if (!newIP.trim()) return;
    try {
      await blockIP(token, newIP.trim(), reason.trim() || undefined);
      setNewIP("");
      setReason("");
      setAdding(false);
      setRefreshKey((k) => k + 1);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function handleUnblock(ip: string) {
    if (!confirm(`Unblock ${ip}?`)) return;
    try {
      await unblockIP(token, ip);
      setRefreshKey((k) => k + 1);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">IP Blocklist</h1>
          <p className="page-subtitle">{entries.length} blocked IP{entries.length !== 1 ? "s" : ""}</p>
        </div>
        <div style={{ display: "flex", gap: "0.5rem" }}>
          <button className="btn btn-ghost" onClick={() => setRefreshKey((k) => k + 1)}>Refresh</button>
          <button className="btn btn-primary" onClick={() => setAdding(true)}>
            <Plus size={14} /> Block IP
          </button>
        </div>
      </div>

      {error && <p style={{ color: "var(--danger)", fontSize: "0.8125rem", marginBottom: "1rem" }}>{error}</p>}

      {adding && (
        <div className="card" style={{ marginBottom: "1.5rem" }}>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr auto", gap: "0.5rem", alignItems: "end" }}>
            <div>
              <label style={{ display: "block", fontSize: "0.75rem", color: "var(--muted)", marginBottom: "0.25rem" }}>IP Address</label>
              <input
                className="input mono"
                value={newIP}
                onChange={(e) => setNewIP(e.target.value)}
                placeholder="192.168.1.100"
                onKeyDown={(e) => e.key === "Enter" && handleBlock()}
              />
            </div>
            <div>
              <label style={{ display: "block", fontSize: "0.75rem", color: "var(--muted)", marginBottom: "0.25rem" }}>Reason (optional)</label>
              <input
                className="input"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="Abuse, rate limit exceeded…"
              />
            </div>
            <div style={{ display: "flex", gap: "0.5rem" }}>
              <button className="btn btn-danger" onClick={handleBlock}>
                <ShieldOff size={13} /> Block
              </button>
              <button className="btn btn-ghost" onClick={() => setAdding(false)}>Cancel</button>
            </div>
          </div>
        </div>
      )}

      {loading && entries.length === 0 ? (
        <SkeletonTableCard cols={4} rows={5} />
      ) : (
        <div className="card" style={{ padding: 0 }}>
          <table>
            <thead>
              <tr>
                <th>IP Address</th>
                <th>Reason</th>
                <th>Blocked At</th>
                <th style={{ width: "5rem" }}></th>
              </tr>
            </thead>
            <tbody>
              {entries.length === 0 ? (
                <tr>
                  <td colSpan={4}>
                    <div className="empty-state">
                      <p className="empty-state-title">No blocked IPs</p>
                      <p className="empty-state-body">Block an IP to prevent it from reaching the gateway</p>
                    </div>
                  </td>
                </tr>
              ) : (
                entries.map((e) => (
                  <tr key={e.ip}>
                    <td className="mono">{e.ip}</td>
                    <td style={{ color: "var(--muted)" }}>{e.reason || "—"}</td>
                    <td className="mono" style={{ color: "var(--muted)", fontSize: "0.75rem" }}>
                      {new Date(e.blocked_at).toLocaleString()}
                    </td>
                    <td>
                      <button
                        className="btn btn-danger"
                        style={{ padding: "0.3rem 0.5rem" }}
                        onClick={() => handleUnblock(e.ip)}
                        title="Unblock"
                      >
                        <Trash2 size={13} />
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
