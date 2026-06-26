"use client";

import { useState, useRef, useEffect } from "react";
import Link from "next/link";
import Image from "next/image";
import { motion } from "framer-motion";
import { Heart, ShoppingCart, Star, Zap } from "lucide-react";
import { cn } from "@/lib/utils";

interface ProductImage {
  url: string;
}

interface Product {
  id: string;
  name: string;
  brand?: string;
  category_name?: string;
  base_price?: number;
  price_amount?: number;
  sku_count?: number;
  images?: ProductImage[];
  rating?: number;
  review_count?: number;
  discount_percent?: number;
  is_new?: boolean;
  free_shipping?: boolean;
  is_verified?: boolean;
}

function formatPrice(cents: number, currency = "INR") {
  return new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency,
    maximumFractionDigits: 0,
  }).format(cents / 100);
}

interface ProductCardProps {
  product: Product;
  images?: ProductImage[];
  priority?: boolean;
}

export default function ProductCard({ product, images, priority = false }: ProductCardProps) {
  const [wishlist, setWishlist] = useState(false);
  const [imgIdx, setImgIdx] = useState(0);
  const [hovered, setHovered] = useState(false);
  const hoverTimer = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    return () => {
      if (hoverTimer.current) clearInterval(hoverTimer.current);
    };
  }, []);

  const allImages = images?.length ? images : (product.images ?? []);
  const hasMultiple = allImages.length > 1;

  const resolvedPrice = product.base_price ?? product.price_amount ?? 0;
  const discountedPrice =
    product.discount_percent && product.discount_percent > 0
      ? Math.round(resolvedPrice * (1 - product.discount_percent / 100))
      : resolvedPrice;

  const handleMouseEnter = () => {
    setHovered(true);
    if (hasMultiple) {
      hoverTimer.current = setInterval(() => {
        setImgIdx(i => (i + 1) % allImages.length);
      }, 1800);
    }
  };

  const handleMouseLeave = () => {
    setHovered(false);
    if (hoverTimer.current) {
      clearInterval(hoverTimer.current);
      hoverTimer.current = null;
    }
    setTimeout(() => setImgIdx(0), 300);
  };

  return (
    <motion.div
      className={cn(
        "bg-white border border-[#E8E8E8] rounded-lg overflow-hidden transition-shadow duration-300",
        hovered
          ? "shadow-[0_4px_12px_rgba(0,0,0,0.08)]"
          : "shadow-sm"
      )}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      whileHover={{ y: -3 }}
      transition={{ duration: 0.25, ease: [0.25, 0.1, 0.25, 1] }}
    >
      <Link href={`/products/${product.id}`} className="block">
        {/* Image area — square aspect ratio */}
        <div className="relative bg-[#F6F6F6] overflow-hidden" style={{ aspectRatio: "1/1" }}>
          {allImages[0] ? (
            <Image
              src={allImages[imgIdx]?.url ?? allImages[0].url}
              alt={product.name}
              fill
              sizes="(max-width: 640px) 50vw, (max-width: 1024px) 33vw, 25vw"
              className={cn(
                "object-contain transition-transform duration-300",
                hovered ? "scale-[1.04]" : "scale-100"
              )}
              priority={priority}
            />
          ) : (
            <div className="absolute inset-0 flex items-center justify-center">
              <Zap className="h-10 w-10 text-[#D0D0D0]" />
            </div>
          )}

          {/* Top-left badges */}
          <div className="absolute top-2 left-2 flex flex-col gap-1">
            {product.discount_percent && product.discount_percent > 0 && (
              <span className="px-1.5 py-0.5 bg-[#E91E8C] text-white text-[10px] font-bold rounded-sm tabular-nums leading-none">
                -{product.discount_percent}%
              </span>
            )}
            {product.is_new && (
              <span className="px-1.5 py-0.5 bg-[#111111] text-white text-[10px] font-bold rounded-sm leading-none">
                NEW
              </span>
            )}
          </div>

          {/* Wishlist button — always visible */}
          <motion.button
            onClick={e => {
              e.preventDefault();
              setWishlist(w => !w);
            }}
            className="absolute top-2 right-2 h-7 w-7 rounded-full bg-white border border-[#E8E8E8] flex items-center justify-center shadow-sm"
            whileTap={{ scale: 0.9 }}
            aria-label={wishlist ? "Remove from wishlist" : "Add to wishlist"}
          >
            <Heart
              className={cn(
                "h-3.5 w-3.5 transition-colors",
                wishlist ? "fill-[#E91E8C] text-[#E91E8C]" : "text-[#999999]"
              )}
            />
          </motion.button>

          {/* Image indicator dots */}
          {hasMultiple && hovered && (
            <div className="absolute bottom-2 left-1/2 -translate-x-1/2 flex items-center gap-1">
              {allImages.slice(0, 5).map((_, i) => (
                <button
                  key={i}
                  onMouseEnter={e => {
                    e.preventDefault();
                    if (hoverTimer.current) clearInterval(hoverTimer.current);
                    setImgIdx(i);
                  }}
                  className={cn(
                    "rounded-full transition-all duration-200",
                    i === imgIdx ? "h-1.5 w-4 bg-white" : "h-1.5 w-1.5 bg-white/60"
                  )}
                />
              ))}
            </div>
          )}
        </div>

        {/* Card body */}
        <div className="px-3 pt-3 pb-3.5">
          {/* Category / Brand */}
          <p className="text-[10px] font-semibold uppercase tracking-wide text-[#999999] leading-none truncate">
            {product.brand ?? product.category_name ?? ""}
          </p>

          {/* Product name */}
          <h3 className="text-sm font-medium text-[#111111] line-clamp-2 leading-snug mt-1">
            {product.name}
          </h3>

          {/* Rating */}
          {product.rating !== undefined && (
            <div className="flex items-center gap-1 mt-1.5">
              {Array.from({ length: 5 }).map((_, i) => (
                <Star
                  key={i}
                  className={cn(
                    "h-3 w-3",
                    i < Math.round(product.rating!)
                      ? "fill-[#D97706] text-[#D97706]"
                      : "fill-[#E8E8E8] text-[#E8E8E8]"
                  )}
                />
              ))}
              {product.review_count !== undefined && (
                <span className="text-[10px] text-[#999999] ml-0.5">
                  ({product.review_count.toLocaleString("en-IN")})
                </span>
              )}
            </div>
          )}

          {/* Price row */}
          <div className="flex items-baseline gap-1.5 mt-2">
            <span className="text-base font-bold text-[#111111] tabular-nums">
              {formatPrice(discountedPrice)}
            </span>
            {product.discount_percent && product.discount_percent > 0 && (
              <span className="text-xs text-[#999999] line-through tabular-nums">
                {formatPrice(resolvedPrice)}
              </span>
            )}
          </div>

          {/* Delivery */}
          <p className="mt-1 text-[11px] leading-none">
            {product.free_shipping ? (
              <span className="text-[#16A34A]">Free delivery</span>
            ) : (
              <span className="text-[#999999]">Delivery charges apply</span>
            )}
          </p>

          {/* Add to cart button */}
          <motion.button
            onClick={e => e.preventDefault()}
            className="w-full h-8 mt-2.5 border border-[#E8E8E8] rounded-md text-xs font-medium text-[#111111] hover:bg-[#111111] hover:text-white hover:border-[#111111] transition-all duration-200 flex items-center justify-center gap-1.5"
            whileTap={{ scale: 0.97 }}
          >
            <ShoppingCart className="h-3.5 w-3.5" />
            + Add to cart
          </motion.button>
        </div>
      </Link>
    </motion.div>
  );
}
