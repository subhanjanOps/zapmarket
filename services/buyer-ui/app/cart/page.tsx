"use client";

import Image from "next/image";
import Link from "next/link";
import { motion, AnimatePresence } from "framer-motion";
import { useCartStore } from "@/lib/cart";
import { Minus, Plus, Trash2, ShoppingBag, Tag } from "lucide-react";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const CARD_SHADOW =
  "shadow-[0_1px_3px_rgba(15,10,4,0.06),0_4px_12px_rgba(15,10,4,0.04)]";

export default function CartPage() {
  const { items, removeItem, updateQty, total } = useCartStore();

  /* ── Empty state ── */
  if (items.length === 0) {
    return (
      <div className="min-h-[70vh] flex items-center justify-center bg-[#F9F8F5] px-4">
        <div className="flex flex-col items-center gap-5 text-center">
          <div className="w-20 h-20 rounded-full bg-white flex items-center justify-center shadow-[0_1px_3px_rgba(15,10,4,0.06),0_4px_12px_rgba(15,10,4,0.04)]">
            <ShoppingBag size={40} className="text-[#EDE9E3]" />
          </div>
          <div className="space-y-1.5">
            <h2
              className="text-2xl font-bold text-[#0F0A04]"
              style={{ fontFamily: "var(--font-syne)" }}
            >
              Your cart is empty
            </h2>
            <p className="text-sm text-[#7A6856]">
              Looks like you haven&apos;t added anything yet.
            </p>
          </div>
          <Link
            href="/products"
            className="inline-flex items-center justify-center h-11 px-8 rounded-2xl bg-[#E91E8C] hover:bg-[#B5166E] text-white text-sm font-bold transition-colors shadow-[0_4px_20px_rgba(233,30,140,0.25)]"
          >
            Shop now
          </Link>
        </div>
      </div>
    );
  }

  const subtotal = total();

  return (
    <div className="min-h-screen bg-[#F9F8F5]">
      <div className="max-w-5xl mx-auto px-4 py-10">
        {/* ── Page heading ── */}
        <div className="flex items-center gap-3 mb-8">
          <h1
            className="text-2xl font-bold text-[#0F0A04]"
            style={{ fontFamily: "var(--font-syne)" }}
          >
            Shopping Cart
          </h1>
          <span className="inline-flex items-center justify-center h-6 min-w-[1.5rem] px-2 rounded-full bg-[#E91E8C] text-white text-xs font-bold">
            {items.length}
          </span>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* ── Cart items ── */}
          <div className="lg:col-span-2 space-y-3">
            <AnimatePresence initial={false}>
              {items.map((item, i) => (
                <motion.div
                  key={item.skuId}
                  initial={{ opacity: 0, x: -8 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: 8, height: 0, marginBottom: 0 }}
                  transition={{ duration: 0.22, delay: i * 0.04 }}
                  className={cn(
                    "bg-white rounded-2xl px-5 py-4 flex items-center gap-4",
                    CARD_SHADOW
                  )}
                >
                  {/* Product image */}
                  <div className="relative w-[72px] h-[72px] shrink-0 rounded-xl overflow-hidden bg-[#F9F8F5] border border-[#EDE9E3]">
                    <Image
                      src={item.image || "/placeholder-product.png"}
                      alt={item.name}
                      fill
                      className="object-contain p-1.5"
                      sizes="72px"
                    />
                  </div>

                  {/* Info + stepper */}
                  <div className="flex-1 min-w-0 space-y-2">
                    <p className="text-sm font-semibold text-[#0F0A04] line-clamp-2 leading-snug">
                      {item.name}
                    </p>
                    <p className="text-xs text-[#7A6856]">
                      {item.currency}&nbsp;
                      {(item.price / 100).toFixed(2)} / unit
                    </p>

                    {/* Qty stepper */}
                    <div className="inline-flex items-center border border-[#EDE9E3] rounded-xl overflow-hidden">
                      <button
                        aria-label="Decrease quantity"
                        onClick={() => updateQty(item.skuId, item.qty - 1)}
                        className="h-8 w-8 flex items-center justify-center text-[#3D2E1A] hover:bg-[#F3F0EB] transition-colors"
                      >
                        <Minus size={13} />
                      </button>
                      <span className="w-10 text-center text-sm font-bold text-[#0F0A04] tabular-nums select-none">
                        {item.qty}
                      </span>
                      <button
                        aria-label="Increase quantity"
                        onClick={() => updateQty(item.skuId, item.qty + 1)}
                        className="h-8 w-8 flex items-center justify-center text-[#3D2E1A] hover:bg-[#F3F0EB] transition-colors"
                      >
                        <Plus size={13} />
                      </button>
                    </div>
                  </div>

                  {/* Line total + remove */}
                  <div className="flex flex-col items-end gap-2 shrink-0">
                    <span
                      className="text-sm font-bold text-[#0F0A04] tabular-nums"
                      style={{ fontFamily: "var(--font-syne)" }}
                    >
                      {item.currency}&nbsp;
                      {((item.price * item.qty) / 100).toFixed(2)}
                    </span>
                    <button
                      aria-label="Remove item"
                      onClick={() => removeItem(item.skuId)}
                      className="h-8 w-8 rounded-xl flex items-center justify-center text-[#B8A898] hover:text-[#E91E8C] hover:bg-[#FDE8F4] transition-colors"
                    >
                      <Trash2 size={15} />
                    </button>
                  </div>
                </motion.div>
              ))}
            </AnimatePresence>
          </div>

          {/* ── Order Summary ── */}
          <div className="lg:col-span-1">
            <div
              className={cn(
                "bg-white rounded-2xl p-6 sticky top-24 space-y-5",
                CARD_SHADOW
              )}
            >
              <h2
                className="text-lg font-bold text-[#0F0A04]"
                style={{ fontFamily: "var(--font-syne)" }}
              >
                Order Summary
              </h2>

              {/* Line rows */}
              <div className="space-y-3 text-sm">
                <div className="flex justify-between">
                  <span className="text-[#7A6856]">Subtotal</span>
                  <span className="font-semibold text-[#0F0A04] tabular-nums">
                    ₹&nbsp;{(subtotal / 100).toFixed(2)}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-[#7A6856]">Shipping</span>
                  <span className="font-semibold text-[#12845F]">Free</span>
                </div>
              </div>

              {/* Divider */}
              <div className="border-t border-[#EDE9E3]" />

              {/* Total */}
              <div className="flex justify-between items-baseline">
                <span className="text-sm font-bold text-[#0F0A04]">Total</span>
                <span
                  className="text-xl font-bold text-[#0F0A04] tabular-nums"
                  style={{ fontFamily: "var(--font-syne)" }}
                >
                  ₹&nbsp;{(subtotal / 100).toFixed(2)}
                </span>
              </div>

              {/* Promo code */}
              <div className="flex gap-2">
                <div className="relative flex-1">
                  <Tag
                    size={14}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-[#B8A898]"
                  />
                  <input
                    type="text"
                    placeholder="Promo code"
                    className="w-full h-10 pl-8 pr-3 text-sm border border-[#EDE9E3] rounded-xl bg-[#F9F8F5] placeholder:text-[#B8A898] text-[#0F0A04] focus:outline-none focus:border-[#E91E8C] transition-colors"
                  />
                </div>
                <button className="h-10 px-4 rounded-xl bg-[#F3F0EB] text-sm font-semibold text-[#3D2E1A] hover:bg-[#EDE9E3] transition-colors shrink-0">
                  Apply
                </button>
              </div>

              {/* Checkout CTA */}
              <Link
                href="/checkout"
                className="flex items-center justify-center w-full h-12 rounded-2xl bg-[#E91E8C] hover:bg-[#B5166E] text-white text-sm font-bold transition-colors shadow-[0_4px_20px_rgba(233,30,140,0.25)]"
              >
                Proceed to Checkout
              </Link>

              {/* Continue shopping */}
              <Link
                href="/products"
                className="flex items-center justify-center w-full text-sm text-[#7A6856] hover:text-[#3D2E1A] transition-colors font-medium"
              >
                Continue Shopping
              </Link>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
