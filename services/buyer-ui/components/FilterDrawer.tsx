"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { SlidersHorizontal, X } from "lucide-react";
import { cn } from "@/lib/utils";

interface Category {
  id: string;
  name: string;
}

interface FilterDrawerProps {
  categories: Category[];
  selectedCategory?: string;
  selectedSort?: string;
}

const SORT_OPTIONS = [
  { value: "",             label: "Relevance" },
  { value: "created_at_desc", label: "Newest first" },
  { value: "price_asc",    label: "Price: Low to High" },
  { value: "price_desc",   label: "Price: High to Low" },
];

export function FilterDrawer({
  categories,
  selectedCategory,
  selectedSort,
}: FilterDrawerProps) {
  const router = useRouter();
  const params = useSearchParams();

  const applyFilter = (key: string, value: string) => {
    const p = new URLSearchParams(params.toString());
    if (value) {
      p.set(key, value);
    } else {
      p.delete(key);
    }
    p.delete("page");
    router.push(`/products?${p.toString()}`);
  };

  const hasActiveFilters = !!(selectedCategory || selectedSort);

  return (
    <Sheet>
      <SheetTrigger asChild>
        <button className="flex items-center gap-2 h-9 px-4 bg-white border border-[#EDE9E3] rounded-xl text-sm font-semibold text-[#3D2E1A] hover:border-[#E91E8C] hover:text-[#E91E8C] transition-colors shadow-sm">
          <SlidersHorizontal className="h-4 w-4" />
          Filters
          {hasActiveFilters && (
            <span className="h-1.5 w-1.5 rounded-full bg-[#E91E8C]" />
          )}
        </button>
      </SheetTrigger>
      <SheetContent side="left" className="w-72 p-0 border-r border-[#EDE9E3]">
        <SheetHeader className="px-5 py-4 border-b border-[#EDE9E3]">
          <SheetTitle className="text-base font-bold text-[#0F0A04] text-left">
            Filters
          </SheetTitle>
        </SheetHeader>

        <div className="overflow-auto p-5 space-y-6">
          {/* Sort */}
          <div>
            <p className="text-xs font-bold tracking-wider text-[#B8A898] uppercase mb-3">
              Sort by
            </p>
            <div className="space-y-0.5">
              {SORT_OPTIONS.map(({ value, label }) => (
                <button
                  key={value}
                  onClick={() => applyFilter("sort", value)}
                  className={cn(
                    "w-full text-left px-3 py-2.5 rounded-xl text-sm font-medium transition-colors",
                    (selectedSort === value) || (!selectedSort && !value)
                      ? "bg-[#FDE8F4] text-[#E91E8C]"
                      : "text-[#3D2E1A] hover:bg-[#F9F8F5]"
                  )}
                >
                  {label}
                </button>
              ))}
            </div>
          </div>

          {/* Categories */}
          {categories.length > 0 && (
            <div>
              <p className="text-xs font-bold tracking-wider text-[#B8A898] uppercase mb-3">
                Category
              </p>
              <div className="space-y-0.5">
                <button
                  onClick={() => applyFilter("category", "")}
                  className={cn(
                    "w-full text-left px-3 py-2.5 rounded-xl text-sm font-medium transition-colors",
                    !selectedCategory
                      ? "bg-[#FDE8F4] text-[#E91E8C]"
                      : "text-[#3D2E1A] hover:bg-[#F9F8F5]"
                  )}
                >
                  All categories
                </button>
                {categories.map(cat => (
                  <button
                    key={cat.id}
                    onClick={() => applyFilter("category", cat.name)}
                    className={cn(
                      "w-full text-left px-3 py-2.5 rounded-xl text-sm font-medium transition-colors",
                      selectedCategory === cat.name
                        ? "bg-[#FDE8F4] text-[#E91E8C]"
                        : "text-[#3D2E1A] hover:bg-[#F9F8F5]"
                    )}
                  >
                    {cat.name}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Clear */}
          {hasActiveFilters && (
            <button
              onClick={() => router.push("/products")}
              className="w-full flex items-center justify-center gap-2 h-9 border border-[#EDE9E3] rounded-xl text-sm font-semibold text-[#7A6856] hover:border-[#E91E8C] hover:text-[#E91E8C] transition-colors"
            >
              <X className="h-4 w-4" />
              Clear all filters
            </button>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}
