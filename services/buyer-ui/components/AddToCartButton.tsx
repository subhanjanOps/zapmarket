"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { useCartStore } from "@/lib/cart";
import { ShoppingCart, Loader2, CheckCircle2, Zap } from "lucide-react";
import { motion } from "framer-motion";
import type { Sku } from "./SkuSelector";

interface Props {
  sku: Sku | null;
  productName: string;
  productImage: string;
  qty?: number;
}

export default function AddToCartButton({ sku, productName, productImage, qty = 1 }: Props) {
  const addItem = useCartStore((s) => s.addItem);
  const router = useRouter();
  const [status, setStatus] = useState<"idle" | "loading" | "success">("idle");

  function handleAdd() {
    if (!sku || status !== "idle") return;
    setStatus("loading");
    for (let i = 0; i < qty; i++) {
      addItem({
        skuId: sku.id,
        name: `${productName} — ${sku.sku_code}`,
        image: productImage,
        price: sku.price_amount,
        currency: sku.currency,
      });
    }
    setTimeout(() => {
      setStatus("success");
      setTimeout(() => setStatus("idle"), 1600);
    }, 400);
  }

  function handleBuyNow() {
    if (!sku) return;
    for (let i = 0; i < qty; i++) {
      addItem({
        skuId: sku.id,
        name: `${productName} — ${sku.sku_code}`,
        image: productImage,
        price: sku.price_amount,
        currency: sku.currency,
      });
    }
    router.push("/checkout");
  }

  const isDisabled = !sku || status !== "idle";

  const addButtonClass = [
    "relative flex-1 flex items-center justify-center gap-2",
    "font-bold rounded-2xl h-12 w-full text-white text-sm",
    "transition-colors duration-300",
    status === "success"
      ? "bg-green-600"
      : "bg-[#E91E8C] hover:bg-[#c91878]",
    isDisabled && status === "idle" ? "opacity-50 cursor-not-allowed" : "cursor-pointer",
    // btn-shimmer: pseudo-element shimmer via Tailwind arbitrary variant not available here,
    // so the shimmer is handled by the overlay span below
  ].join(" ");

  return (
    <div className="flex gap-3 w-full">
      {/* Add to Cart */}
      <motion.button
        onClick={handleAdd}
        disabled={isDisabled}
        whileTap={isDisabled ? undefined : { scale: 0.97 }}
        className={addButtonClass}
        style={{ background: status === "success" ? undefined : undefined }}
      >
        {/* Shimmer overlay */}
        {status === "idle" && sku && (
          <span
            aria-hidden
            className="pointer-events-none absolute inset-0 rounded-2xl overflow-hidden"
          >
            <span className="absolute inset-0 -translate-x-full animate-[shimmer_2.4s_infinite] bg-gradient-to-r from-transparent via-white/20 to-transparent" />
          </span>
        )}

        {status === "loading" && <Loader2 size={16} className="animate-spin" />}
        {status === "success" && <CheckCircle2 size={16} />}
        {status === "idle" && <ShoppingCart size={16} />}

        {status === "loading" && "Adding…"}
        {status === "success" && "Added to Cart!"}
        {status === "idle" && "Add to Cart"}
      </motion.button>

      {/* Buy Now */}
      <motion.button
        onClick={handleBuyNow}
        disabled={!sku}
        whileTap={!sku ? undefined : { scale: 0.97 }}
        className={[
          "flex-1 flex items-center justify-center gap-2",
          "font-bold rounded-2xl h-12 w-full text-white text-sm",
          "bg-[#0F0A04] hover:bg-[#1f1408] transition-colors duration-300",
          !sku ? "opacity-50 cursor-not-allowed" : "cursor-pointer",
        ].join(" ")}
      >
        <Zap size={16} fill="#fff" stroke="#fff" />
        Buy Now
      </motion.button>
    </div>
  );
}
