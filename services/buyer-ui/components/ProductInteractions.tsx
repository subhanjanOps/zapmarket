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
        <p className="text-2xl font-bold text-gray-900 mb-4">
          {selectedSku.currency} {(selectedSku.price_amount / 100).toFixed(2)}
        </p>
      )}
      <SkuSelector skus={skus} onSelect={setSelectedSku} />
      <AddToCartButton sku={selectedSku} productName={productName} productImage={productImage} />
    </div>
  );
}
