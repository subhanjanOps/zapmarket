"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

interface RecentlyViewedItem {
  product_id: string;
  name: string;
  image: string;
  timestamp: number;
}

const STORAGE_KEY = "recently_viewed";
const MAX_ITEMS = 10;

export function trackRecentlyViewed(item: Omit<RecentlyViewedItem, "timestamp">) {
  if (typeof window === "undefined") return;
  const raw = localStorage.getItem(STORAGE_KEY);
  const existing: RecentlyViewedItem[] = raw ? JSON.parse(raw) : [];
  const filtered = existing.filter((i) => i.product_id !== item.product_id);
  const updated = [{ ...item, timestamp: Date.now() }, ...filtered].slice(0, MAX_ITEMS);
  localStorage.setItem(STORAGE_KEY, JSON.stringify(updated));
}

export function RecentlyViewed() {
  const [items, setItems] = useState<RecentlyViewedItem[]>([]);

  useEffect(() => {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) setItems(JSON.parse(raw));
  }, []);

  if (items.length === 0) return null;

  return (
    <section className="py-8">
      <h2 className="text-xl font-semibold mb-4">Recently Viewed</h2>
      <div className="flex gap-4 overflow-x-auto pb-2">
        {items.map((item) => (
          <Link
            key={item.product_id}
            href={`/products/${item.product_id}`}
            className="flex-shrink-0 w-32 group"
          >
            {item.image && (
              <img
                src={item.image}
                alt={item.name}
                className="w-32 h-32 object-cover rounded-lg mb-2 group-hover:opacity-90 transition-opacity"
              />
            )}
            <p className="text-sm text-center line-clamp-2">{item.name}</p>
          </Link>
        ))}
      </div>
    </section>
  );
}
