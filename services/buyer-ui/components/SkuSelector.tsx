"use client";
import { useState } from "react";

export interface Sku {
  id: string;
  sku_code: string;
  price_amount: number;
  currency: string;
  is_active: boolean;
  attributes?: Record<string, string>;
}

interface SkuSelectorProps {
  skus: Sku[];
  onSelect: (sku: Sku) => void;
}

export default function SkuSelector({ skus, onSelect }: SkuSelectorProps) {
  const [selectedId, setSelectedId] = useState<string>(skus[0]?.id ?? "");

  const attributeKeys = skus.length > 0 && skus[0].attributes
    ? Object.keys(skus[0].attributes)
    : [];

  function select(id: string) {
    setSelectedId(id);
    const sku = skus.find((s) => s.id === id);
    if (sku) onSelect(sku);
  }

  if (attributeKeys.length === 0) {
    return (
      <div className="flex gap-2 flex-wrap">
        {skus.map((s) => (
          <button key={s.id} onClick={() => select(s.id)}
            className={`px-3 py-1.5 rounded-lg text-sm cursor-pointer transition-colors ${s.id === selectedId ? "font-semibold text-white" : "hover:border-[#FF2D78]"} ${!s.is_active ? "opacity-40 cursor-not-allowed" : ""}`}
            style={s.id === selectedId ? { background: "#FF2D78", border: "2px solid #FF2D78" } : { border: "2px solid #F0EDE8" }}
            disabled={!s.is_active}>
            {s.sku_code}
          </button>
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {attributeKeys.map((key) => {
        const values = [...new Set(skus.map((s) => s.attributes?.[key]).filter(Boolean))] as string[];
        return (
          <div key={key}>
            <p className="text-xs font-semibold mb-2 uppercase tracking-wide" style={{ color: "#6B6052" }}>{key}</p>
            <div className="flex gap-2 flex-wrap">
              {values.map((val) => {
                const sku = skus.find((s) => s.attributes?.[key] === val);
                const active = sku?.id === selectedId;
                return (
                  <button key={val} onClick={() => sku && select(sku.id)}
                    className={`px-3 py-1.5 rounded-lg text-sm cursor-pointer transition-colors ${active ? "font-semibold text-white" : "hover:border-[#FF2D78]"}`}
                    style={active ? { background: "#FF2D78", border: "2px solid #FF2D78" } : { border: "2px solid #F0EDE8" }}>
                    {val}
                  </button>
                );
              })}
            </div>
          </div>
        );
      })}
    </div>
  );
}
