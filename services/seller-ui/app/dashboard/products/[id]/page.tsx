"use client";
import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, Save } from "lucide-react";
import { getProduct, updateProduct, getSkus, createSku, updateSku, deleteSku, getImages, uploadImage, deleteImage, getCategories, Product, SKU, ProductImage, Category } from "@/lib/api";
import { StatusBadge } from "@/app/components/StatusBadge";
import { SKUEditor, SKUDraft, skuToAttributes } from "@/app/components/SKUEditor";
import { ImageDropzone } from "@/app/components/ImageDropzone";
import { SkeletonTableCard } from "@/app/components/Skeleton";
import { useCurrency, toCents, fromCents } from "@/lib/currency";
import { Trash2 } from "lucide-react";

const inputStyle: React.CSSProperties = { padding: "0.46875rem 0.875rem", borderRadius: 7, border: "1px solid var(--border)", background: "var(--surface2)", color: "var(--text)", fontSize: "0.8125rem", fontFamily: "inherit", width: "100%", outline: "none" };

function slugify(s: string) {
  return s.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
}

export default function EditProductPage() {
  const { id } = useParams<{ id: string }>();
  const router  = useRouter();
  const { currencies, formatFrom } = useCurrency();

  const [product, setProduct]     = useState<Product | null>(null);
  const [skus, setSkus]           = useState<SKU[]>([]);
  const [images, setImages]       = useState<ProductImage[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading]     = useState(true);
  const [error, setError]         = useState("");
  const [saving, setSaving]       = useState(false);

  const [name, setName]           = useState("");
  const [slug, setSlug]           = useState("");
  const [description, setDescription] = useState("");
  const [categoryId, setCategoryId]   = useState("");
  const [status, setStatus]           = useState("DRAFT");
  const [skuDrafts, setSkuDrafts]     = useState<SKUDraft[]>([]);

  useEffect(() => {
    Promise.all([getProduct(id), getSkus(id), getImages(id), getCategories()])
      .then(([p, s, img, cats]) => {
        setProduct(p);
        setName(p.name);
        setSlug(p.slug);
        setDescription(p.description ?? "");
        setCategoryId(p.category_id ?? "");
        setStatus(p.status);
        setSkus(s.skus ?? []);
        setSkuDrafts((s.skus ?? []).map((sk) => ({
          _key: sk.id ?? crypto.randomUUID(),
          id: sk.id,
          sku_code: sk.sku_code,
          price_amount: fromCents(sk.price_amount, sk.price_currency, currencies),
          price_currency: sk.price_currency,
          compare_price: fromCents(sk.compare_price ?? 0, sk.price_currency, currencies),
          weight_grams: sk.weight_grams ?? 0,
          is_active: sk.is_active,
          attributes: skuToAttributes(sk),
        })));
        setImages(img.images ?? []);
        setCategories(cats.categories ?? []);
        setLoading(false);
      })
      .catch((e) => { setError(e.message); setLoading(false); });
  }, [id]);

  async function saveProduct() {
    if (!product) return;
    setSaving(true); setError("");
    try {
      await updateProduct(id, { name, slug: slug || slugify(name), description, category_id: categoryId, status });

      const existingIds = new Set(skus.map((s) => s.id));
      const skuResults = await Promise.allSettled(
        skuDrafts.map((draft) => {
          const attrs = Object.fromEntries(draft.attributes.filter((a) => a.key).map((a) => [a.key, a.value]));
          if (draft.id && existingIds.has(draft.id)) {
            return updateSku(draft.id, {
              sku_code: draft.sku_code,
              attributes: attrs,
              price_amount: toCents(draft.price_amount, draft.price_currency, currencies),
              compare_price: draft.compare_price ? toCents(draft.compare_price, draft.price_currency, currencies) : undefined,
              weight_grams: draft.weight_grams || undefined,
              is_active: draft.is_active,
            });
          } else if (draft.sku_code.trim()) {
            return createSku({ product_id: id, sku_code: draft.sku_code, attributes: attrs, price_amount: toCents(draft.price_amount, draft.price_currency, currencies), price_currency: draft.price_currency, compare_price: draft.compare_price ? toCents(draft.compare_price, draft.price_currency, currencies) : undefined, weight_grams: draft.weight_grams || undefined, is_active: draft.is_active });
          }
          return Promise.resolve();
        })
      );
      const failures = skuResults.filter((r) => r.status === "rejected");
      if (failures.length > 0) { setError(`${failures.length} SKU(s) failed to save.`); setSaving(false); return; }

      const draftIds = new Set(skuDrafts.filter((d) => d.id).map((d) => d.id));
      await Promise.allSettled(skus.filter((s) => !draftIds.has(s.id)).map((s) => deleteSku(s.id)));

      router.push("/dashboard/products");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
    }
  }

  async function handleImageUpload(file: File) {
    const img = await uploadImage(id, file);
    setImages((prev) => [...prev, img]);
  }

  async function handleImageDelete(imgId: string) {
    await deleteImage(id, imgId);
    setImages((prev) => prev.filter((i) => i.id !== imgId));
  }

  if (loading) return <SkeletonTableCard cols={4} rows={6} />;

  return (
    <>
      <div className="page-header">
        <div style={{ display: "flex", alignItems: "center", gap: "0.75rem" }}>
          <Link href="/dashboard/products" className="btn btn-ghost" style={{ padding: "0.3rem 0.5rem", textDecoration: "none" }}>
            <ArrowLeft size={15} />
          </Link>
          <div>
            <h1 className="page-title">{product?.name ?? "Edit Product"}</h1>
            <div style={{ display: "flex", alignItems: "center", gap: "0.5rem", marginTop: "0.25rem" }}>
              <StatusBadge status={status} />
              <span className="mono" style={{ fontSize: "0.75rem", color: "var(--muted)" }}>{id.slice(0, 12)}…</span>
            </div>
          </div>
        </div>
        <button className="btn btn-primary" onClick={saveProduct} disabled={saving}>
          <Save size={13} /> {saving ? "Saving…" : "Save Changes"}
        </button>
      </div>

      {error && <div style={{ background: "color-mix(in srgb, var(--danger) 10%, transparent)", border: "1px solid color-mix(in srgb, var(--danger) 25%, transparent)", borderRadius: 7, padding: "0.625rem 0.875rem", fontSize: "0.8125rem", color: "var(--danger)", marginBottom: "1rem" }}>{error}</div>}

      <div className="dash-overview-grid" style={{ display: "grid", gridTemplateColumns: "1fr 22rem", gap: "1.5rem", alignItems: "start" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: "1.5rem" }}>
          <div className="card">
            <h3 style={{ margin: "0 0 1rem", fontSize: "0.875rem", fontWeight: 600 }}>Basic Information</h3>
            <div style={{ display: "flex", flexDirection: "column", gap: "0.875rem" }}>
              <div>
                <label htmlFor="edit-name" className="form-label">Product Name *</label>
                <input id="edit-name" style={inputStyle} value={name} onChange={(e) => setName(e.target.value)} />
              </div>
              <div>
                <label htmlFor="edit-slug" className="form-label">Slug</label>
                <input id="edit-slug" style={{ ...inputStyle, fontFamily: '"DM Mono", monospace' }} value={slug} onChange={(e) => setSlug(e.target.value)} />
              </div>
              <div>
                <label htmlFor="edit-desc" className="form-label">Description</label>
                <textarea id="edit-desc" style={{ ...inputStyle, minHeight: "6rem", resize: "vertical" }} value={description} onChange={(e) => setDescription(e.target.value)} />
              </div>
              <div>
                <label htmlFor="edit-category" className="form-label">Category</label>
                <select id="edit-category" style={inputStyle} value={categoryId} onChange={(e) => setCategoryId(e.target.value)}>
                  <option value="">— Select category —</option>
                  {categories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
                </select>
              </div>
            </div>
          </div>

          <div>
            <h3 style={{ margin: "0 0 0.75rem", fontSize: "0.875rem", fontWeight: 600, color: "var(--text)" }}>SKUs / Variants</h3>
            <SKUEditor initial={skuDrafts} onChange={setSkuDrafts} />
          </div>
        </div>

        <div style={{ display: "flex", flexDirection: "column", gap: "1.25rem" }}>
          <div className="card">
            <h3 style={{ margin: "0 0 0.875rem", fontSize: "0.875rem", fontWeight: 600 }}>Status</h3>
            <label htmlFor="edit-status" className="form-label" style={{ position: "absolute", width: 1, height: 1, overflow: "hidden", clip: "rect(0,0,0,0)" }}>Status</label>
            <select id="edit-status" style={inputStyle} value={status} onChange={(e) => setStatus(e.target.value)}>
              <option value="DRAFT">Draft</option>
              <option value="ACTIVE">Active</option>
              <option value="ARCHIVED">Archived</option>
            </select>
          </div>

          <div className="card">
            <h3 style={{ margin: "0 0 0.875rem", fontSize: "0.875rem", fontWeight: 600 }}>Images</h3>
            {images.length > 0 && (
              <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: "0.5rem", marginBottom: "0.875rem" }}>
                {images.map((img) => (
                  <div key={img.id} style={{ position: "relative", borderRadius: 7, overflow: "hidden", border: "1px solid var(--border)", aspectRatio: "1" }}>
                    {/* eslint-disable-next-line @next/next/no-img-element */}
                    <img src={img.url} alt="" style={{ width: "100%", height: "100%", objectFit: "cover", display: "block" }} />
                    <button onClick={() => handleImageDelete(img.id)} style={{ position: "absolute", top: 3, right: 3, background: "rgba(0,0,0,0.6)", border: "none", borderRadius: 4, cursor: "pointer", padding: "2px", display: "flex" }}>
                      <Trash2 size={11} color="#fff" />
                    </button>
                  </div>
                ))}
              </div>
            )}
            <ImageDropzone onUpload={handleImageUpload} />
          </div>
        </div>
      </div>
    </>
  );
}