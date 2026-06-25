"use client";
import { useState, useEffect, useRef, useCallback } from "react";
import Link from "next/link";
import Image from "next/image";
import { ShoppingCart, Heart, ChevronRight } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

interface Product {
  id: string;
  name: string;
  slug?: string;
  price_amount?: number;
  currency?: string;
  images?: { url: string }[];
  sku_count?: number;
}

const ACCENTS = [
  { color: "#FF2D78", bg: "#FF2D7812" },
  { color: "#00736A", bg: "#00736A12" },
  { color: "#FF8C00", bg: "#FF8C0012" },
  { color: "#6B2FFF", bg: "#6B2FFF12" },
];

function accentFromId(id: string) {
  let h = 0;
  for (let i = 0; i < id.length; i++) h = ((h * 31) + id.charCodeAt(i)) >>> 0;
  return ACCENTS[h % ACCENTS.length];
}

export default function ProductCard({ product }: { product: Product }) {
  const images = product.images ?? [];
  const imgs = images.length > 0 ? images.map((i) => i.url) : ["/placeholder-product.png"];
  const hasMultiple = imgs.length > 1;

  const [idx, setIdx] = useState(0);
  const [fading, setFading] = useState(false);
  const [hovered, setHovered] = useState(false);
  const [wished, setWished] = useState(false);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const price = product.price_amount ? (product.price_amount / 100).toFixed(2) : null;
  const currencySymbol = product.currency === "INR" || !product.currency ? "₹" : product.currency;
  const accent = accentFromId(product.id);

  const advance = useCallback(() => {
    setFading(true);
    setTimeout(() => {
      setIdx((i) => (i + 1) % imgs.length);
      setFading(false);
    }, 220);
  }, [imgs.length]);

  const goTo = useCallback((next: number) => {
    if (fading) return;
    setFading(true);
    setTimeout(() => {
      setIdx(next);
      setFading(false);
    }, 220);
  }, [fading]);

  useEffect(() => {
    if (!hovered || !hasMultiple) return;
    timerRef.current = setInterval(advance, 2400);
    return () => { if (timerRef.current) clearInterval(timerRef.current); };
  }, [hovered, hasMultiple, advance]);

  return (
    <Link
      href={`/products/${product.id}`}
      className="group block cursor-pointer"
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <Card className="group overflow-hidden cursor-pointer transition-all duration-300 hover:-translate-y-1 hover:shadow-lg border-[#F0EDE8]">
        {/* Image area */}
        <div
          className="relative overflow-hidden aspect-square"
          style={{ background: accent.bg }}
        >
          <Image
            src={imgs[idx]}
            alt={product.name}
            fill
            className="object-contain transition-all duration-500 group-hover:scale-105"
            sizes="(max-width: 640px) 50vw, 25vw"
            style={{
              opacity: fading ? 0 : 1,
              transition: "opacity 220ms ease, transform 500ms cubic-bezier(0.4,0,0.2,1)",
              padding: "12%",
            }}
          />

          {/* Wishlist button */}
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={wished ? "Remove from wishlist" : "Add to wishlist"}
            onClick={(e) => { e.preventDefault(); setWished((v) => !v); }}
            className="absolute top-2.5 right-2.5 w-8 h-8 rounded-full z-10
                       opacity-0 group-hover:opacity-100 hover:scale-110 active:scale-90
                       transition-all duration-200"
            style={{
              background: wished ? "#FF2D78" : "rgba(255,255,255,0.92)",
              boxShadow: "0 2px 8px rgba(26,18,8,0.12)",
            }}
          >
            <Heart
              size={13}
              fill={wished ? "#fff" : "none"}
              stroke={wished ? "#fff" : "#9CA3AF"}
              strokeWidth={2}
            />
          </Button>

          {/* Variant count badge */}
          {product.sku_count != null && product.sku_count > 1 && (
            <Badge
              className="absolute top-2.5 left-2.5 text-[9px] font-bold uppercase tracking-wide px-2 py-1 rounded-full border-0"
              style={{ background: accent.color, color: "#fff" }}
            >
              {product.sku_count} variants
            </Badge>
          )}

          {/* Image dots */}
          {hasMultiple && (
            <div
              className="absolute bottom-2.5 left-0 right-0 flex justify-center gap-1 z-10
                         opacity-0 group-hover:opacity-100 transition-opacity duration-200"
            >
              {imgs.slice(0, 5).map((_, i) => (
                <button
                  key={i}
                  type="button"
                  onClick={(e) => { e.preventDefault(); goTo(i); }}
                  className="rounded-full cursor-pointer transition-all duration-300"
                  style={{
                    width: i === idx ? "18px" : "5px",
                    height: "5px",
                    background: i === idx ? accent.color : "rgba(255,255,255,0.6)",
                    boxShadow: i === idx ? `0 0 0 2px ${accent.color}44` : "none",
                  }}
                  aria-label={`Image ${i + 1}`}
                />
              ))}
            </div>
          )}
        </div>

        {/* Info area */}
        <CardContent className="p-3.5 pb-4">
          <p
            className="text-[13px] font-medium line-clamp-2 leading-snug mb-2.5"
            style={{ color: "#1A1208" }}
          >
            {product.name}
          </p>

          <div className="flex items-center justify-between gap-2">
            {price ? (
              <p
                className="text-base font-bold leading-none text-primary"
                style={{
                  color: accent.color,
                  fontVariantNumeric: "tabular-nums",
                  fontFamily: "var(--font-syne)",
                }}
              >
                {currencySymbol} {price}
              </p>
            ) : (
              <span className="text-xs font-medium text-muted-foreground">—</span>
            )}

            {/* Shop CTA badge — slides in on hover */}
            <Badge
              className="flex items-center gap-1 text-[11px] font-bold rounded-full px-2.5 py-1 border-0
                         transition-all duration-200
                         opacity-0 translate-x-2 group-hover:opacity-100 group-hover:translate-x-0"
              style={{ background: accent.bg, color: accent.color }}
            >
              <ShoppingCart size={10} />
              Shop
              <ChevronRight size={10} strokeWidth={2.5} />
            </Badge>
          </div>
        </CardContent>

        {/* Accent bottom bar */}
        <div
          className="h-0.5 transition-all duration-300 group-hover:h-1"
          style={{ background: accent.color }}
        />
      </Card>
    </Link>
  );
}
