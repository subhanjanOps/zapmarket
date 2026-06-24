const MAP: Record<string, { cls: string; pulse?: boolean }> = {
  ACTIVE:    { cls: "badge-green",  pulse: true  },
  CONFIRMED: { cls: "badge-green",  pulse: true  },
  DONE:      { cls: "badge-green"               },
  DRAFT:     { cls: "badge-yellow"              },
  PENDING:   { cls: "badge-yellow"              },
  RESERVED:  { cls: "badge-blue"               },
  ARCHIVED:  { cls: "badge-gray"               },
  CANCELLED: { cls: "badge-red"                },
  FAILED:    { cls: "badge-red"                },
  INACTIVE:  { cls: "badge-gray"               },
};

export function StatusBadge({ status }: { status: string }) {
  const entry = MAP[status?.toUpperCase()] ?? { cls: "badge-gray" };
  return (
    <span className={`badge ${entry.cls}`} style={{ display: "inline-flex", alignItems: "center", gap: "0.3rem" }}>
      {entry.pulse && (
        <span
          className="status-dot status-dot-green status-dot-pulse"
          style={{ width: 5, height: 5 }}
        />
      )}
      {status}
    </span>
  );
}
