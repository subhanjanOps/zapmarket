"use client";
import { useState } from "react";
import { Check } from "lucide-react";

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

const COLOR_MAP: Record<string, string> = {
  red: "#EF4444", blue: "#3B82F6", green: "#22C55E", black: "#1A1208",
  white: "#F9FAFB", yellow: "#EAB308", pink: "#EC4899", purple: "#8B5CF6",
  orange: "#F97316", gray: "#6B7280", grey: "#6B7280", brown: "#92400E",
  navy: "#1E3A5F", teal: "#00736A",
};

function isColorAttribute(key: string): boolean {
  return ["color", "colour", "shade"].includes(key.toLowerCase());
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
      <div>
        <p className="text-xs font-bold uppercase tracking-wider mb-2.5" style={{ color: "#6B6052" }}>
          Option
        </p>
        <div className="flex gap-2 flex-wrap">
          {skus.map((s) => {
            const active = s.id === selectedId;
            return (
              <button
                key={s.id}
                onClick={() => select(s.id)}
                disabled={!s.is_active}
                className="relative px-4 py-2 rounded-xl text-sm font-semibold cursor-pointer
                           transition-all duration-150 hover:-translate-y-px active:scale-95
                           disabled:opacity-35 disabled:cursor-not-allowed disabled:hover:translate-y-0"
                style={
                  active
                    ? { background: "#FF2D78", color: "#fff", border: "2px solid #FF2D78", boxShadow: "0 2px 8px rgba(255,45,120,0.3)" }
                    : { background: "#fff", color: "#1A1208", border: "2px solid #F0EDE8" }
                }
              >
                {active && <Check size={12} className="inline mr-1" />}
                {s.sku_code}
              </button>
            );
          })}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {attributeKeys.map((key) => {
        const values = [...new Set(skus.map((s) => s.attributes?.[key]).filter(Boolean))] as string[];
        const isColor = isColorAttribute(key);

        return (
          <div key={key}>
            <p className="text-xs font-bold uppercase tracking-wider mb-2.5" style={{ color: "#6B6052" }}>
              {key}
            </p>
            <div className="flex gap-2 flex-wrap">
              {values.map((val) => {
                const sku = skus.find((s) => s.attributes?.[key] === val);
                const active = sku?.id === selectedId;
                const disabled = !sku?.is_active;

                if (isColor) {
                  const hex = COLOR_MAP[val.toLowerCase()] ?? "#9CA3AF";
                  return (
                    <button
                      key={val}
                      onClick={() => sku && select(sku.id)}
                      disabled={disabled}
                      title={val}
                      className="w-9 h-9 rounded-full cursor-pointer transition-all duration-150
                                 disabled:opacity-35 disabled:cursor-not-allowed hover:scale-110 active:scale-95"
                      style={{
                        background: hex,
                        border: active ? "3px solid #FF2D78" : "3px solid transparent",
                        boxShadow: active
                          ? "0 0 0 2px #fff, 0 0 0 4px #FF2D78"
                          : "0 0 0 2px #F0EDE8",
                      }}
                      aria-label={val}
                    >
                      {active && (
                        <Check
                          size={14}
                          className="mx-auto"
                          style={{ color: val.toLowerCase() === "white" ? "#1A1208" : "#fff" }}
                        />
                      )}
                    </button>
                  );
                }

                return (
                  <button
                    key={val}
                    onClick={() => sku && select(sku.id)}
                    disabled={disabled}
                    className="px-4 py-2 rounded-xl text-sm font-semibold cursor-pointer
                               transition-all duration-150 hover:-translate-y-px active:scale-95
                               disabled:opacity-35 disabled:cursor-not-allowed disabled:hover:translate-y-0"
                    style={
                      active
                        ? { background: "#FF2D78", color: "#fff", border: "2px solid #FF2D78", boxShadow: "0 2px 8px rgba(255,45,120,0.3)" }
                        : { background: "#fff", color: "#1A1208", border: "2px solid #F0EDE8" }
                    }
                  >
                    {active && <Check size={12} className="inline mr-1" />}
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
