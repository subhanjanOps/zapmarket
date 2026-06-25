"use client";
import Image from "next/image";
import Link from "next/link";
import { useCartStore } from "@/lib/cart";
import { Minus, Plus, Trash2, ShoppingBag } from "lucide-react";

export default function CartPage() {
  const { items, removeItem, updateQty, total } = useCartStore();

  if (items.length === 0) {
    return (
      <div className="max-w-3xl mx-auto px-4 py-24 text-center animate-fade-up">
        <div
          className="w-20 h-20 rounded-full mx-auto mb-6 flex items-center justify-center"
          style={{ background: "#FFF0F3" }}
        >
          <ShoppingBag size={32} style={{ color: "#FF2D78" }} />
        </div>
        <h1 className="text-2xl font-bold mb-2" style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}>
          Your cart is empty
        </h1>
        <p className="text-sm mb-8" style={{ color: "#6B6052" }}>
          Looks like you haven&apos;t added anything yet.
        </p>
        <Link
          href="/products"
          className="inline-block font-bold px-8 py-3.5 rounded-xl text-white
                     transition-all duration-200 hover:scale-105 hover:shadow-lg active:scale-95"
          style={{ background: "#FF2D78", fontFamily: "var(--font-syne)" }}
        >
          Start Shopping
        </Link>
      </div>
    );
  }

  const subtotal = total();

  return (
    <div className="max-w-5xl mx-auto px-4 py-10">
      <h1
        className="text-2xl font-bold mb-7 animate-fade-up"
        style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}
      >
        Shopping Cart
      </h1>
      <div className="flex flex-col lg:flex-row gap-8">
        {/* Items */}
        <div className="flex-1 space-y-3">
          {items.map((item, i) => (
            <div
              key={item.skuId}
              className="flex gap-4 rounded-2xl p-4 animate-fade-up"
              style={{
                background: "#fff",
                border: "1px solid #F0EDE8",
                boxShadow: "0 1px 4px rgba(26,18,8,0.04)",
                animationDelay: `${i * 60}ms`,
              }}
            >
              <div
                className="w-20 h-20 relative shrink-0 rounded-xl overflow-hidden"
                style={{ background: "#FFFCF5" }}
              >
                <Image
                  src={item.image || "/placeholder-product.png"}
                  alt={item.name}
                  fill
                  className="object-contain p-1"
                  sizes="80px"
                />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate" style={{ color: "#1A1208" }}>{item.name}</p>
                <p
                  className="text-sm font-extrabold mt-1"
                  style={{ color: "#FF2D78", fontVariantNumeric: "tabular-nums", fontFamily: "var(--font-syne)" }}
                >
                  {item.currency} {(item.price / 100).toFixed(2)}
                </p>
                <div className="flex items-center gap-2 mt-2.5">
                  <button
                    onClick={() => updateQty(item.skuId, item.qty - 1)}
                    className="p-1.5 rounded-lg transition-all duration-150 hover:bg-gray-100 active:scale-90 cursor-pointer"
                    style={{ border: "1px solid #F0EDE8" }}
                    aria-label="Decrease"
                  >
                    <Minus size={13} />
                  </button>
                  <span
                    className="text-sm w-7 text-center font-bold"
                    style={{ color: "#1A1208", fontVariantNumeric: "tabular-nums" }}
                  >
                    {item.qty}
                  </span>
                  <button
                    onClick={() => updateQty(item.skuId, item.qty + 1)}
                    className="p-1.5 rounded-lg transition-all duration-150 hover:bg-gray-100 active:scale-90 cursor-pointer"
                    style={{ border: "1px solid #F0EDE8" }}
                    aria-label="Increase"
                  >
                    <Plus size={13} />
                  </button>
                  <button
                    onClick={() => removeItem(item.skuId)}
                    className="ml-2 p-1.5 rounded-lg transition-all duration-150 hover:bg-red-50 text-red-400 hover:text-red-600 active:scale-90 cursor-pointer"
                    aria-label="Remove"
                  >
                    <Trash2 size={13} />
                  </button>
                </div>
              </div>
              <p
                className="text-sm font-bold shrink-0 self-center"
                style={{ color: "#1A1208", fontVariantNumeric: "tabular-nums", fontFamily: "var(--font-syne)" }}
              >
                {item.currency} {((item.price * item.qty) / 100).toFixed(2)}
              </p>
            </div>
          ))}
        </div>

        {/* Summary */}
        <div className="lg:w-72">
          <div
            className="rounded-2xl p-6 sticky top-20 animate-scale-in"
            style={{ background: "#fff", border: "1px solid #F0EDE8", boxShadow: "0 2px 12px rgba(26,18,8,0.06)" }}
          >
            <h2
              className="font-bold mb-5 text-base"
              style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}
            >
              Order Summary
            </h2>
            <div className="space-y-3 text-sm">
              <div className="flex justify-between">
                <span style={{ color: "#6B6052" }}>Subtotal</span>
                <span className="font-semibold" style={{ fontVariantNumeric: "tabular-nums" }}>
                  ₹ {(subtotal / 100).toFixed(2)}
                </span>
              </div>
              <div className="flex justify-between">
                <span style={{ color: "#6B6052" }}>Estimated tax</span>
                <span style={{ color: "#9CA3AF" }}>Included</span>
              </div>
            </div>
            <div
              className="flex justify-between font-bold mt-4 pt-4 text-base"
              style={{ borderTop: "1px solid #F0EDE8" }}
            >
              <span>Total</span>
              <span style={{ color: "#FF2D78", fontVariantNumeric: "tabular-nums", fontFamily: "var(--font-syne)" }}>
                ₹ {(subtotal / 100).toFixed(2)}
              </span>
            </div>
            <Link
              href="/checkout"
              className="mt-5 block w-full text-center font-bold py-3.5 rounded-xl text-white
                         transition-all duration-200 hover:scale-[1.02] hover:shadow-lg active:scale-[0.98] cursor-pointer"
              style={{ background: "#FF2D78", fontFamily: "var(--font-syne)" }}
            >
              Proceed to Checkout
            </Link>
            <Link
              href="/products"
              className="mt-3 block w-full text-center text-sm font-semibold py-2.5 rounded-xl
                         transition-all duration-150 hover:bg-gray-50 cursor-pointer"
              style={{ color: "#6B6052" }}
            >
              Continue shopping
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
