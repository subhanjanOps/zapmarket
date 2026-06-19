"use client";
import { useState } from "react";
import { getBlocklist, blockIP, unblockIP, BlocklistEntry } from "@/lib/api";
import { useDataFetch } from "@/lib/hooks";
import { ShieldOff, Plus, Trash2 } from "lucide-react";
import { SkeletonTableCard } from "@/app/components/Skeleton";

const IP_RE = /^(\d{1,3}\.){3}\d{1,3}$|^([0-9a-fA-F]{0,4}:){2,7}[0-9a-fA-F]{0,4}$/;

function validateIP(ip: string): string | null {
  if (!ip.trim()) return "IP address is required";
  if (!IP_RE.test(ip.trim())) return "Enter a valid IPv4 or IPv6 address";
  return null;
}

export default function BlocklistPage() {
  const { data, loading, error: fetchError, refresh } = useDataFetch<BlocklistEntry[]>(getBlocklist);
  const entries = data ?? [];

  const [actionError, setActionError] = useState("");
  const [newIP, setNewIP] = useState("");
  const [ipError, setIPError] = useState("");
  const [reason, setReason] = useState("");
  const [adding, setAdding] = useState(false);
  const [confirmingIP, setConfirmingIP] = useState<string | null>(null);

  const error = actionError || fetchError;

  async function handleBlock() {
    const validationError = validateIP(newIP);
    if (validationError) { setIPError(validationError); return; }
    setIPError("");
    try {
      await blockIP(newIP.trim(), reason.trim() || undefined);
      setNewIP("");
      setReason("");
      setAdding(false);
      setActionError("");
      refresh();
    } catch (e: unknown) {
      setActionError(e instanceof Error ? e.message : String(e));
    }
  }

  async function handleUnblock(ip: string) {
    try {
      await unblockIP(ip);
      setConfirmingIP(null);
      setActionError("");
      refresh();
    } catch (e: unknown) {
      setActionError(e instanceof Error ? e.message : String(e));
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
          <button className="btn btn-ghost" onClick={refresh}>Refresh</button>
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
                onChange={(e) => { setNewIP(e.target.value); setIPError(""); }}
                placeholder="192.168.1.100"
                onKeyDown={(e) => e.key === "Enter" && handleBlock()}
              />
              {ipError && <p style={{ fontSize: "0.75rem", color: "var(--danger)", marginTop: "0.25rem", marginBottom: 0 }}>{ipError}</p>}
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
              <button className="btn btn-ghost" onClick={() => { setAdding(false); setIPError(""); }}>Cancel</button>
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
                <th style={{ width: "9rem" }}></th>
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
              ) : entries.map((e) => (
                <tr key={e.ip}>
                  <td className="mono">{e.ip}</td>
                  <td style={{ color: "var(--muted)" }}>{e.reason || "—"}</td>
                  <td className="mono" style={{ color: "var(--muted)", fontSize: "0.75rem" }}>
                    {new Date(e.blocked_at).toLocaleString()}
                  </td>
                  <td>
                    {confirmingIP === e.ip ? (
                      <div style={{ display: "flex", gap: "0.375rem", alignItems: "center" }}>
                        <span style={{ fontSize: "0.75rem", color: "var(--muted)" }}>Sure?</span>
                        <button className="btn btn-danger" style={{ padding: "0.25rem 0.5rem", fontSize: "0.75rem" }} onClick={() => handleUnblock(e.ip)}>
                          Yes
                        </button>
                        <button className="btn btn-ghost" style={{ padding: "0.25rem 0.5rem", fontSize: "0.75rem" }} onClick={() => setConfirmingIP(null)}>
                          No
                        </button>
                      </div>
                    ) : (
                      <button
                        className="btn btn-danger"
                        style={{ padding: "0.3rem 0.5rem" }}
                        onClick={() => setConfirmingIP(e.ip)}
                        title="Unblock"
                      >
                        <Trash2 size={13} />
                      </button>
                    )}
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
