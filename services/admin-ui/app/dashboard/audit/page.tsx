"use client";
import { useEffect, useState } from "react";
import { getToken } from "@/lib/auth";
import { getAudit, blockIP, AuditEntry } from "@/lib/api";
import { RefreshCw, Search, Play, Square, ShieldOff } from "lucide-react";

const EVENTS = ["", "AUTH_REJECTED", "RATE_LIMITED", "UPSTREAM_5XX", "CIRCUIT_OPEN", "ROUTE_CONFLICT"];

function EventBadge({ event }: { event: string }) {
  const map: Record<string, string> = {
    AUTH_REJECTED: "badge badge-red",
    RATE_LIMITED: "badge badge-yellow",
    UPSTREAM_5XX: "badge badge-red",
    CIRCUIT_OPEN: "badge badge-yellow",
    ROUTE_CONFLICT: "badge badge-blue",
  };
  return <span className={map[event] ?? "badge badge-gray"}>{event}</span>;
}

export default function AuditPage() {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [searchKey, setSearchKey] = useState(0);
  const [tailing, setTailing] = useState(false);
  const [blockingIP, setBlockingIP] = useState<string | null>(null);

  const [filterEvent, setFilterEvent] = useState("");
  const [filterUser, setFilterUser] = useState("");
  const [filterFrom, setFilterFrom] = useState("");
  const [filterTo, setFilterTo] = useState("");
  const [limit, setLimit] = useState(100);

  const [appliedFilters, setAppliedFilters] = useState({
    event: "",
    user_id: "",
    from: "",
    to: "",
    limit: 100,
  });

  // Base fetch
  useEffect(() => {
    let cancelled = false;

    async function fetchData() {
      const token = getToken();
      if (!token) return;
      setLoading(true);
      try {
        const data = await getAudit(token, {
          event: appliedFilters.event || undefined,
          user_id: appliedFilters.user_id || undefined,
          from: appliedFilters.from || undefined,
          to: appliedFilters.to || undefined,
          limit: appliedFilters.limit,
        });
        if (!cancelled) {
          setEntries(data);
          setError("");
        }
      } catch (e: unknown) {
        if (!cancelled) setError(e instanceof Error ? e.message : "Failed to load audit log");
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    fetchData();
    return () => { cancelled = true; };
  }, [searchKey, appliedFilters]);

  // Real-time tail: poll every 5s for entries newer than the top-most id
  useEffect(() => {
    if (!tailing) return;
    let cancelled = false;
    const id = setInterval(async () => {
      const token = getToken();
      if (!token) return;
      const topId = entries[0]?.id;
      try {
        const fresh = await getAudit(token, {
          after_id: topId,
          limit: 50,
        });
        if (!cancelled && fresh.length > 0) {
          setEntries((prev) => [...fresh, ...prev]);
        }
      } catch { /* silent */ }
    }, 5_000);
    return () => { cancelled = true; clearInterval(id); };
  }, [tailing, entries]);

  function handleSearch() {
    setAppliedFilters({ event: filterEvent, user_id: filterUser, from: filterFrom, to: filterTo, limit });
    setSearchKey((k) => k + 1);
  }

  async function handleBlockIP(ip: string) {
    const token = getToken();
    if (!token) return;
    setBlockingIP(ip);
    try {
      await blockIP(token, ip, "Blocked from audit log");
      setError("");
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBlockingIP(null);
    }
  }

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "2rem" }}>
        <div>
          <h1 style={{ fontSize: "1.125rem", fontWeight: 600, color: "var(--text)", margin: 0 }}>Audit Log</h1>
          <p style={{ fontSize: "0.8125rem", color: "var(--muted)", margin: "0.25rem 0 0" }}>
            {entries.length} event{entries.length !== 1 ? "s" : ""}
            {tailing && <span style={{ color: "var(--accent)", marginLeft: "0.5rem" }}>· live</span>}
          </p>
        </div>
        <div style={{ display: "flex", gap: "0.5rem" }}>
          <button
            className={`btn ${tailing ? "btn-primary" : "btn-ghost"}`}
            onClick={() => setTailing((t) => !t)}
            title={tailing ? "Stop live tail" : "Start live tail"}
          >
            {tailing ? <Square size={13} /> : <Play size={13} />}
            {tailing ? "Stop" : "Tail"}
          </button>
          <button className="btn btn-ghost" onClick={() => setSearchKey((k) => k + 1)}>
            <RefreshCw size={14} />
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="card" style={{ marginBottom: "1rem", display: "flex", flexWrap: "wrap", gap: "0.75rem", alignItems: "flex-end" }}>
        <div style={{ flex: "1 1 9rem" }}>
          <label style={{ display: "block", fontSize: "0.75rem", color: "var(--muted)", marginBottom: "0.25rem" }}>Event type</label>
          <select className="input" value={filterEvent} onChange={(e) => setFilterEvent(e.target.value)}>
            {EVENTS.map((ev) => <option key={ev} value={ev}>{ev || "All events"}</option>)}
          </select>
        </div>
        <div style={{ flex: "1 1 9rem" }}>
          <label style={{ display: "block", fontSize: "0.75rem", color: "var(--muted)", marginBottom: "0.25rem" }}>User ID</label>
          <input className="input" value={filterUser} onChange={(e) => setFilterUser(e.target.value)} placeholder="uuid" />
        </div>
        <div style={{ flex: "1 1 9rem" }}>
          <label style={{ display: "block", fontSize: "0.75rem", color: "var(--muted)", marginBottom: "0.25rem" }}>From</label>
          <input className="input" type="datetime-local" value={filterFrom} onChange={(e) => setFilterFrom(e.target.value)} />
        </div>
        <div style={{ flex: "1 1 9rem" }}>
          <label style={{ display: "block", fontSize: "0.75rem", color: "var(--muted)", marginBottom: "0.25rem" }}>To</label>
          <input className="input" type="datetime-local" value={filterTo} onChange={(e) => setFilterTo(e.target.value)} />
        </div>
        <div style={{ width: "5rem" }}>
          <label style={{ display: "block", fontSize: "0.75rem", color: "var(--muted)", marginBottom: "0.25rem" }}>Limit</label>
          <input
            className="input"
            type="number"
            value={limit}
            min={1}
            max={1000}
            onChange={(e) => setLimit(Number(e.target.value))}
          />
        </div>
        <button className="btn btn-primary" onClick={handleSearch}>
          <Search size={14} /> Search
        </button>
      </div>

      {error && <p style={{ color: "var(--danger)", fontSize: "0.8125rem", marginBottom: "1rem" }}>{error}</p>}

      <div className="card" style={{ padding: 0, overflow: "hidden" }}>
        {loading ? (
          <p style={{ padding: "1.5rem", fontSize: "0.8125rem", color: "var(--muted)" }}>Loading…</p>
        ) : entries.length === 0 ? (
          <p style={{ padding: "1.5rem", fontSize: "0.8125rem", color: "var(--muted)" }}>No events found.</p>
        ) : (
          <div style={{ overflowX: "auto" }}>
            <table>
              <thead>
                <tr>
                  <th>Time</th>
                  <th>Event</th>
                  <th>Method</th>
                  <th>Path</th>
                  <th>Upstream</th>
                  <th>Status</th>
                  <th>IP</th>
                  <th>Detail</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {entries.map((e) => (
                  <tr key={e.id}>
                    <td className="mono" style={{ fontSize: "0.75rem", color: "var(--muted)", whiteSpace: "nowrap" }}>
                      {new Date(e.ts).toLocaleString()}
                    </td>
                    <td><EventBadge event={e.event} /></td>
                    <td style={{ fontSize: "0.75rem" }}>{e.method}</td>
                    <td className="mono" style={{ color: "var(--accent)" }}>{e.path}</td>
                    <td className="mono" style={{ fontSize: "0.8125rem" }}>{e.upstream}</td>
                    <td>
                      {e.status_code > 0 && (
                        <span className={`badge ${e.status_code >= 500 ? "badge-red" : e.status_code >= 400 ? "badge-yellow" : "badge-green"}`}>
                          {e.status_code}
                        </span>
                      )}
                    </td>
                    <td className="mono" style={{ fontSize: "0.75rem", color: "var(--muted)" }}>{e.ip}</td>
                    <td
                      style={{ fontSize: "0.75rem", color: "var(--muted)", maxWidth: "16rem", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}
                    >
                      {e.detail}
                    </td>
                    <td>
                      <button
                        className="btn btn-ghost"
                        style={{ padding: "0.2rem 0.4rem", fontSize: "0.7rem" }}
                        title={`Block ${e.ip}`}
                        disabled={blockingIP === e.ip}
                        onClick={() => handleBlockIP(e.ip)}
                      >
                        <ShieldOff size={11} />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
