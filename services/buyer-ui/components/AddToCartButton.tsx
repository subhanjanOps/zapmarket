"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { useCartStore } from "@/lib/cart";
import { ShoppingCart, Zap, Check } from "lucide-react";
import { Button } from "@/components/ui/button";
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
      <Button
        onClick={handleAdd}
        disabled={!sku}
        size="lg"
        className={
          added
            ? "flex-1 gap-2 bg-green-600 hover:bg-green-700 text-white"
            : "flex-1 gap-2 bg-primary hover:bg-primary/90 text-white"
        }
      >
        {added ? <Check size={16} /> : <ShoppingCart size={16} />}
        {added ? "Added to cart!" : "Add to Cart"}
      </Button>
      <Button
        onClick={handleBuyNow}
        disabled={!sku}
        size="lg"
        className="flex-1 gap-2 bg-[#00736A] hover:bg-[#005c54] text-white"
      >
        <Zap size={16} fill="#fff" stroke="#fff" />
        Buy Now
      </Button>
    </div>
  );
}
