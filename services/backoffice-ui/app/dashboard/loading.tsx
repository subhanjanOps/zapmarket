export default function Loading() {
  return (
    <div style={{ padding: "2rem" }}>
      <div className="skeleton" style={{ height: 28, width: 200, marginBottom: "1.75rem" }} />
      <div style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: "1rem", marginBottom: "1.75rem" }}>
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="card" style={{ padding: "1.25rem" }}>
            <div className="skeleton" style={{ height: 12, width: 80, marginBottom: "0.75rem" }} />
            <div className="skeleton" style={{ height: 28, width: 60 }} />
          </div>
        ))}
      </div>
    </div>
  );
}
