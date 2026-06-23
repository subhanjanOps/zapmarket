"use client";
import { useRouter } from "next/navigation";
import { useCartStore } from "@/lib/cart";
import type { Sku } from "./SkuSelector";

interface Props { sku: Sku | null; productName: string; productImage: string; }

export default function AddToCartButton({ sku, productName, productImage }: Props) {
  const addItem = useCartStore((s) => s.addItem);
  const router = useRouter();

  function handleAdd() {
    if (!sku) return;
    addItem({ skuId: sku.id, name: `${productName} — ${sku.sku_code}`, image: productImage, price: sku.price_amount, currency: sku.currency });
  }

  function handleBuyNow() {
    handleAdd();
    router.push("/checkout");
  }

  return (
    <div className="flex gap-3 mt-4">
      <button onClick={handleAdd} disabled={!sku}
        className="flex-1 bg-[#FF9900] text-black font-semibold py-3 rounded hover:bg-[#e68900] disabled:opacity-50 disabled:cursor-not-allowed">
        Add to Cart
      </button>
      <button onClick={handleBuyNow} disabled={!sku}
        className="flex-1 bg-[#232F3E] text-white font-semibold py-3 rounded hover:bg-[#374151] disabled:opacity-50 disabled:cursor-not-allowed">
        Buy Now
      </button>
    </div>
  );
}
