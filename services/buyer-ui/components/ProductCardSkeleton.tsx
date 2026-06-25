export function ProductCardSkeleton() {
  return (
    <div className="bg-white rounded-2xl overflow-hidden shadow-[0_1px_3px_rgba(15,10,4,0.06)]">
      <div className="skeleton" style={{ aspectRatio: "4/5" }} />
      <div className="p-4 space-y-2.5">
        <div className="skeleton h-3 w-1/3 rounded-full" />
        <div className="skeleton h-4 w-4/5 rounded-full" />
        <div className="skeleton h-4 w-2/3 rounded-full" />
        <div className="flex items-center gap-2 pt-1">
          <div className="skeleton h-5 w-16 rounded-full" />
          <div className="skeleton h-4 w-12 rounded-full" />
        </div>
        <div className="skeleton h-9 w-full rounded-xl mt-1" />
      </div>
    </div>
  );
}
