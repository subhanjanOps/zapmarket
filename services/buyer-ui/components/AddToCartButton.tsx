"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { useCartStore } from "@/lib/cart";
import { ShoppingCart, Zap, Check } from "lucide-react";
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
  const [added, setAdded] = useState(false);

  function handleAdd() {
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
    setAdded(true);
    setTimeout(() => setAdded(false), 1800);
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

  return (
    <div className="flex gap-3">
      <button
        onClick={handleAdd}
        disabled={!sku}
        className="flex-1 flex items-center justify-center gap-2 font-bold py-3.5 rounded-xl
                   text-white transition-all duration-200
                   hover:scale-[1.02] hover:shadow-lg active:scale-[0.97]
                   disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100 cursor-pointer"
        style={{
          background: added ? "#00736A" : "#FF2D78",
          fontFamily: "var(--font-syne)",
          boxShadow: added ? "0 4px 14px rgba(0,115,106,0.3)" : "0 4px 14px rgba(255,45,120,0.25)",
        }}
      >
        {added ? <Check size={16} /> : <ShoppingCart size={16} />}
        {added ? "Added to cart!" : "Add to Cart"}
      </button>
      <button
        onClick={handleBuyNow}
        disabled={!sku}
        className="flex-1 flex items-center justify-center gap-2 font-bold py-3.5 rounded-xl
                   text-white transition-all duration-200
                   hover:scale-[1.02] hover:shadow-lg active:scale-[0.97]
                   disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100 cursor-pointer"
        style={{
          background: "#00736A",
          fontFamily: "var(--font-syne)",
          boxShadow: "0 4px 14px rgba(0,115,106,0.25)",
        }}
      >
        <Zap size={16} fill="#fff" stroke="#fff" />
        Buy Now
      </button>
    </div>
  );
}
