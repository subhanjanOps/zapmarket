"use client";
import { useEffect, useRef, useState, useCallback } from "react";
import { getAudit, blockIP, AuditEntry } from "@/lib/api";
import { RefreshCw, Search, Play, Square, ShieldOff } from "lucide-react";
import { SkeletonTableRows } from "@/app/components/Skeleton";

const EVENTS = ["", "AUTH_REJECTED", "RATE_LIMITED", "UPSTREAM_5XX", "CIRCUIT_OPEN", "ROUTE_CONFLICT"];
const ENTRY_CAP = 500;

function EventBadge({ event }: { event: string }) {
  const map: Record<string, string> = {
    AUTH_REJECTED: "badge badge-red",
    RATE_LIMITED:  "badge badge-yellow",
    UPSTREAM_5XX:  "badge badge-red",
    CIRCUIT_OPEN:  "badge badge-yellow",
    ROUTE_CONFLICT:"badge badge-blue",
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
  const tailFailsRef = useRef(0);

  const [filterEvent, setFilterEvent] = useState("");
  const [filterUser, setFilterUser] = useState("");
  const [filterFrom, setFilterFrom] = useState("");
  const [filterTo, setFilterTo] = useState("");
  const [limit, setLimit] = useState(100);

  const [appliedFilters, setAppliedFilters] = useState({
    event: "", user_id: "", from: "", to: "", limit: 100,
  });

  // Track the top entry ID in a ref so the tail effect never has a stale closure
  const topIdRef = useRef<number | undefined>(undefined);

  useEffect(() => {
    let cancelled = false;
    async function fetchData() {
      try {
        const data = await getAudit({
          event:   appliedFilters.event   || undefined,
          user_id: appliedFilters.user_id || undefined,
          from:    appliedFilters.from    || undefined,
          to:      appliedFilters.to      || undefined,
          limit:   appliedFilters.limit,
        });
        if (!cancelled) {
          setEntries(data);
          topIdRef.current = data[0]?.id;
          setError("");
          setLoading(false);
        }
      } catch (e: unknown) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : "Failed to load audit log");
          setLoading(false);
        }
      }
    }
    fetchData();
    return () => { cancelled = true; };
  }, [searchKey, appliedFilters]);

  const stopTailing = useCallback(() => setTailing(false), []);

  // Live tail: polls every 5s; reads topIdRef to avoid stale closure
  useEffect(() => {
    if (!tailing) return;
    tailFailsRef.current = 0;
    let cancelled = false;
    const id = setInterval(async () => {
      try {
        const fresh = await getAudit({ after_id: topIdRef.current, limit: 50 });
        if (!cancelled && fresh.length > 0) {
          tailFailsRef.current = 0;
          topIdRef.current = fresh[0].id;
          setEntries((prev) => [...fresh, ...prev].slice(0, ENTRY_CAP));
        }
      } catch (e: unknown) {
        if (!cancelled) {
          tailFailsRef.current += 1;
          if (tailFailsRef.current >= 3) {
            clearInterval(id);
            stopTailing();
            setError(e instanceof Error ? e.message : "Live tail failed — stopped after 3 errors");
          }
        }
      }
    }, 5_000);
    return () => { cancelled = true; clearInterval(id); };
  }, [tailing, stopTailing]); // intentionally excludes `entries` — topIdRef carries the latest value

  function handleSearch() {
    setAppliedFilters({ event: filterEvent, user_id: filterUser, from: filterFrom, to: filterTo, limit });
    setSearchKey((k) => k + 1);
  }

  async function handleBlockIP(ip: string) {
    setBlockingIP(ip);
    try {
      await blockIP(ip, "Blocked from audit log");
      setError("");
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBlockingIP(null);
    }
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">Audit Log</h1>
          <p className="page-subtitle">
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
          <label className="form-label">Event type</label>
          <select className="input" value={filterEvent} onChange={(e) => setFilterEvent(e.target.value)}>
            {EVENTS.map((ev) => <option key={ev} value={ev}>{ev || "All events"}</option>)}
          </select>
        </div>
        <div style={{ flex: "1 1 9rem" }}>
          <label className="form-label">User ID</label>
          <input className="input" value={filterUser} onChange={(e) => setFilterUser(e.target.value)} placeholder="uuid" />
        </div>
        <div style={{ flex: "1 1 9rem" }}>
          <label className="form-label">From</label>
          <input className="input" type="datetime-local" value={filterFrom} onChange={(e) => setFilterFrom(e.target.value)} />
        </div>
        <div style={{ flex: "1 1 9rem" }}>
          <label className="form-label">To</label>
          <input className="input" type="datetime-local" value={filterTo} onChange={(e) => setFilterTo(e.target.value)} />
        </div>
        <div style={{ width: "5rem" }}>
          <label className="form-label">Limit</label>
          <input className="input" type="number" value={limit} min={1} max={100} onChange={(e) => setLimit(Number(e.target.value))} />
        </div>
        <button className="btn btn-primary" onClick={handleSearch}>
          <Search size={14} /> Search
        </button>
      </div>

      {error && <p style={{ color: "var(--danger)", fontSize: "0.8125rem", marginBottom: "1rem" }}>{error}</p>}

      <div className="card" style={{ padding: 0, overflow: "hidden" }}>
        {loading ? (
          <table>
            <thead>
              <tr>{["Time","Event","Method","Path","Upstream","Status","IP","Detail",""].map((h, i) => <th key={i}>{h}</th>)}</tr>
            </thead>
            <tbody><SkeletonTableRows cols={9} rows={8} /></tbody>
          </table>
        ) : entries.length === 0 ? (
          <div className="empty-state">
            <p className="empty-state-title">No events found</p>
            <p className="empty-state-body">Adjust filters or wait for traffic to flow through the gateway</p>
          </div>
        ) : (
          <div style={{ overflowX: "auto" }}>
            <table>
              <thead>
                <tr>
                  <th>Time</th><th>Event</th><th>Method</th><th>Path</th>
                  <th>Upstream</th><th>Status</th><th>IP</th><th>Detail</th><th></th>
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
                    <td style={{ fontSize: "0.75rem", color: "var(--muted)", maxWidth: "16rem", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
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
