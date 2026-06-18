"use client";
import { useEffect, useState } from "react";
import { getToken } from "@/lib/auth";
import { getBlocklist, blockIP, unblockIP, BlocklistEntry } from "@/lib/api";
import { ShieldOff, Plus, Trash2 } from "lucide-react";

export default function BlocklistPage() {
  const [entries, setEntries] = useState<BlocklistEntry[]>([]);
  const [error, setError] = useState("");
  const [refreshKey, setRefreshKey] = useState(0);
  const [newIP, setNewIP] = useState("");
  const [reason, setReason] = useState("");
  const [adding, setAdding] = useState(false);

  useEffect(() => {
    let cancelled = false;
    const token = getToken();
    if (!token) return;
    getBlocklist(token)
      .then((b) => { if (!cancelled) { setEntries(b); setError(""); } })
      .catch((e) => { if (!cancelled) setError(e.message); });
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
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "2rem" }}>
        <div>
          <h1 style={{ fontSize: "1.125rem", fontWeight: 600, color: "var(--text)", margin: 0 }}>IP Blocklist</h1>
          <p style={{ fontSize: "0.8125rem", color: "var(--muted)", margin: "0.25rem 0 0" }}>
            {entries.length} blocked IP{entries.length !== 1 ? "s" : ""}
          </p>
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
                <td colSpan={4} style={{ textAlign: "center", color: "var(--muted)", padding: "2rem" }}>
                  No blocked IPs
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
    </div>
  );
}
