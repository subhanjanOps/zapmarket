"use client";

import { useState, useEffect, useRef } from "react";
import { useRouter } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import { Search, X, TrendingUp, Clock, ArrowRight } from "lucide-react";

const TRENDING = [
  "Wireless earbuds",
  "Running shoes",
  "Skincare kit",
  "Gaming chair",
  "Coffee maker",
];

const POPULAR_CATEGORIES = [
  "Electronics",
  "Fashion",
  "Home & Kitchen",
  "Sports",
  "Beauty",
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
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [onClose]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!query.trim()) return;
    router.push(`/products?search=${encodeURIComponent(query.trim())}`);
    onClose();
  };

  const handleSuggestion = (term: string) => {
    router.push(`/products?search=${encodeURIComponent(term)}`);
    onClose();
  };

  return (
    <AnimatePresence>
      {open && (
        <>
          <motion.div
            className="fixed inset-0 bg-black/40 backdrop-blur-sm z-50"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
            onClick={onClose}
          />
          <motion.div
            className="fixed top-0 left-0 right-0 z-50 bg-white shadow-[0_24px_64px_rgba(15,10,4,0.18)]"
            initial={{ y: -16, opacity: 0 }}
            animate={{ y: 0, opacity: 1 }}
            exit={{ y: -16, opacity: 0 }}
            transition={{ duration: 0.25, ease: [0.16, 1, 0.3, 1] }}
          >
            <div className="container-zap py-4">
              <form onSubmit={handleSubmit} className="relative">
                <Search className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-[#7A6856]" />
                <input
                  ref={inputRef}
                  value={query}
                  onChange={e => setQuery(e.target.value)}
                  placeholder="Search for products, brands, categories..."
                  className="w-full h-14 pl-12 pr-16 bg-[#F9F8F5] rounded-2xl text-base font-medium text-[#0F0A04] placeholder-[#B8A898] border-2 border-transparent focus:border-[#E91E8C] focus:bg-white outline-none transition-all"
                />
                <button
                  type="button"
                  onClick={onClose}
                  className="absolute right-3 top-1/2 -translate-y-1/2 p-2 rounded-xl hover:bg-[#F3F0EB] transition-colors"
                  aria-label="Close search"
                >
                  <X className="h-5 w-5 text-[#7A6856]" />
                </button>
              </form>

              <div className="mt-6 pb-6 grid sm:grid-cols-2 gap-6">
                <div>
                  <div className="flex items-center gap-2 mb-3">
                    <TrendingUp className="h-4 w-4 text-[#E91E8C]" />
                    <span className="text-xs font-bold tracking-wider text-[#7A6856] uppercase">
                      Trending
                    </span>
                  </div>
                  <ul className="space-y-0.5">
                    {TRENDING.map(term => (
                      <li key={term}>
                        <button
                          onClick={() => handleSuggestion(term)}
                          className="w-full text-left px-3 py-2 rounded-xl text-sm font-medium text-[#3D2E1A] hover:bg-[#F9F8F5] hover:text-[#E91E8C] transition-colors flex items-center justify-between group"
                        >
                          {term}
                          <ArrowRight className="h-3.5 w-3.5 opacity-0 group-hover:opacity-100 transition-opacity" />
                        </button>
                      </li>
                    ))}
                  </ul>
                </div>

                <div>
                  <div className="flex items-center gap-2 mb-3">
                    <Clock className="h-4 w-4 text-[#7A6856]" />
                    <span className="text-xs font-bold tracking-wider text-[#7A6856] uppercase">
                      Popular Categories
                    </span>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {POPULAR_CATEGORIES.map(cat => (
                      <button
                        key={cat}
                        onClick={() => handleSuggestion(cat)}
                        className="px-3.5 py-1.5 bg-[#F9F8F5] hover:bg-[#FDE8F4] hover:text-[#E91E8C] text-sm font-medium text-[#3D2E1A] rounded-full transition-colors border border-[#EDE9E3]"
                      >
                        {cat}
                      </button>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
}
