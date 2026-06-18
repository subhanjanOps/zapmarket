const MAP: Record<string, string> = {
  ACTIVE:    "badge-green",
  APPROVED:  "badge-green",
  DRAFT:     "badge-yellow",
  PENDING:   "badge-yellow",
  ARCHIVED:  "badge-gray",
  SUSPENDED: "badge-red",
  INACTIVE:  "badge-gray",
};

export default function StatusBadge({ status }: { status: string }) {
  const cls = MAP[status?.toUpperCase()] ?? "badge-gray";
  return <span className={`badge ${cls}`}>{status}</span>;
}
