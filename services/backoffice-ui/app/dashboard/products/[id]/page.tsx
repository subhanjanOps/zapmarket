"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { getToken } from "@/lib/auth";
import {
  getProduct, getSkus, getImages, updateProduct, deleteImage,
  type Product, type SKU, type ProductImage,
} from "@/lib/api";
import StatusBadge from "@/app/components/StatusBadge";
import { TableSkeleton } from "@/app/components/Skeleton";
import { showAlert, showConfirm } from "@/app/components/Dialog";

export default function ProductDetailPage() {
  const { id }                     = useParams<{ id: string }>();
  const [product, setProduct]      = useState<Product | null>(null);
  const [skus, setSkus]            = useState<SKU[]>([]);
  const [images, setImages]        = useState<ProductImage[]>([]);
  const [loading, setLoading]      = useState(true);
  const [saving, setSaving]        = useState(false);
  const [statusSel, setStatusSel]  = useState("");
  const [statusErr, setStatusErr]  = useState("");

  useEffect(() => {
    const token = getToken() ?? undefined;
    Promise.all([
      getProduct(id),
      getSkus({ product_id: id }, token),
      getImages(id),
    ]).then(([p, s, img]) => {
      setProduct(p);
      setStatusSel(p.status);
      setSkus(s.data);
      setImages(img.data);
    }).catch(console.error)
      .finally(() => setLoading(false));
  }, [id]);

  async function saveStatus() {
    const token = getToken();
    if (!token || !product) return;
    setSaving(true);
    setStatusErr("");
    try {
      const updated = await updateProduct(token, id, { status: statusSel });
      setProduct(updated);
    } catch (e: unknown) {
      setStatusErr(e instanceof Error ? e.message : "Update failed");
    } finally {
      setSaving(false);
    }
  }

  async function handleDeleteImage(imageId: string) {
    if (!await showConfirm("Remove this image?")) return;
    const token = getToken();
    if (!token) return;
    try {
      await deleteImage(token, id, imageId);
      setImages((imgs) => imgs.filter((i) => i.id !== imageId));
    } catch (e: unknown) {
      await showAlert(e instanceof Error ? e.message : "Delete failed");
    }
  }

  if (loading) return null;
  if (!product) return (
    <div style={{ padding: "2rem" }}>
      <p style={{ color: "var(--danger)" }}>Product not found.</p>
      <Link href="/dashboard/products" className="btn btn-ghost" style={{ marginTop: "1rem" }}>← Back</Link>
    </div>
  );

  const dirty = statusSel !== product.status;

  return (
    <div style={{ padding: "2rem" }}>
      {/* Breadcrumb */}
      <div style={{ marginBottom: "1.25rem", display: "flex", alignItems: "center", gap: "0.375rem", fontSize: "0.8rem", color: "var(--muted)" }}>
        <Link href="/dashboard/products" style={{ color: "var(--muted)", textDecoration: "none" }}>Products</Link>
        <span>›</span>
        <span style={{ color: "var(--text-2)" }}>{product.name}</span>
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "1fr 300px", gap: "1.5rem", alignItems: "start" }}>
        {/* Left */}
        <div style={{ display: "flex", flexDirection: "column", gap: "1.25rem" }}>
          <div className="card">
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: "1rem" }}>
              <div>
                <h1 style={{ margin: "0 0 0.25rem", fontSize: "1.125rem", fontWeight: 600, color: "var(--text)" }}>{product.name}</h1>
                <span className="mono" style={{ fontSize: "0.75rem", color: "var(--muted)" }}>{product.slug}</span>
              </div>
              <StatusBadge status={product.status} />
            </div>
            {product.description && (
              <p style={{ margin: 0, color: "var(--text-2)", fontSize: "0.875rem", lineHeight: 1.6 }}>{product.description}</p>
            )}
          </div>

          {/* Images */}
          <div className="card">
            <div className="card-title" style={{ marginBottom: "1rem" }}>Images ({images.length})</div>
            {images.length === 0 ? (
              <p style={{ color: "var(--muted)", fontSize: "0.8rem", margin: 0 }}>No images uploaded.</p>
            ) : (
              <div style={{ display: "flex", flexWrap: "wrap", gap: "0.75rem" }}>
                {images.map((img) => (
                  <div key={img.id} style={{ position: "relative" }}>
                    <img
                      src={img.url}
                      alt=""
                      style={{ width: 80, height: 80, objectFit: "cover", borderRadius: 6, border: "1px solid var(--border)", display: "block" }}
                    />
                    <div
                      style={{
                        position: "absolute",
                        top: 2,
                        right: 2,
                        background: "rgba(0,0,0,0.65)",
                        borderRadius: 4,
                        fontSize: 10,
                        color: "#fff",
                        padding: "1px 4px",
                      }}
                    >
                      #{img.position}
                    </div>
                    <button
                      onClick={() => handleDeleteImage(img.id)}
                      style={{
                        position: "absolute",
                        bottom: 2,
                        right: 2,
                        background: "var(--danger)",
                        border: "none",
                        borderRadius: 4,
                        color: "#fff",
                        fontSize: 10,
                        padding: "1px 5px",
                        cursor: "pointer",
                      }}
                    >
                      ✕
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* SKUs */}
          <div className="card" style={{ padding: 0, overflow: "hidden" }}>
            <div className="card-header">
              <span className="card-title">SKUs ({skus.length})</span>
              <Link href={`/dashboard/skus?product_id=${product.id}`} className="btn btn-ghost" style={{ fontSize: "0.75rem", padding: "0.25rem 0.625rem" }}>
                View all
              </Link>
            </div>
            <table>
              <thead>
                <tr>
                  <th>SKU Code</th>
                  <th>Price</th>
                  <th>Currency</th>
                  <th>Active</th>
                  <th>Attributes</th>
                </tr>
              </thead>
              <tbody>
                {skus.length === 0 ? (
                  <tr>
                    <td colSpan={5}>
                      <div className="empty-state">
                        <p className="empty-state-title">No SKUs</p>
                        <p className="empty-state-body">Add SKUs from the SKUs page.</p>
                      </div>
                    </td>
                  </tr>
                ) : (
                  skus.map((s) => (
                    <tr key={s.id}>
                      <td><span className="mono">{s.sku_code}</span></td>
                      <td>{s.price_amount}</td>
                      <td><span className="mono" style={{ color: "var(--text-2)" }}>{s.currency}</span></td>
                      <td>
                        <span className={`badge ${s.is_active ? "badge-green" : "badge-gray"}`}>
                          {s.is_active ? "Yes" : "No"}
                        </span>
                      </td>
                      <td style={{ color: "var(--muted)", fontSize: "0.75rem" }}>
                        {s.variant_attributes ? JSON.stringify(s.variant_attributes) : "—"}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* Right sidebar */}
        <div style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
          <div className="card">
            <div className="card-title" style={{ marginBottom: "0.875rem" }}>Status</div>
            <select className="input" value={statusSel} onChange={(e) => setStatusSel(e.target.value)}>
              <option value="DRAFT">DRAFT</option>
              <option value="ACTIVE">ACTIVE</option>
              <option value="ARCHIVED">ARCHIVED</option>
            </select>
            {statusErr && <p style={{ color: "var(--danger)", fontSize: "0.75rem", margin: "0.375rem 0 0" }}>{statusErr}</p>}
            <button
              className="btn btn-primary"
              style={{ width: "100%", marginTop: "0.75rem", justifyContent: "center" }}
              onClick={saveStatus}
              disabled={!dirty || saving}
            >
              {saving ? "Saving…" : "Update status"}
            </button>
          </div>

          <div className="card">
            <div className="card-title" style={{ marginBottom: "0.75rem" }}>Details</div>
            <dl style={{ margin: 0, display: "flex", flexDirection: "column", gap: "0.5rem" }}>
              {[
                ["ID", <span key="id" className="mono" style={{ fontSize: "0.75rem" }}>{product.id.slice(0, 16)}…</span>],
                ["Seller", <span key="s" className="mono" style={{ fontSize: "0.75rem" }}>{product.seller_id.slice(0, 16)}…</span>],
                ["Created", new Date(product.created_at).toLocaleDateString()],
                ["Updated", new Date(product.updated_at).toLocaleDateString()],
              ].map(([k, v]) => (
                <div key={String(k)} style={{ display: "flex", justifyContent: "space-between", gap: "0.5rem" }}>
                  <dt style={{ fontSize: "0.75rem", color: "var(--muted)", flexShrink: 0 }}>{k}</dt>
                  <dd style={{ margin: 0, fontSize: "0.8125rem", color: "var(--text-2)", textAlign: "right" }}>{v}</dd>
                </div>
              ))}
            </dl>
          </div>
        </div>
      </div>
    </div>
  );
}
