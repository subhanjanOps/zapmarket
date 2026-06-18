const MAP: Record<string, string> = {
  ACTIVE:    "badge-green",
  CONFIRMED: "badge-green",
  DONE:      "badge-green",
  DRAFT:     "badge-yellow",
  PENDING:   "badge-yellow",
  RESERVED:  "badge-blue",
  ARCHIVED:  "badge-gray",
  CANCELLED: "badge-red",
  FAILED:    "badge-red",
  INACTIVE:  "badge-gray",
};

export function StatusBadge({ status }: { status: string }) {
  const cls = MAP[status?.toUpperCase()] ?? "badge-gray";
  return <span className={`badge ${cls}`}>{status}</span>;
}
