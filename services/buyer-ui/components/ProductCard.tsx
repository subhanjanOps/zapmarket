"use client";
import { useState, useEffect, useRef, useCallback } from "react";
import Link from "next/link";
import Image from "next/image";

interface Product {
  id: string;
  name: string;
  slug?: string;
  price_amount?: number;
  currency?: string;
  images?: { url: string }[];
  sku_count?: number;
}

const TOP_COLORS = ["#FF2D78", "#00736A", "#FF8C00", "#6B2FFF"];

function accentFromId(id: string): string {
  let h = 0;
  for (let i = 0; i < id.length; i++) h = ((h * 31) + id.charCodeAt(i)) >>> 0;
  return TOP_COLORS[h % TOP_COLORS.length];
}

export default function ProductCard({ product }: { product: Product }) {
  const images = product.images ?? [];
  const imgs = images.length > 0 ? images.map((i) => i.url) : ["/placeholder-product.png"];
  const hasMultiple = imgs.length > 1;

  const [idx, setIdx] = useState(0);
  const [fading, setFading] = useState(false);
  const [hovered, setHovered] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const price = product.price_amount ? (product.price_amount / 100).toFixed(2) : null;
  const currencySymbol = product.currency === "INR" || !product.currency ? "₹" : product.currency;
  const accent = accentFromId(product.id);

  const goTo = useCallback((next: number) => {
    if (fading) return;
    setFading(true);
    setTimeout(() => {
      setIdx(next);
      setFading(false);
    }, 220);
  }, [fading]);

  // Auto-advance when hovered
  useEffect(() => {
    if (!hovered || !hasMultiple) return;
    timerRef.current = setInterval(() => {
      setIdx((i) => {
        const next = (i + 1) % imgs.length;
        setFading(true);
        setTimeout(() => setFading(false), 220);
        return next;
      });
    }, 2200);
    return () => { if (timerRef.current) clearInterval(timerRef.current); };
  }, [hovered, hasMultiple, imgs.length]);

  function handleMouseEnter() { setHovered(true); }
  function handleMouseLeave() {
    setHovered(false);
    if (timerRef.current) clearInterval(timerRef.current);
  }

  return (
    <Link
      href={`/products/${product.id}`}
      className="group block rounded-2xl overflow-hidden cursor-pointer
                 transition-all duration-300 hover:-translate-y-2 hover:shadow-xl"
      style={{
        background: "#fff",
        border: "1px solid #F0EDE8",
        boxShadow: "0 1px 4px rgba(26,18,8,0.04)",
      }}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
      {/* Accent top bar */}
      <div
        className="h-1 w-full transition-all duration-300 group-hover:h-1.5"
        style={{ background: accent }}
      />

      {/* Image area */}
      <div className="aspect-square relative overflow-hidden" style={{ background: "#FFFCF5" }}>
        <Image
          src={imgs[idx]}
          alt={product.name}
          fill
          className="object-contain p-3 product-img"
          sizes="(max-width: 640px) 50vw, 25vw"
          style={{ opacity: fading ? 0 : 1, transition: "opacity 220ms ease" }}
        />

        {/* Quick-view overlay */}
        <div
          className="quick-view-overlay absolute inset-x-0 bottom-0 flex items-center justify-center pb-3 pt-5"
          style={{ background: "linear-gradient(to top, rgba(26,18,8,0.55) 0%, transparent 100%)" }}
        >
          <span
            className="text-xs font-bold text-white px-3 py-1.5 rounded-full"
            style={{ background: accent }}
          >
            Quick view →
          </span>
        </div>

        {/* Dot indicators */}
        {hasMultiple && (
          <div className="absolute bottom-2 left-0 right-0 flex justify-center gap-1 z-10 opacity-0 group-hover:opacity-100 transition-opacity duration-200">
            {imgs.map((_, i) => (
              <button
                key={i}
                onClick={(e) => { e.preventDefault(); goTo(i); }}
                className="rounded-full transition-all duration-300 cursor-pointer"
                style={{
                  width: i === idx ? "16px" : "5px",
                  height: "5px",
                  background: i === idx ? "#fff" : "rgba(255,255,255,0.5)",
                }}
                aria-label={`Image ${i + 1}`}
              />
            ))}
          </div>
        )}
      </div>

      {/* Info */}
      <div className="p-3 pb-4">
        <p className="text-sm font-medium line-clamp-2 leading-snug" style={{ color: "#1A1208" }}>
          {product.name}
        </p>
        <div className="mt-2 flex items-center justify-between gap-2">
          {price ? (
            <p
              className="text-base font-extrabold"
              style={{ color: "#FF2D78", fontVariantNumeric: "tabular-nums", fontFamily: "var(--font-syne)" }}
            >
              {currencySymbol} {price}
            </p>
          ) : <span />}
          {product.sku_count != null && product.sku_count > 1 && (
            <span
              className="text-[10px] font-semibold px-2 py-0.5 rounded-full"
              style={{ background: "#F0EDE8", color: "#6B6052" }}
            >
              {product.sku_count} variants
            </span>
          )}
        </div>
      </div>
    </Link>
  );
}
