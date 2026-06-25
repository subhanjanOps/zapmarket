function SkeletonCard() {
  return (
    <div className="rounded-2xl overflow-hidden animate-pulse" style={{ background: "#fff", border: "1px solid #F0EDE8" }}>
      <div className="h-1 w-full" style={{ background: "#F0EDE8" }} />
      <div className="aspect-square" style={{ background: "#F5F3EF" }} />
      <div className="p-3 space-y-2">
        <div className="h-3 rounded-full w-3/4" style={{ background: "#F0EDE8" }} />
        <div className="h-3 rounded-full w-1/2" style={{ background: "#F0EDE8" }} />
        <div className="h-4 rounded-full w-1/3 mt-2" style={{ background: "#FFE8F0" }} />
      </div>
    </div>
  );
}

export default function ProductsLoading() {
  return (
    <div className="max-w-7xl mx-auto px-4 py-6">
      <div className="flex gap-6">
        {/* Sidebar skeleton */}
        <div className="hidden lg:block w-56 shrink-0">
          <div className="rounded-2xl p-4 space-y-3 animate-pulse" style={{ background: "#fff", border: "1px solid #F0EDE8" }}>
            <div className="h-4 rounded-full w-2/3" style={{ background: "#F0EDE8" }} />
            {[...Array(6)].map((_, i) => (
              <div key={i} className="h-3 rounded-full" style={{ background: "#F5F3EF", width: `${55 + (i % 3) * 15}%` }} />
            ))}
          </div>
        </div>
        {/* Grid skeleton */}
        <div className="flex-1 grid grid-cols-2 sm:grid-cols-3 xl:grid-cols-4 gap-4">
          {[...Array(8)].map((_, i) => <SkeletonCard key={i} />)}
        </div>
      </div>
    </div>
  );
}
