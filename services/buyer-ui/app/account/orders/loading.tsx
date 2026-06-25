export default function OrdersLoading() {
  return (
    <div className="max-w-3xl mx-auto px-4 py-10 animate-pulse">
      {/* Header */}
      <div className="flex items-center gap-3 mb-8">
        <div className="w-10 h-10 rounded-2xl" style={{ background: "#F0EDE8" }} />
        <div className="space-y-1.5">
          <div className="h-5 rounded-full w-32" style={{ background: "#F0EDE8" }} />
          <div className="h-3 rounded-full w-20" style={{ background: "#F5F3EF" }} />
        </div>
      </div>
      {/* Order cards */}
      <div className="space-y-3">
        {[...Array(4)].map((_, i) => (
          <div key={i} className="flex items-center justify-between rounded-2xl p-5" style={{ background: "#fff", border: "1px solid #F0EDE8" }}>
            <div className="space-y-2">
              <div className="h-2.5 rounded-full w-16" style={{ background: "#F0EDE8" }} />
              <div className="h-4 rounded-full w-28" style={{ background: "#F0EDE8" }} />
              <div className="h-5 rounded-full w-20" style={{ background: "#FFE8F0" }} />
            </div>
            <div className="flex items-center gap-3">
              <div className="h-6 rounded-full w-20" style={{ background: "#F0EDE8" }} />
              <div className="w-4 h-4 rounded-full" style={{ background: "#F0EDE8" }} />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
