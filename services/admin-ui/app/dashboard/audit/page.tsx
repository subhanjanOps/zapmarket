"use client";
import { useEffect, useState } from "react";
import { getToken } from "@/lib/auth";
import { getAudit, AuditEntry } from "@/lib/api";
import { RefreshCw, Search } from "lucide-react";

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

  const [filterEvent, setFilterEvent] = useState("");
  const [filterUser, setFilterUser] = useState("");
  const [filterFrom, setFilterFrom] = useState("");
  const [filterTo, setFilterTo] = useState("");
  const [limit, setLimit] = useState(100);

  // Snapshot of filters at search time — avoids mid-fetch state changes
  const [appliedFilters, setAppliedFilters] = useState({
    event: "",
    user_id: "",
    from: "",
    to: "",
    limit: 100,
  });

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

  function handleSearch() {
    setAppliedFilters({ event: filterEvent, user_id: filterUser, from: filterFrom, to: filterTo, limit });
    setSearchKey((k) => k + 1);
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-semibold">Audit Log</h1>
          <p className="text-sm mt-1" style={{ color: "var(--muted)" }}>
            {entries.length} event{entries.length !== 1 ? "s" : ""}
          </p>
        </div>
        <button className="btn btn-ghost" onClick={() => setSearchKey((k) => k + 1)}>
          <RefreshCw size={14} />
        </button>
      </div>

      {/* Filters */}
      <div className="card mb-4 flex flex-wrap gap-3 items-end">
        <div className="flex-1 min-w-36">
          <label className="block text-xs mb-1" style={{ color: "var(--muted)" }}>Event type</label>
          <select className="input" value={filterEvent} onChange={(e) => setFilterEvent(e.target.value)}>
            {EVENTS.map((ev) => <option key={ev} value={ev}>{ev || "All events"}</option>)}
          </select>
        </div>
        <div className="flex-1 min-w-36">
          <label className="block text-xs mb-1" style={{ color: "var(--muted)" }}>User ID</label>
          <input className="input" value={filterUser} onChange={(e) => setFilterUser(e.target.value)} placeholder="uuid" />
        </div>
        <div className="flex-1 min-w-36">
          <label className="block text-xs mb-1" style={{ color: "var(--muted)" }}>From</label>
          <input className="input" type="datetime-local" value={filterFrom} onChange={(e) => setFilterFrom(e.target.value)} />
        </div>
        <div className="flex-1 min-w-36">
          <label className="block text-xs mb-1" style={{ color: "var(--muted)" }}>To</label>
          <input className="input" type="datetime-local" value={filterTo} onChange={(e) => setFilterTo(e.target.value)} />
        </div>
        <div style={{ width: "5rem" }}>
          <label className="block text-xs mb-1" style={{ color: "var(--muted)" }}>Limit</label>
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

      {error && <p className="mb-4 text-sm" style={{ color: "var(--danger)" }}>{error}</p>}

      <div className="card p-0 overflow-hidden">
        {loading ? (
          <p className="p-6 text-sm" style={{ color: "var(--muted)" }}>Loading…</p>
        ) : entries.length === 0 ? (
          <p className="p-6 text-sm" style={{ color: "var(--muted)" }}>No events found.</p>
        ) : (
          <div className="overflow-x-auto">
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
                </tr>
              </thead>
              <tbody>
                {entries.map((e) => (
                  <tr key={e.id}>
                    <td className="text-xs whitespace-nowrap" style={{ color: "var(--muted)" }}>
                      {new Date(e.ts).toLocaleString()}
                    </td>
                    <td><EventBadge event={e.event} /></td>
                    <td className="text-xs">{e.method}</td>
                    <td><span className="mono" style={{ fontSize: "0.8125rem", color: "var(--accent-hover)" }}>{e.path}</span></td>
                    <td className="mono" style={{ fontSize: "0.8125rem" }}>{e.upstream}</td>
                    <td>
                      {e.status_code > 0 && (
                        <span className={`badge ${e.status_code >= 500 ? "badge-red" : e.status_code >= 400 ? "badge-yellow" : "badge-green"}`}>
                          {e.status_code}
                        </span>
                      )}
                    </td>
                    <td className="text-xs" style={{ color: "var(--muted)" }}>{e.ip}</td>
                    <td
                      className="text-xs"
                      style={{ color: "var(--muted)", maxWidth: "16rem", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}
                    >
                      {e.detail}
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
