export function ProductCardSkeleton() {
  return (
    <div className="bg-white border border-[#E8E8E8] rounded-lg overflow-hidden">
      {/* Image area */}
      <div className="aspect-square bg-[#F6F6F6] animate-pulse" />

      {/* Body */}
      <div className="px-3 pt-3 pb-3.5">
        {/* Category */}
        <div className="h-3 w-16 bg-[#F0F0F0] rounded animate-pulse" />

        {/* Name line 1 */}
        <div className="h-4 w-full bg-[#F0F0F0] rounded animate-pulse mt-2" />

        {/* Name line 2 */}
        <div className="h-4 w-2/3 bg-[#F0F0F0] rounded animate-pulse mt-1.5" />

        {/* Price */}
        <div className="h-5 w-20 bg-[#F0F0F0] rounded animate-pulse mt-2.5" />

        {/* Button */}
        <div className="h-8 w-full bg-[#F0F0F0] rounded-md animate-pulse mt-2.5" />
      </div>
    </div>
  );
}
