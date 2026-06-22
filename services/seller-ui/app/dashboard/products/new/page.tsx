"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { createProduct, createSku, uploadImage } from "@/lib/api";
import CategoryPicker from "@/app/components/CategoryPicker";
import { SKUEditor, SKUDraft } from "@/app/components/SKUEditor";
import { ImageDropzone } from "@/app/components/ImageDropzone";
import { useCurrency } from "@/lib/currency";

const SLUG_RE = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

function slugify(s: string) {
  return s.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
}

const inputStyle: React.CSSProperties = {
  padding: "0.46875rem 0.875rem", borderRadius: 7, border: "1px solid var(--border)",
  background: "var(--surface2)", color: "var(--text)", fontSize: "0.8125rem",
  fontFamily: "inherit", width: "100%", outline: "none",
};

export default function NewProductPage() {
  const router = useRouter();
  const { currencies } = useCurrency();

  function toCents(amount: number, currencyCode: string): number {
    const decimals = currencies.find((c) => c.code === currencyCode)?.decimals ?? 2;
    return decimals === 0 ? Math.round(amount) : Math.round(amount * 100);
  }

  const [name, setName]               = useState("");
  const [slug, setSlug]               = useState("");
  const [slugManual, setSlugManual]   = useState(false);
  const [slugError, setSlugError]     = useState("");
  const [description, setDescription] = useState("");
  const [categoryId, setCategoryId]   = useState("");
  const [status, setStatus]           = useState<"DRAFT" | "ACTIVE">("DRAFT");
  const [skus, setSkus]               = useState<SKUDraft[]>([]);
  const [productId, setProductId]     = useState<string | null>(null);
  const [saving, setSaving]           = useState(false);
  const [error, setError]             = useState("");
  const [step, setStep]               = useState<"info" | "skus" | "images" | "publish">("info");

  function handleNameChange(v: string) {
    setName(v);
    if (!slugManual) setSlug(slugify(v));
  }

  function handleSlugBlur() {
    const formatted = slugify(slug);
    setSlug(formatted);
    setSlugError(formatted && !SLUG_RE.test(formatted) ? "Slug must be lowercase letters, numbers, and hyphens only" : "");
  }

  async function saveBasicInfo() {
    if (!name.trim()) { setError("Product name is required"); return; }
    const finalSlug = slug || slugify(name);
    if (!SLUG_RE.test(finalSlug)) { setError("Slug contains invalid characters"); return; }
    setSaving(true); setError("");
    try {
      const p = await createProduct({ name: name.trim(), slug: finalSlug, description, category_id: categoryId, status: "DRAFT" });
      setProductId(p.id);
      setStep("skus");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to create product");
    } finally {
      setSaving(false);
    }
  }

  async function saveSkus() {
    if (!productId) return;
    setSaving(true); setError("");
    const validSkus = skus.filter((s) => s.sku_code.trim());
    try {
      const results = await Promise.allSettled(
        validSkus.map((s) => {
          const attrs = Object.fromEntries(s.attributes.filter((a) => a.key).map((a) => [a.key, a.value]));
          return createSku({
            product_id: productId,
            sku_code: s.sku_code,
            attributes: attrs,
            price_amount: toCents(s.price_amount, s.price_currency),
            price_currency: s.price_currency,
            compare_price: s.compare_price ? toCents(s.compare_price, s.price_currency) : undefined,
            weight_grams: s.weight_grams || undefined,
            is_active: s.is_active,
          });
        }),
      );
      const failures = results.filter((r): r is PromiseRejectedResult => r.status === "rejected");
      if (failures.length > 0) {
        setError(`${failures.length} SKU(s) failed to save: ${failures[0].reason?.message ?? "Unknown error"}`);
        setSaving(false);
        return;
      }
      setStep("images");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save SKUs");
    } finally {
      setSaving(false);
    }
  }

  async function handleImageUpload(file: File) {
    if (!productId) throw new Error("No product created yet — complete basic info first");
    await uploadImage(productId, file);
  }

  async function publish() {
    if (!productId) return;
    setSaving(true); setError("");
    try {
      const { updateProduct } = await import("@/lib/api");
      await updateProduct(productId, { status });
      router.push(`/dashboard/products/${productId}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to publish");
      setSaving(false);
    }
  }

  const steps = ["info", "skus", "images", "publish"] as const;
  const stepIdx = steps.indexOf(step);

  return (
    <>
      <div className="page-header">
        <div>
          <h1 className="page-title">New Product</h1>
          <p className="page-subtitle">Fill in the details below to list your product</p>
        </div>
      </div>

      {/* Step indicator */}
      <div className="timeline" style={{ marginBottom: "2rem", maxWidth: "32rem" }}>
        {steps.map((s, i) => {
          const labels: Record<string, string> = { info: "Basic Info", skus: "SKUs", images: "Images", publish: "Publish" };
          const done   = i < stepIdx;
          const active = i === stepIdx;
          return (
            <div key={s} className={`timeline-step${done ? " done" : ""}`}>
              <div className={`timeline-dot${active ? " active" : done ? " done" : ""}`}>{done ? "✓" : i + 1}</div>
              <span className={`timeline-label${active ? " active" : done ? " done" : ""}`}>{labels[s]}</span>
            </div>
          );
        })}
      </div>

      {error && <div style={{ background: "color-mix(in srgb, var(--danger) 10%, transparent)", border: "1px solid color-mix(in srgb, var(--danger) 25%, transparent)", borderRadius: 7, padding: "0.625rem 0.875rem", fontSize: "0.8125rem", color: "var(--danger)", marginBottom: "1rem" }}>{error}</div>}

      {step === "info" && (
        <div className="card" style={{ maxWidth: "42rem" }}>
          <div style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
            <div>
              <label className="form-label" htmlFor="new-name">Product Name *</label>
              <input id="new-name" style={inputStyle} value={name} onChange={(e) => handleNameChange(e.target.value)} placeholder="e.g. Leather Chelsea Boots" />
            </div>
            <div>
              <label className="form-label" htmlFor="new-slug">Slug (URL path)</label>
              <input
                id="new-slug"
                style={{ ...inputStyle, fontFamily: '"DM Mono", monospace' }}
                value={slug}
                onChange={(e) => { setSlug(e.target.value); setSlugManual(true); setSlugError(""); }}
                onBlur={handleSlugBlur}
                placeholder="auto-generated from name"
              />
              {slugError && <p style={{ margin: "0.25rem 0 0", fontSize: "0.75rem", color: "var(--danger)" }}>{slugError}</p>}
            </div>
            <div>
              <label className="form-label" htmlFor="new-desc">Description</label>
              <textarea id="new-desc" style={{ ...inputStyle, minHeight: "6rem", resize: "vertical" }} value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Describe your product…" />
            </div>
            <div>
              <label className="form-label">Category</label>
              <CategoryPicker value={categoryId} onChange={setCategoryId} />
            </div>
            <div style={{ display: "flex", justifyContent: "flex-end" }}>
              <button className="btn btn-primary" onClick={saveBasicInfo} disabled={saving}>
                {saving ? "Saving…" : "Next: SKUs →"}
              </button>
            </div>
          </div>
        </div>
      )}

      {step === "skus" && (
        <div style={{ maxWidth: "52rem" }}>
          <SKUEditor onChange={setSkus} />
          <div style={{ display: "flex", justifyContent: "space-between", marginTop: "1rem" }}>
            <button className="btn btn-ghost" onClick={() => setStep("info")}>← Back</button>
            <button className="btn btn-primary" onClick={saveSkus} disabled={saving}>
              {saving ? "Saving…" : "Next: Images →"}
            </button>
          </div>
        </div>
      )}

      {step === "images" && (
        <div className="card" style={{ maxWidth: "42rem" }}>
          <h3 style={{ margin: "0 0 1rem", fontSize: "0.9375rem", fontWeight: 600 }}>Product Images</h3>
          <ImageDropzone onUpload={handleImageUpload} />
          <div style={{ display: "flex", justifyContent: "space-between", marginTop: "1.5rem" }}>
            <button className="btn btn-ghost" onClick={() => setStep("skus")}>← Back</button>
            <button className="btn btn-primary" onClick={() => setStep("publish")}>Next: Publish →</button>
          </div>
        </div>
      )}

      {step === "publish" && (
        <div className="card" style={{ maxWidth: "32rem" }}>
          <h3 style={{ margin: "0 0 1rem", fontSize: "0.9375rem", fontWeight: 600 }}>Publish</h3>
          <div style={{ marginBottom: "1.5rem" }}>
            <label className="form-label" htmlFor="new-status">Status</label>
            <select id="new-status" style={inputStyle} value={status} onChange={(e) => setStatus(e.target.value as "DRAFT" | "ACTIVE")}>
              <option value="DRAFT">Draft — not publicly listed</option>
              <option value="ACTIVE">Active — live on marketplace</option>
            </select>
          </div>
          <div style={{ display: "flex", justifyContent: "space-between" }}>
            <button className="btn btn-ghost" onClick={() => setStep("images")}>← Back</button>
            <button className="btn btn-primary" onClick={publish} disabled={saving}>
              {saving ? "Saving…" : "Save Product"}
            </button>
          </div>
        </div>
      )}
    </>
  );
}
