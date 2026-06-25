import { ProductCardSkeleton } from "@/components/ProductCardSkeleton";

export default function ProductsLoading() {
  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      <div className="flex gap-7">
        {/* Sidebar skeleton — matches md:block sidebar in page.tsx */}
        <div className="hidden md:block w-56 shrink-0">
          <div className="rounded-2xl border border-border p-5 space-y-5">
            {/* Categories label */}
            <div className="skeleton h-4 w-24 rounded-full" />
            {/* Category buttons */}
            <div className="space-y-1">
              {Array.from({ length: 7 }).map((_, i) => (
                <div key={i} className="skeleton h-9 rounded-xl" />
              ))}
            </div>
            {/* Divider */}
            <div className="skeleton h-px w-full" />
            {/* Sort label */}
            <div className="skeleton h-4 w-16 rounded-full" />
            {/* Sort buttons */}
            <div className="space-y-1">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="skeleton h-9 rounded-xl" />
              ))}
            </div>
          </div>
        </div>

        {/* Main content skeleton */}
        <div className="flex-1 min-w-0">
          {/* Header row */}
          <div className="flex items-center gap-3 mb-5">
            <div className="skeleton h-8 w-32 rounded-full" />
            <div className="skeleton h-6 w-20 rounded-full" />
          </div>

          {/* Product grid — matches grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4 */}
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
            {Array.from({ length: 12 }).map((_, i) => (
              <ProductCardSkeleton key={i} />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
