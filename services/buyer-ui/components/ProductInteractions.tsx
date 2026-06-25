"use client";
import { useState } from "react";
import SkuSelector, { type Sku } from "./SkuSelector";
import AddToCartButton from "./AddToCartButton";
import { Truck, RotateCcw, ShieldCheck, Minus, Plus, Star } from "lucide-react";

interface Props {
  skus: Sku[];
  productName: string;
  productImage: string;
}

const TRUST = [
  { icon: Truck,       label: "Free delivery",   sub: "On orders above ₹499" },
  { icon: RotateCcw,   label: "Easy returns",    sub: "7-day return policy" },
  { icon: ShieldCheck, label: "Secure payment",  sub: "100% protected" },
];

export default function ProductInteractions({ skus, productName, productImage }: Props) {
  const [selectedSku, setSelectedSku] = useState<Sku | null>(skus[0] ?? null);
  const [qty, setQty] = useState(1);

  const price = selectedSku ? (selectedSku.price_amount / 100).toFixed(2) : null;
  const currency = selectedSku?.currency ?? "INR";
  const symbol = currency === "INR" ? "₹" : currency;

  return (
    <div className="space-y-5">
      {/* Rating strip */}
      <div className="flex items-center gap-2">
        <div className="flex">
          {[1, 2, 3, 4, 5].map((n) => (
            <Star
              key={n}
              size={14}
              fill={n <= 4 ? "#FF8C00" : "none"}
              stroke="#FF8C00"
            />
          ))}
        </div>
        <span className="text-xs font-semibold" style={{ color: "#FF8C00" }}>4.0</span>
        <span className="text-xs" style={{ color: "#9CA3AF" }}>· 24 reviews</span>
      </div>

      {/* Price */}
      {price && (
        <div>
          <p
            className="text-4xl font-extrabold leading-none"
            style={{ color: "#FF2D78", fontFamily: "var(--font-syne)", fontVariantNumeric: "tabular-nums" }}
          >
            {symbol} {price}
          </p>
          <p className="text-xs mt-1" style={{ color: "#9CA3AF" }}>Inclusive of all taxes</p>
        </div>
      )}

      {/* Divider */}
      <div style={{ borderTop: "1px solid #F0EDE8" }} />

      {/* SKU Selector */}
      <SkuSelector skus={skus} onSelect={setSelectedSku} />

      {/* Quantity */}
      <div>
        <p className="text-xs font-bold uppercase tracking-wider mb-2.5" style={{ color: "#6B6052" }}>
          Quantity
        </p>
        <div className="flex items-center gap-0">
          <button
            onClick={() => setQty((q) => Math.max(1, q - 1))}
            className="w-10 h-10 flex items-center justify-center rounded-l-xl cursor-pointer
                       transition-all duration-150 hover:bg-gray-50 active:scale-90"
            style={{ border: "2px solid #F0EDE8", borderRight: "none" }}
            aria-label="Decrease quantity"
          >
            <Minus size={14} />
          </button>
          <div
            className="w-12 h-10 flex items-center justify-center text-sm font-bold"
            style={{
              border: "2px solid #F0EDE8",
              borderLeft: "none",
              borderRight: "none",
              color: "#1A1208",
              fontVariantNumeric: "tabular-nums",
            }}
          >
            {qty}
          </div>
          <button
            onClick={() => setQty((q) => q + 1)}
            className="w-10 h-10 flex items-center justify-center rounded-r-xl cursor-pointer
                       transition-all duration-150 hover:bg-gray-50 active:scale-90"
            style={{ border: "2px solid #F0EDE8", borderLeft: "none" }}
            aria-label="Increase quantity"
          >
            <Plus size={14} />
          </button>
        </div>
      </div>

      {/* CTA */}
      <AddToCartButton sku={selectedSku} productName={productName} productImage={productImage} qty={qty} />

      {/* Trust badges */}
      <div
        className="grid grid-cols-3 gap-3 pt-4"
        style={{ borderTop: "1px solid #F0EDE8" }}
      >
        {TRUST.map(({ icon: Icon, label, sub }) => (
          <div key={label} className="flex flex-col items-center text-center gap-1.5">
            <div
              className="w-9 h-9 rounded-xl flex items-center justify-center shrink-0"
              style={{ background: "#E6F4F2" }}
            >
              <Icon size={16} style={{ color: "#00736A" }} />
            </div>
            <p className="text-[11px] font-bold leading-tight" style={{ color: "#1A1208" }}>{label}</p>
            <p className="text-[10px] leading-tight" style={{ color: "#9CA3AF" }}>{sub}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
