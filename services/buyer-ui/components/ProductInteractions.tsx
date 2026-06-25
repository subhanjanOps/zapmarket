"use client";
import { useState } from "react";
import SkuSelector, { type Sku } from "./SkuSelector";
import AddToCartButton from "./AddToCartButton";
import { Truck, RotateCcw, ShieldCheck, Minus, Plus } from "lucide-react";

interface Props {
  skus: Sku[];
  productName: string;
  productImage: string;
}

const TRUST = [
  { icon: Truck,       label: "Free delivery",  sub: "On orders above ₹499" },
  { icon: RotateCcw,   label: "Easy returns",   sub: "7-day return policy" },
  { icon: ShieldCheck, label: "Secure payment", sub: "100% protected" },
];

export default function ProductInteractions({ skus, productName, productImage }: Props) {
  const [selectedSku, setSelectedSku] = useState<Sku | null>(skus[0] ?? null);
  const [qty, setQty] = useState(1);

  const price = selectedSku ? (selectedSku.price_amount / 100).toFixed(2) : null;
  const symbol = selectedSku?.currency === "INR" ? "₹" : (selectedSku?.currency ?? "₹");

  return (
    <div className="space-y-5">
      {/* Price */}
      {price && (
        <div>
          <p
            className="text-4xl font-extrabold leading-none"
            style={{ color: "#E91E8C", fontFamily: "var(--font-syne)", fontVariantNumeric: "tabular-nums" }}
          >
            {symbol} {price}
          </p>
          <p className="text-xs mt-1.5" style={{ color: "#B8A898" }}>Inclusive of all taxes</p>
        </div>
      )}

      <div style={{ borderTop: "1px solid #EDE9E3" }} />

      {/* SKU Selector */}
      <SkuSelector skus={skus} onSelect={setSelectedSku} />

      {/* Quantity */}
      <div>
        <p className="text-xs font-bold uppercase tracking-wider mb-2.5" style={{ color: "#7A6856" }}>
          Quantity
        </p>
        <div className="flex items-center gap-0">
          <button
            onClick={() => setQty((q) => Math.max(1, q - 1))}
            className="w-10 h-10 flex items-center justify-center rounded-l-xl cursor-pointer
                       transition-all duration-150 hover:bg-[#F3F0EB] active:scale-95"
            style={{ border: "2px solid #EDE9E3", borderRight: "none" }}
            aria-label="Decrease quantity"
          >
            <Minus size={14} />
          </button>
          <div
            className="w-12 h-10 flex items-center justify-center text-sm font-bold"
            style={{ border: "2px solid #EDE9E3", borderLeft: "none", borderRight: "none", color: "#0F0A04", fontVariantNumeric: "tabular-nums" }}
          >
            {qty}
          </div>
          <button
            onClick={() => setQty((q) => q + 1)}
            className="w-10 h-10 flex items-center justify-center rounded-r-xl cursor-pointer
                       transition-all duration-150 hover:bg-[#F3F0EB] active:scale-95"
            style={{ border: "2px solid #EDE9E3", borderLeft: "none" }}
            aria-label="Increase quantity"
          >
            <Plus size={14} />
          </button>
        </div>
      </div>

      {/* CTA */}
      <AddToCartButton sku={selectedSku} productName={productName} productImage={productImage} qty={qty} />

      {/* Trust badges */}
      <div className="grid grid-cols-3 gap-3 pt-4" style={{ borderTop: "1px solid #EDE9E3" }}>
        {TRUST.map(({ icon: Icon, label, sub }) => (
          <div key={label} className="flex flex-col items-center text-center gap-1.5">
            <div className="w-9 h-9 rounded-xl flex items-center justify-center shrink-0" style={{ background: "#F0FDF4" }}>
              <Icon size={16} style={{ color: "#00736A" }} />
            </div>
            <p className="text-[11px] font-bold leading-tight" style={{ color: "#0F0A04" }}>{label}</p>
            <p className="text-[10px] leading-tight" style={{ color: "#B8A898" }}>{sub}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
