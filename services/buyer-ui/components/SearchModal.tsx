"use client";

import { useState, useEffect, useRef } from "react";
import { useRouter } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import { Search, ChevronRight } from "lucide-react";

const TRENDING = [
  "Wireless earbuds",
  "Running shoes",
  "Skincare kit",
  "Gaming chair",
  "Coffee maker",
  "Mechanical keyboard",
];

const POPULAR_CATEGORIES = [
  { label: "Electronics", href: "/products?category=electronics" },
  { label: "Fashion", href: "/products?category=fashion" },
  { label: "Home & Kitchen", href: "/products?category=home-kitchen" },
  { label: "Sports & Fitness", href: "/products?category=sports" },
  { label: "Beauty & Personal Care", href: "/products?category=beauty" },
  { label: "Books & Stationery", href: "/products?category=books" },
];

interface SearchModalProps {
  open: boolean;
  onClose: () => void;
}

export function SearchModal({ open, onClose }: SearchModalProps) {
  const [query, setQuery] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);
  const router = useRouter();

  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 80);
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
      setQuery("");
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [open]);

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [onClose]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!query.trim()) return;
    router.push(`/products?search=${encodeURIComponent(query.trim())}`);
    onClose();
  };

  const handleChip = (term: string) => {
    router.push(`/products?search=${encodeURIComponent(term)}`);
    onClose();
  };

  const handleCategory = (href: string) => {
    router.push(href);
    onClose();
  };

  return (
    <AnimatePresence>
      {open && (
        <>
          {/* Backdrop */}
          <motion.div
            className="fixed inset-0 bg-black/40 z-50"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2, ease: "easeOut" }}
            onClick={onClose}
          />

          {/* Panel */}
          <motion.div
            className="fixed top-0 inset-x-0 z-50 max-w-2xl mx-auto mt-16 bg-white rounded-lg shadow-xl overflow-hidden"
            initial={{ opacity: 0, y: -8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -8 }}
            transition={{ duration: 0.25, ease: [0.25, 0.1, 0.25, 1] }}
          >
            {/* Search input row */}
            <form onSubmit={handleSubmit} className="relative flex items-center border-b border-[#E8E8E8]">
              <Search className="absolute left-4 h-4 w-4 text-[#999999] pointer-events-none" />
              <input
                ref={inputRef}
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Search for products, brands, categories..."
                className="h-12 w-full pl-10 pr-24 text-base font-medium text-[#111111] placeholder:text-[#999999] bg-white focus:outline-none"
              />
              <span className="absolute right-4 flex items-center gap-1 text-[10px] text-[#999999] border border-[#E8E8E8] rounded px-1 py-0.5 select-none">
                ESC
              </span>
            </form>

            {/* Below input — shown when no query */}
            {!query && (
              <div className="pb-4">
                {/* Trending searches */}
                <p className="px-4 pt-4 pb-2 text-xs font-semibold uppercase tracking-wide text-[#999999]">
                  Trending searches
                </p>
                <div className="flex flex-wrap gap-2 px-4 pb-4">
                  {TRENDING.map((term) => (
                    <button
                      key={term}
                      type="button"
                      onClick={() => handleChip(term)}
                      className="border border-[#E8E8E8] rounded-md px-3 py-1.5 text-sm text-[#555555] hover:border-[#111111] hover:text-[#111111] transition-colors duration-150 cursor-pointer"
                    >
                      {term}
                    </button>
                  ))}
                </div>

                {/* Popular categories */}
                <p className="px-4 pt-2 pb-2 text-xs font-semibold uppercase tracking-wide text-[#999999]">
                  Popular categories
                </p>
                <ul>
                  {POPULAR_CATEGORIES.map((cat) => (
                    <li key={cat.label}>
                      <button
                        type="button"
                        onClick={() => handleCategory(cat.href)}
                        className="w-full flex items-center justify-between px-4 py-2.5 text-sm text-[#555555] hover:bg-[#F6F6F6] hover:text-[#111111] transition-colors duration-150 cursor-pointer"
                      >
                        <span>{cat.label}</span>
                        <ChevronRight className="h-4 w-4 text-[#999999]" />
                      </button>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
}
