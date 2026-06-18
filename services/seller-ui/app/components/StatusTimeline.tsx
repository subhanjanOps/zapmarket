const STEPS = ["PENDING", "RESERVED", "CONFIRMED"] as const;

export function StatusTimeline({ status }: { status: string }) {
  const cancelled = status === "CANCELLED";
  const activeIdx = cancelled ? -1 : STEPS.indexOf(status as typeof STEPS[number]);

  if (cancelled) {
    return (
      <div style={{ display: "flex", alignItems: "center", gap: "0.625rem", padding: "0.75rem 1rem", background: "color-mix(in srgb, var(--danger) 8%, transparent)", borderRadius: 8 }}>
        <span className="badge badge-red">CANCELLED</span>
        <span style={{ fontSize: "0.8125rem", color: "var(--muted)" }}>This order has been cancelled.</span>
      </div>
    );
  }

  return (
    <div className="timeline">
      {STEPS.map((step, i) => {
        const done   = i < activeIdx;
        const active = i === activeIdx;
        return (
          <div key={step} className={`timeline-step${done ? " done" : ""}`}>
            <div className={`timeline-dot${active ? " active" : done ? " done" : ""}`}>
              {done ? "✓" : i + 1}
            </div>
            <span className={`timeline-label${active ? " active" : done ? " done" : ""}`}>{step}</span>
          </div>
        );
      })}
    </div>
  );
}
