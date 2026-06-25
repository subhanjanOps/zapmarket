"use client";
import { useState } from "react";
import { Check } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

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

  const attributeKeys =
    skus.length > 0 && skus[0].attributes ? Object.keys(skus[0].attributes) : [];

  function select(id: string) {
    setSelectedId(id);
    const sku = skus.find((s) => s.id === id);
    if (sku) onSelect(sku);
  }

  const selectedSku = skus.find((s) => s.id === selectedId);

  const renderNoAttributeSkus = () => (
    <div>
      <p className="text-xs font-bold uppercase tracking-wider mb-2.5 text-muted-foreground">
        Option
      </p>
      <div className="flex gap-2 flex-wrap">
        {skus.map((s) => {
          const active = s.id === selectedId;
          return (
            <Button
              key={s.id}
              onClick={() => select(s.id)}
              disabled={!s.is_active}
              variant={active ? "default" : "outline"}
              className={cn(
                "rounded-xl text-sm font-semibold transition-all duration-150",
                "hover:-translate-y-px active:scale-95",
                "disabled:opacity-35 disabled:hover:translate-y-0"
              )}
            >
              {active && <Check size={12} className="mr-1" />}
              {s.sku_code}
            </Button>
          );
        })}
      </div>
    </div>
  );

  const renderAttributeGroups = () => (
    <div className="space-y-4">
      {attributeKeys.map((key) => {
        const values = [
          ...new Set(skus.map((s) => s.attributes?.[key]).filter(Boolean)),
        ] as string[];
        const isColor = isColorAttribute(key);

        return (
          <div key={key}>
            <p className="text-xs font-bold uppercase tracking-wider mb-2.5 text-muted-foreground">
              {key}
            </p>
            <div className="flex gap-2 flex-wrap">
              {values.map((val) => {
                const sku = skus.find((s) => s.attributes?.[key] === val);
                const active = sku?.id === selectedId;
                const disabled = !sku?.is_active;

                if (isColor) {
                  const hex = COLOR_MAP[val.toLowerCase()] ?? "#9CA3AF";
                  const isLight = val.toLowerCase() === "white";
                  return (
                    <Button
                      key={val}
                      size="icon"
                      onClick={() => sku && select(sku.id)}
                      disabled={disabled}
                      title={val}
                      aria-label={val}
                      className={cn(
                        "rounded-full w-9 h-9 p-0 border-0 transition-all duration-150",
                        "hover:scale-110 active:scale-95",
                        "disabled:opacity-35 disabled:hover:scale-100",
                        active && "ring-2 ring-offset-2 ring-primary"
                      )}
                      style={{ backgroundColor: hex }}
                    >
                      {active && (
                        <Check
                          size={14}
                          style={{ color: isLight ? "#1A1208" : "#fff" }}
                        />
                      )}
                    </Button>
                  );
                }

                return (
                  <Button
                    key={val}
                    onClick={() => sku && select(sku.id)}
                    disabled={disabled}
                    variant={active ? "default" : "outline"}
                    className={cn(
                      "rounded-xl text-sm font-semibold transition-all duration-150",
                      "hover:-translate-y-px active:scale-95",
                      "disabled:opacity-35 disabled:hover:translate-y-0"
                    )}
                  >
                    {active && <Check size={12} className="mr-1" />}
                    {val}
                  </Button>
                );
              })}
            </div>
          </div>
        );
      })}
    </div>
  );

  return (
    <div className="space-y-4">
      {attributeKeys.length === 0 ? renderNoAttributeSkus() : renderAttributeGroups()}

      {selectedSku && (
        <div className="flex items-center gap-3 pt-1">
          <span className="text-primary font-bold text-lg">
            {selectedSku.currency} {selectedSku.price_amount.toFixed(2)}
          </span>
          <Badge variant={selectedSku.is_active ? "default" : "secondary"}>
            {selectedSku.is_active ? "In Stock" : "Out of Stock"}
          </Badge>
        </div>
      )}
    </div>
  );
}
