"use client";
import { useState } from "react";
import SkuSelector, { type Sku } from "./SkuSelector";
import AddToCartButton from "./AddToCartButton";

interface Props {
  skus: Sku[];
  productName: string;
  productImage: string;
}

export default function ProductInteractions({ skus, productName, productImage }: Props) {
  const [selectedSku, setSelectedSku] = useState<Sku | null>(skus[0] ?? null);

  return (
    <div>
      {selectedSku && (
        <p className="text-3xl font-extrabold mb-4" style={{ color: "#FF2D78", fontFamily: "var(--font-syne)", fontVariantNumeric: "tabular-nums" }}>
          {selectedSku.currency} {(selectedSku.price_amount / 100).toFixed(2)}
        </p>
      )}
      <SkuSelector skus={skus} onSelect={setSelectedSku} />
      <AddToCartButton sku={selectedSku} productName={productName} productImage={productImage} />
    </div>
  );
}
