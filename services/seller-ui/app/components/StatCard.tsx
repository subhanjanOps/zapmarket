export function StatCard({
  label,
  value,
  sub,
  accent = false,
}: {
  label: string;
  value: string | number;
  sub?: string;
  accent?: boolean;
}) {
  return (
    <div className="card" style={{ display: "flex", flexDirection: "column", gap: "0.375rem" }}>
      <div style={{ fontSize: "0.75rem", color: "var(--muted)", fontWeight: 500 }}>{label}</div>
      <div style={{
        fontSize: "1.75rem",
        fontWeight: 700,
        lineHeight: 1.1,
        color: accent ? "var(--accent)" : "var(--text)",
        letterSpacing: "-0.02em",
      }}>
        {value}
      </div>
      {sub && <div style={{ fontSize: "0.75rem", color: "var(--muted)" }}>{sub}</div>}
    </div>
  );
}
