export default function ProductDetailLoading() {
  return (
    <div className="max-w-7xl mx-auto px-4 py-8 animate-pulse">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 mb-6">
        {[...Array(3)].map((_, i) => (
          <div key={i} className="h-3 rounded-full" style={{ background: "#F0EDE8", width: i === 2 ? "120px" : "64px" }} />
        ))}
      </div>

      <div className="grid lg:grid-cols-2 gap-8 lg:gap-12">
        {/* Gallery skeleton */}
        <div className="space-y-3">
          <div className="aspect-square rounded-3xl" style={{ background: "#F5F3EF" }} />
          <div className="flex gap-2">
            {[...Array(4)].map((_, i) => (
              <div key={i} className="w-16 h-16 rounded-xl" style={{ background: "#F0EDE8" }} />
            ))}
          </div>
        </div>

        {/* Info skeleton */}
        <div className="space-y-5">
          <div className="h-8 rounded-full w-2/3" style={{ background: "#F0EDE8" }} />
          <div className="h-10 rounded-full w-1/3" style={{ background: "#FFE8F0" }} />
          <div className="h-px w-full" style={{ background: "#F0EDE8" }} />
          <div className="space-y-2">
            {[...Array(4)].map((_, i) => (
              <div key={i} className="h-3 rounded-full" style={{ background: "#F5F3EF", width: `${70 + (i % 2) * 20}%` }} />
            ))}
          </div>
          <div className="h-12 rounded-xl w-full" style={{ background: "#FFE8F0" }} />
        </div>
      </div>
    </div>
  );
}
