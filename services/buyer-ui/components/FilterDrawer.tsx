"use client";

import { AnimatePresence, motion } from "framer-motion";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";

interface FilterDrawerProps {
  open: boolean;
  onClose: () => void;
  categories: { id: string; name: string }[];
  activeCategoryId?: string;
  activeSort?: string;
  onCategoryChange: (id: string | undefined) => void;
  onSortChange: (sort: string) => void;
}

const SORT_OPTIONS = [
  { value: "newest",     label: "Newest" },
  { value: "price_asc",  label: "Price: Low to High" },
  { value: "price_desc", label: "Price: High to Low" },
];

export function FilterDrawer({
  open,
  onClose,
  categories,
  activeCategoryId,
  activeSort,
  onCategoryChange,
  onSortChange,
}: FilterDrawerProps) {
  const handleClear = () => {
    onCategoryChange(undefined);
    onSortChange("");
    onClose();
  };

  const handleApply = () => {
    onClose();
  };

  return (
    <AnimatePresence>
      {open && (
        <>
          {/* Backdrop */}
          <motion.div
            key="backdrop"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="fixed inset-0 bg-black/40 backdrop-blur-sm z-50"
            onClick={onClose}
          />

          {/* Drawer */}
          <motion.div
            key="drawer"
            initial={{ x: "100%" }}
            animate={{ x: 0 }}
            exit={{ x: "100%" }}
            transition={{ type: "spring", stiffness: 340, damping: 34 }}
            className="fixed right-0 top-0 bottom-0 w-80 bg-white z-50 flex flex-col"
            style={{ boxShadow: "-24px 0 64px rgba(15,10,4,0.12)" }}
          >
            {/* Header */}
            <div className="flex items-center justify-between px-5 py-4 border-b border-[#EDE9E3] shrink-0">
              <span className="font-display font-bold text-[#0F0A04] text-base tracking-tight">
                Filters
              </span>
              <button
                onClick={onClose}
                className="h-8 w-8 flex items-center justify-center rounded-full text-[#7A6856] hover:bg-[#F9F8F5] hover:text-[#3D2E1A] transition-colors"
                aria-label="Close filters"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            {/* Scrollable body */}
            <div className="flex-1 overflow-y-auto px-5 py-6 space-y-7">
              {/* Categories */}
              <section>
                <p className="text-xs font-bold uppercase tracking-wider text-[#7A6856] mb-3">
                  Category
                </p>
                <div className="flex flex-wrap gap-2">
                  {/* All pill */}
                  <button
                    onClick={() => onCategoryChange(undefined)}
                    className={cn(
                      "px-3.5 py-1.5 rounded-full text-sm font-medium border transition-colors cursor-pointer",
                      !activeCategoryId
                        ? "bg-[#E91E8C] text-white border-[#E91E8C]"
                        : "bg-white text-[#3D2E1A] border-[#EDE9E3] hover:border-[#E91E8C]"
                    )}
                  >
                    All
                  </button>

                  {categories.map((cat) => (
                    <button
                      key={cat.id}
                      onClick={() => onCategoryChange(cat.id)}
                      className={cn(
                        "px-3.5 py-1.5 rounded-full text-sm font-medium border transition-colors cursor-pointer",
                        activeCategoryId === cat.id
                          ? "bg-[#E91E8C] text-white border-[#E91E8C]"
                          : "bg-white text-[#3D2E1A] border-[#EDE9E3] hover:border-[#E91E8C]"
                      )}
                    >
                      {cat.name}
                    </button>
                  ))}
                </div>
              </section>

              {/* Sort */}
              <section>
                <p className="text-xs font-bold uppercase tracking-wider text-[#7A6856] mb-3">
                  Sort by
                </p>
                <div className="space-y-1">
                  {SORT_OPTIONS.map(({ value, label }) => {
                    const isActive = activeSort === value;
                    return (
                      <div
                        key={value}
                        onClick={() => onSortChange(value)}
                        className="flex items-center gap-3 px-1 py-2.5 cursor-pointer select-none"
                      >
                        {/* Radio indicator */}
                        <span
                          className={cn(
                            "h-4 w-4 rounded-full border-2 flex items-center justify-center shrink-0 transition-colors",
                            isActive
                              ? "border-[#E91E8C]"
                              : "border-[#C9BEB2]"
                          )}
                        >
                          {isActive && (
                            <span className="h-2 w-2 rounded-full bg-[#E91E8C]" />
                          )}
                        </span>
                        <span
                          className={cn(
                            "text-sm font-medium transition-colors",
                            isActive ? "text-[#E91E8C]" : "text-[#3D2E1A]"
                          )}
                        >
                          {label}
                        </span>
                      </div>
                    );
                  })}
                </div>
              </section>

              {/* Price Range (static/visual) */}
              <section>
                <p className="text-xs font-bold uppercase tracking-wider text-[#7A6856] mb-3">
                  Price Range
                </p>
                <div className="flex items-center gap-2">
                  <input
                    type="number"
                    placeholder="From"
                    className="flex-1 border border-[#EDE9E3] rounded-xl h-10 px-3 text-sm text-[#3D2E1A] placeholder:text-[#C9BEB2] focus:outline-none focus:border-[#E91E8C] transition-colors"
                  />
                  <span className="text-[#C9BEB2] font-medium text-sm shrink-0">—</span>
                  <input
                    type="number"
                    placeholder="To"
                    className="flex-1 border border-[#EDE9E3] rounded-xl h-10 px-3 text-sm text-[#3D2E1A] placeholder:text-[#C9BEB2] focus:outline-none focus:border-[#E91E8C] transition-colors"
                  />
                </div>
              </section>
            </div>

            {/* Footer */}
            <div className="shrink-0 sticky bottom-0 border-t border-[#EDE9E3] bg-white px-5 py-4 flex gap-3">
              <button
                onClick={handleClear}
                className="flex-1 h-11 rounded-2xl border border-[#EDE9E3] text-sm font-semibold text-[#7A6856] hover:border-[#E91E8C] hover:text-[#E91E8C] transition-colors"
              >
                Clear all
              </button>
              <motion.button
                onClick={handleApply}
                whileTap={{ scale: 0.97 }}
                className="flex-1 h-11 rounded-2xl bg-[#E91E8C] text-white text-sm font-semibold hover:bg-[#D01A7D] transition-colors"
              >
                Apply
              </motion.button>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
}
