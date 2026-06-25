"use client";

import { useState, useRef } from "react";
import Link from "next/link";
import Image from "next/image";
import { motion, AnimatePresence } from "framer-motion";
import { Heart, ShoppingCart, Star, Truck, Zap, Eye, BadgeCheck } from "lucide-react";
import { cn } from "@/lib/utils";

interface ProductImage {
  url: string;
}

interface Product {
  id: string;
  name: string;
  brand?: string;
  category_name?: string;
  base_price: number;
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

  const allImages = images?.length ? images : (product.images ?? []);
  const hasMultiple = allImages.length > 1;

  const discountedPrice =
    product.discount_percent && product.discount_percent > 0
      ? Math.round(product.base_price * (1 - product.discount_percent / 100))
      : product.base_price;

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
      className="group bg-white rounded-2xl overflow-hidden shadow-[0_1px_3px_rgba(15,10,4,0.06),0_4px_12px_rgba(15,10,4,0.04)] hover:shadow-[0_8px_24px_rgba(15,10,4,0.12),0_2px_6px_rgba(15,10,4,0.06)] transition-shadow duration-300"
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      whileHover={{ y: -2 }}
      transition={{ duration: 0.25, ease: [0.16, 1, 0.3, 1] }}
    >
      <Link href={`/products/${product.id}`} className="block">
        {/* Image container */}
        <div className="relative bg-[#F9F8F5] overflow-hidden" style={{ aspectRatio: "4/5" }}>
          {allImages[0] ? (
            <Image
              src={allImages[imgIdx]?.url ?? allImages[0].url}
              alt={product.name}
              fill
              sizes="(max-width: 640px) 50vw, (max-width: 1024px) 33vw, 25vw"
              className="object-cover transition-transform duration-500 ease-[cubic-bezier(0.16,1,0.3,1)] group-hover:scale-105"
              priority={priority}
            />
          ) : (
            <div className="absolute inset-0 flex items-center justify-center">
              <Zap className="h-12 w-12 text-[#EDE9E3]" />
            </div>
          )}

          {/* Top badges */}
          <div className="absolute top-2.5 left-2.5 flex flex-col gap-1.5">
            {product.discount_percent && product.discount_percent > 0 && (
              <span className="px-2 py-0.5 bg-[#E91E8C] text-white text-xs font-bold rounded-lg shadow-sm tabular-nums">
                -{product.discount_percent}%
              </span>
            )}
            {product.is_new && (
              <span className="px-2 py-0.5 bg-[#0F0A04] text-white text-xs font-bold rounded-lg">
                NEW
              </span>
            )}
          </div>

          {/* Wishlist button */}
          <motion.button
            onClick={e => {
              e.preventDefault();
              setWishlist(w => !w);
            }}
            className="absolute top-2.5 right-2.5 h-8 w-8 rounded-full bg-white/90 backdrop-blur-sm flex items-center justify-center shadow-sm opacity-0 group-hover:opacity-100 transition-opacity"
            whileTap={{ scale: 0.8 }}
            aria-label={wishlist ? "Remove from wishlist" : "Add to wishlist"}
          >
            <Heart
              className={cn(
                "h-4 w-4 transition-colors",
                wishlist ? "fill-[#E91E8C] text-[#E91E8C]" : "text-[#7A6856]"
              )}
            />
          </motion.button>

          {/* Image indicator dots */}
          {hasMultiple && hovered && (
            <div className="absolute bottom-2.5 left-1/2 -translate-x-1/2 flex items-center gap-1">
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

          {/* Quick view overlay */}
          <AnimatePresence>
            {hovered && (
              <motion.div
                className="absolute bottom-0 left-0 right-0 p-2.5"
                initial={{ y: 10, opacity: 0 }}
                animate={{ y: 0, opacity: 1 }}
                exit={{ y: 10, opacity: 0 }}
                transition={{ duration: 0.18 }}
              >
                <div className="flex items-center justify-center gap-1.5 bg-white/95 backdrop-blur-sm rounded-xl py-2 px-3 shadow-sm">
                  <Eye className="h-3.5 w-3.5 text-[#7A6856]" />
                  <span className="text-xs font-semibold text-[#3D2E1A]">Quick view</span>
                </div>
              </motion.div>
            )}
          </AnimatePresence>
        </div>

        {/* Card body */}
        <div className="px-4 pt-3.5 pb-2">
          {/* Brand + verified */}
          <div className="flex items-center gap-1.5 mb-1">
            {product.brand && (
              <span className="text-xs font-bold text-[#E91E8C] tracking-wide uppercase leading-none">
                {product.brand}
              </span>
            )}
            {product.is_verified && (
              <BadgeCheck className="h-3.5 w-3.5 text-[#12845F] shrink-0" />
            )}
            {product.category_name && (
              <span className="text-xs text-[#B8A898] truncate leading-none">
                {product.brand ? `· ${product.category_name}` : product.category_name}
              </span>
            )}
          </div>

          {/* Name */}
          <h3 className="text-sm font-semibold text-[#0F0A04] leading-snug line-clamp-2 mb-2">
            {product.name}
          </h3>

          {/* Rating */}
          {product.rating !== undefined && (
            <div className="flex items-center gap-1.5 mb-2">
              <div className="flex items-center gap-0.5">
                {Array.from({ length: 5 }).map((_, i) => (
                  <Star
                    key={i}
                    className={cn(
                      "h-3 w-3",
                      i < Math.round(product.rating!)
                        ? "fill-[#D97706] text-[#D97706]"
                        : "text-[#EDE9E3] fill-[#EDE9E3]"
                    )}
                  />
                ))}
              </div>
              {product.review_count !== undefined && (
                <span className="text-xs text-[#B8A898]">
                  ({product.review_count.toLocaleString("en-IN")})
                </span>
              )}
            </div>
          )}

          {/* Price */}
          <div className="flex items-baseline gap-2 mb-2">
            <span className="text-base font-bold text-[#0F0A04] font-display tabular-nums">
              {formatPrice(discountedPrice)}
            </span>
            {product.discount_percent && product.discount_percent > 0 && (
              <span className="text-xs text-[#B8A898] line-through tabular-nums">
                {formatPrice(product.base_price)}
              </span>
            )}
          </div>

          {/* Delivery */}
          <div className="flex items-center gap-1.5">
            {product.free_shipping ? (
              <>
                <Truck className="h-3 w-3 text-[#12845F]" />
                <span className="text-xs text-[#12845F] font-medium">Free delivery</span>
              </>
            ) : (
              <span className="text-xs text-[#B8A898]">Delivery by tomorrow</span>
            )}
          </div>
        </div>
      </Link>

      {/* Add to cart */}
      <div className="px-4 pb-4 pt-2">
        <motion.button
          className="w-full flex items-center justify-center gap-2 h-9 bg-[#F9F8F5] hover:bg-[#E91E8C] text-[#3D2E1A] hover:text-white text-xs font-semibold rounded-xl transition-all group/btn"
          whileTap={{ scale: 0.97 }}
        >
          <ShoppingCart className="h-3.5 w-3.5 transition-transform group-hover/btn:scale-110" />
          Add to cart
        </motion.button>
      </div>
    </motion.div>
  );
}
