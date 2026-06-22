"use client";
import { useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { SKU } from "@/lib/api";
import { useCurrency } from "@/lib/currency";

export interface SKUDraft {
  _key: string;
  id?: string;
  sku_code: string;
  price_amount: number;
  price_currency: string;
  compare_price: number;
  weight_grams: number;
  is_active: boolean;
  attributes: { key: string; value: string }[];
}

const skuInputStyle: React.CSSProperties = { padding: "0.375rem 0.625rem", borderRadius: 6, border: "1px solid var(--border)", background: "var(--surface2)", color: "var(--text)", fontSize: "0.8125rem", fontFamily: "inherit", width: "100%" };

function emptyDraft(currency = "USD"): SKUDraft {
  return {
    _key: crypto.randomUUID(),
    sku_code: "",
    price_amount: 0,
    price_currency: currency,
    compare_price: 0,
    weight_grams: 0,
    is_active: true,
    attributes: [],
  };
}

export function skuToAttributes(sku: SKU): { key: string; value: string }[] {
  return Object.entries(sku.attributes ?? {}).map(([key, value]) => ({ key, value }));
}

interface Props {
  initial?: SKUDraft[];
  onChange: (skus: SKUDraft[]) => void;
}

export function SKUEditor({ initial = [], onChange }: Props) {
  const { currency, currencies } = useCurrency();
  const [skus, setSkus] = useState<SKUDraft[]>(initial.length ? initial : [emptyDraft(currency)]);

  function decimalsFor(code: string): number {
    return currencies.find((c) => c.code === code)?.decimals ?? 2;
  }

  function update(idx: number, patch: Partial<SKUDraft>) {
    const next = skus.map((s, i) => i === idx ? { ...s, ...patch } : s);
    setSkus(next);
    onChange(next);
  }

  function addAttr(skuIdx: number) {
    update(skuIdx, { attributes: [...skus[skuIdx].attributes, { key: "", value: "" }] });
  }

  function updateAttr(skuIdx: number, attrIdx: number, field: "key" | "value", val: string) {
    const attrs = skus[skuIdx].attributes.map((a, i) => i === attrIdx ? { ...a, [field]: val } : a);
    update(skuIdx, { attributes: attrs });
  }

  function removeAttr(skuIdx: number, attrIdx: number) {
    update(skuIdx, { attributes: skus[skuIdx].attributes.filter((_, i) => i !== attrIdx) });
  }

  function addSku() {
    const next = [...skus, emptyDraft(currency)];
    setSkus(next);
    onChange(next);
  }

  function removeSku(idx: number) {
    const next = skus.filter((_, i) => i !== idx);
    setSkus(next);
    onChange(next);
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
      {skus.map((sku, si) => (
        <div key={sku._key} className="card" style={{ padding: "1rem 1.25rem" }}>
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "0.875rem" }}>
            <span style={{ fontSize: "0.8125rem", fontWeight: 500, color: "var(--text-2)" }}>SKU #{si + 1}</span>
            {skus.length > 1 && (
              <button onClick={() => removeSku(si)} style={{ background: "none", border: "none", cursor: "pointer", color: "var(--danger)", display: "flex", padding: 0 }}>
                <Trash2 size={14} />
              </button>
            )}
          </div>

          <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: "0.75rem", marginBottom: "0.75rem" }}>
            <div>
              <label style={{ fontSize: "0.75rem", color: "var(--text-2)", fontWeight: 500, display: "block", marginBottom: "0.25rem" }}>SKU Code *</label>
              <input style={skuInputStyle} value={sku.sku_code} onChange={(e) => update(si, { sku_code: e.target.value })} placeholder="e.g. TSHIRT-BLK-M" />
            </div>
            <div>
              <label style={{ fontSize: "0.75rem", color: "var(--text-2)", fontWeight: 500, display: "block", marginBottom: "0.25rem" }}>
                Price *
                <select
                  value={sku.price_currency}
                  onChange={(e) => update(si, { price_currency: e.target.value })}
                  style={{ marginLeft: "0.375rem", fontSize: "0.75rem", background: "var(--surface)", border: "1px solid var(--border)", borderRadius: 4, color: "var(--accent)", padding: "0 0.25rem", cursor: "pointer" }}
                >
                  {currencies.length > 0
                    ? currencies.map((c) => <option key={c.code} value={c.code}>{c.code}</option>)
                    : <option value={sku.price_currency}>{sku.price_currency}</option>}
                </select>
              </label>
              <input
                style={skuInputStyle}
                type="number"
                min="0"
                step={decimalsFor(sku.price_currency) === 0 ? "1" : "0.01"}
                value={sku.price_amount || ""}
                onChange={(e) => update(si, { price_amount: parseFloat(e.target.value) || 0 })}
                placeholder={decimalsFor(sku.price_currency) === 0 ? "0" : "0.00"}
              />
            </div>
            <div>
              <label style={{ fontSize: "0.75rem", color: "var(--text-2)", fontWeight: 500, display: "block", marginBottom: "0.25rem" }}>Compare Price ({sku.price_currency})</label>
              <input
                style={skuInputStyle}
                type="number"
                min="0"
                step={decimalsFor(sku.price_currency) === 0 ? "1" : "0.01"}
                value={sku.compare_price || ""}
                onChange={(e) => update(si, { compare_price: parseFloat(e.target.value) || 0 })}
                placeholder={decimalsFor(sku.price_currency) === 0 ? "0" : "0.00"}
              />
            </div>
            <div>
              <label style={{ fontSize: "0.75rem", color: "var(--text-2)", fontWeight: 500, display: "block", marginBottom: "0.25rem" }}>Weight (g)</label>
              <input style={skuInputStyle} type="number" min="0" value={sku.weight_grams || ""} onChange={(e) => update(si, { weight_grams: parseInt(e.target.value) || 0 })} placeholder="0" />
            </div>
            <div style={{ display: "flex", alignItems: "flex-end", gap: "0.5rem" }}>
              <label style={{ display: "flex", alignItems: "center", gap: "0.5rem", cursor: "pointer", paddingBottom: "0.375rem" }}>
                <input type="checkbox" checked={sku.is_active} onChange={(e) => update(si, { is_active: e.target.checked })} />
                <span style={{ fontSize: "0.8125rem", color: "var(--text-2)" }}>Active</span>
              </label>
            </div>
          </div>

          {/* Variant attributes */}
          <div style={{ marginTop: "0.5rem" }}>
            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "0.5rem" }}>
              <span style={{ fontSize: "0.75rem", color: "var(--muted)", fontWeight: 500 }}>Variant Attributes</span>
              <button onClick={() => addAttr(si)} className="btn btn-ghost" style={{ fontSize: "0.75rem", padding: "0.25rem 0.5rem", gap: "0.25rem" }}>
                <Plus size={11} />Add
              </button>
            </div>
            {sku.attributes.map((attr, ai) => (
              <div key={ai} style={{ display: "flex", gap: "0.5rem", marginBottom: "0.375rem", alignItems: "center" }}>
                <input style={{ ...skuInputStyle, flex: 1 }} value={attr.key} onChange={(e) => updateAttr(si, ai, "key", e.target.value)} placeholder="e.g. Color" />
                <input style={{ ...skuInputStyle, flex: 1 }} value={attr.value} onChange={(e) => updateAttr(si, ai, "value", e.target.value)} placeholder="e.g. Black" />
                <button onClick={() => removeAttr(si, ai)} style={{ background: "none", border: "none", cursor: "pointer", color: "var(--danger)", flexShrink: 0, display: "flex" }}>
                  <Trash2 size={13} />
                </button>
              </div>
            ))}
          </div>
        </div>
      ))}

      <button onClick={addSku} className="btn btn-ghost" style={{ alignSelf: "flex-start" }}>
        <Plus size={13} /> Add SKU
      </button>
    </div>
  );
}
