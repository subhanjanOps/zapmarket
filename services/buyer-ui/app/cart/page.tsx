"use client";

import Image from "next/image";
import Link from "next/link";
import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { useCartStore } from "@/lib/cart";
import { Minus, Plus, Trash2, ShoppingBag } from "lucide-react";

export default function CartPage() {
  const { items, removeItem, updateQty, total } = useCartStore();
  const [promoCode, setPromoCode] = useState("");

  /* ── Empty state ── */
  if (items.length === 0) {
    return (
      <div className="min-h-[60vh] flex items-center justify-center px-4">
        <motion.div
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.35, ease: [0.25, 0.1, 0.25, 1] }}
          className="flex flex-col items-center gap-4 text-center"
        >
          <ShoppingBag className="h-12 w-12 text-[#E8E8E8]" />
          <div>
            <h2 className="text-xl font-semibold text-[#111111]">
              Your cart is empty
            </h2>
            <p className="text-sm text-[#555555] mt-1">
              Add some products to continue shopping.
            </p>
          </div>
          <Link
            href="/products"
            className="inline-flex items-center justify-center h-10 px-8 rounded-md bg-[#E91E8C] hover:bg-[#C2187A] text-white text-sm font-medium transition-colors"
          >
            Start Shopping
          </Link>
        </motion.div>
      </div>
    );
  }

  const subtotal = total();

  return (
    <div className="bg-white min-h-screen">
      <div className="max-w-5xl mx-auto px-4 py-10">
        {/* ── Page heading ── */}
        <motion.h1
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.35, ease: [0.25, 0.1, 0.25, 1] }}
          className="text-2xl font-bold text-[#111111] mb-8"
        >
          Cart ({items.length} {items.length === 1 ? "item" : "items"})
        </motion.h1>

        <div className="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-8">
          {/* ── Cart items ── */}
          <div>
            <AnimatePresence initial={false}>
              {items.map((item) => (
                <motion.div
                  key={item.skuId}
                  initial={{ opacity: 0, y: 8 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, height: 0, marginBottom: 0 }}
                  transition={{ duration: 0.25, ease: [0.25, 0.1, 0.25, 1] }}
                  className="flex items-center gap-4 py-5 border-b border-[#E8E8E8]"
                >
                  {/* Product image */}
                  <div className="relative w-20 h-20 shrink-0 rounded-md border border-[#E8E8E8] bg-[#F6F6F6] overflow-hidden">
                    <Image
                      src={item.image || "/placeholder-product.png"}
                      alt={item.name}
                      fill
                      className="object-contain p-2"
                      sizes="80px"
                    />
                  </div>

                  {/* Info */}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-[#111111] line-clamp-2 leading-snug">
                      {item.name}
                    </p>
                    <p className="text-xs text-[#555555] mt-1">
                      {item.currency}&nbsp;
                      {(item.price / 100).toFixed(2)} / unit
                    </p>

                    {/* Qty stepper */}
                    <div className="inline-flex items-center gap-0 border border-[#E8E8E8] rounded-md overflow-hidden h-8 mt-3">
                      <button
                        aria-label="Decrease quantity"
                        onClick={() => updateQty(item.skuId, item.qty - 1)}
                        className="h-8 w-8 flex items-center justify-center text-[#555555] hover:bg-[#F6F6F6] transition-colors"
                      >
                        <Minus size={13} />
                      </button>
                      <span className="w-9 text-center text-sm font-medium text-[#111111] select-none tabular-nums">
                        {item.qty}
                      </span>
                      <button
                        aria-label="Increase quantity"
                        onClick={() => updateQty(item.skuId, item.qty + 1)}
                        className="h-8 w-8 flex items-center justify-center text-[#555555] hover:bg-[#F6F6F6] transition-colors"
                      >
                        <Plus size={13} />
                      </button>
                    </div>
                  </div>

                  {/* Line total + remove */}
                  <div className="flex flex-col items-end gap-2 shrink-0">
                    <span className="text-sm font-bold text-[#111111] tabular-nums">
                      {item.currency}&nbsp;
                      {((item.price * item.qty) / 100).toFixed(2)}
                    </span>
                    <button
                      aria-label="Remove item"
                      onClick={() => removeItem(item.skuId)}
                      className="flex items-center justify-center text-[#999999] hover:text-[#DC2626] transition-colors"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </motion.div>
              ))}
            </AnimatePresence>
          </div>

          {/* ── Order Summary ── */}
          <div>
            <div className="sticky top-24 bg-[#F6F6F6] rounded-lg p-6 border border-[#E8E8E8]">
              <h2 className="text-base font-semibold text-[#111111] mb-4">
                Order Summary
              </h2>

              {/* Promo code */}
              <div className="flex gap-2 mb-5">
                <input
                  type="text"
                  value={promoCode}
                  onChange={(e) => setPromoCode(e.target.value)}
                  placeholder="Promo code"
                  className="flex-1 h-9 bg-white border border-[#E8E8E8] rounded-md text-sm px-3 placeholder:text-[#999999] text-[#111111] focus:outline-none focus:border-[#D0D0D0] transition-colors"
                />
                <button className="h-9 px-4 border border-[#E8E8E8] rounded-md text-sm text-[#111111] bg-white hover:bg-white transition-colors shrink-0">
                  Apply
                </button>
              </div>

              {/* Line rows */}
              <div className="space-y-3 text-sm">
                <div className="flex justify-between">
                  <span className="text-[#555555]">Subtotal</span>
                  <span className="font-medium text-[#111111] tabular-nums">
                    ₹&nbsp;{(subtotal / 100).toFixed(2)}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-[#555555]">Shipping</span>
                  <span className="font-medium text-[#16A34A]">Free</span>
                </div>
              </div>

              {/* Divider */}
              <div className="border-t border-[#E8E8E8] my-4" />

              {/* Total */}
              <div className="flex justify-between items-center">
                <span className="text-base font-bold text-[#111111]">Total</span>
                <span className="text-base font-bold text-[#111111] tabular-nums">
                  ₹&nbsp;{(subtotal / 100).toFixed(2)}
                </span>
              </div>

              {/* Checkout CTA */}
              <Link
                href="/checkout"
                className="flex items-center justify-center w-full h-11 bg-[#E91E8C] hover:bg-[#C2187A] text-white font-medium rounded-md text-sm mt-4 transition-colors"
              >
                Proceed to Checkout
              </Link>

              {/* Continue shopping */}
              <Link
                href="/products"
                className="block text-sm text-[#555555] hover:text-[#111111] mt-3 text-center transition-colors"
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
