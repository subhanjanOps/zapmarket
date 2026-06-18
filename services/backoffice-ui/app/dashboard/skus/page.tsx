"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { Search, X } from "lucide-react";
import { getToken } from "@/lib/auth";
import { getSkus, updateSku, deleteSku, type SKU } from "@/lib/api";
import { TableSkeleton } from "@/app/components/Skeleton";
import { showAlert, showConfirm } from "@/app/components/Dialog";

const PAGE_SIZE = 20;

export default function SkusPage() {
  const searchParams = useSearchParams();

  const [rows, setRows]         = useState<SKU[]>([]);
  const [total, setTotal]       = useState(0);
  const [page, setPage]         = useState(0);
  const [loading, setLoading]   = useState(true);
  const [productId, setProductId] = useState(searchParams.get("product_id") ?? "");
  const [skuCode, setSkuCode]   = useState(searchParams.get("sku_code") ?? "");
  const [deleting, setDeleting] = useState<string | null>(null);
  const [toggling, setToggling] = useState<string | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    const token = getToken() ?? undefined;
    getSkus({
      product_id: productId || undefined,
      sku_code: skuCode || undefined,
      limit: PAGE_SIZE,
      offset: page * PAGE_SIZE,
    }, token)
      .then((r) => { setRows(r.data); setTotal(r.total); })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, [productId, skuCode, page]);

  useEffect(() => { load(); }, [load]);

  async function toggleActive(s: SKU) {
    const token = getToken();
    if (!token) return;
    setToggling(s.id);
    try {
      await updateSku(token, s.id, { is_active: !s.is_active });
      load();
    } catch (e: unknown) {
      await showAlert(e instanceof Error ? e.message : "Update failed");
    } finally {
      setToggling(null);
    }
  }

  async function handleDelete(id: string) {
    if (!await showConfirm("Delete this SKU?")) return;
    const token = getToken();
    if (!token) return;
    setDeleting(id);
    try { await deleteSku(token, id); load(); }
    catch (e: unknown) { await showAlert(e instanceof Error ? e.message : "Delete failed"); }
    finally { setDeleting(null); }
  }

  const pages = Math.ceil(total / PAGE_SIZE);

  return (
    <div className="page-content">
      <div className="page-header">
        <div>
          <h1 className="page-title">SKUs</h1>
          <p className="page-subtitle">{total} SKU{total === 1 ? "" : "s"}</p>
        </div>
      </div>

      {/* Filters */}
      <div style={{ display: "flex", gap: "0.625rem", marginBottom: "1.25rem", flexWrap: "wrap", alignItems: "center" }}>
        {/* SKU code search */}
        <div style={{ position: "relative", flex: "1 1 180px", minWidth: 160 }}>
          <Search size={14} style={{ position: "absolute", left: 10, top: "50%", transform: "translateY(-50%)", color: "var(--muted)", pointerEvents: "none" }} />
          <input
            className="input"
            style={{ paddingLeft: 30, paddingRight: skuCode ? 30 : undefined }}
            placeholder="Search by SKU code…"
            value={skuCode}
            onChange={(e) => {
              const v = e.target.value;
              setSkuCode(v);
              if (debounceRef.current) clearTimeout(debounceRef.current);
              debounceRef.current = setTimeout(() => setPage(0), 300);
            }}
          />
          {skuCode && (
            <button onClick={() => { setSkuCode(""); setPage(0); }} style={{ position: "absolute", right: 8, top: "50%", transform: "translateY(-50%)", background: "none", border: "none", cursor: "pointer", color: "var(--muted)", display: "flex", alignItems: "center", padding: 2 }}>
              <X size={13} />
            </button>
          )}
        </div>
        {/* Product ID filter */}
        <div style={{ position: "relative", flex: "1 1 220px", minWidth: 180 }}>
          <Search size={14} style={{ position: "absolute", left: 10, top: "50%", transform: "translateY(-50%)", color: "var(--muted)", pointerEvents: "none" }} />
          <input
            className="input"
            style={{ paddingLeft: 30, paddingRight: productId ? 30 : undefined, fontFamily: "monospace", fontSize: "0.8rem" }}
            placeholder="Filter by Product ID (UUID)…"
            value={productId}
            onChange={(e) => {
              const v = e.target.value;
              setProductId(v);
              if (debounceRef.current) clearTimeout(debounceRef.current);
              debounceRef.current = setTimeout(() => setPage(0), 300);
            }}
          />
          {productId && (
            <button onClick={() => { setProductId(""); setPage(0); }} style={{ position: "absolute", right: 8, top: "50%", transform: "translateY(-50%)", background: "none", border: "none", cursor: "pointer", color: "var(--muted)", display: "flex", alignItems: "center", padding: 2 }}>
              <X size={13} />
            </button>
          )}
        </div>
        {(skuCode || productId) && (
          <button className="btn btn-ghost" style={{ fontSize: "0.8125rem", padding: "0.375rem 0.75rem", whiteSpace: "nowrap" }}
            onClick={() => { setSkuCode(""); setProductId(""); setPage(0); }}>
            Clear
          </button>
        )}
      </div>

      <div style={{ border: "1px solid var(--border)", borderRadius: 7, overflow: "hidden" }}>
        <table>
          <thead>
            <tr>
              <th>SKU Code</th>
              <th>Product ID</th>
              <th>Price</th>
              <th>Currency</th>
              <th>Active</th>
              <th>Attributes</th>
              <th style={{ width: 140 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <TableSkeleton rows={8} cols={7} />
            ) : rows.length === 0 ? (
              <tr>
                <td colSpan={7}>
                  <div className="empty-state">
                    <p className="empty-state-title">No SKUs found</p>
                    <p className="empty-state-body">Try adjusting your filters.</p>
                  </div>
                </td>
              </tr>
            ) : (
              rows.map((s) => (
                <tr key={s.id}>
                  <td><span className="mono" style={{ fontWeight: 500 }}>{s.sku_code}</span></td>
                  <td>
                    <Link
                      href={`/dashboard/products/${s.product_id}`}
                      style={{ color: "var(--accent)", textDecoration: "none", fontFamily: "monospace", fontSize: "0.75rem" }}
                    >
                      {s.product_id.slice(0, 12)}…
                    </Link>
                  </td>
                  <td style={{ color: "var(--text)" }}>{s.price_amount}</td>
                  <td><span className="mono" style={{ color: "var(--text-2)" }}>{s.currency}</span></td>
                  <td>
                    <span className={`badge ${s.is_active ? "badge-green" : "badge-gray"}`}>
                      {s.is_active ? "Yes" : "No"}
                    </span>
                  </td>
                  <td style={{ maxWidth: 180, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap", color: "var(--muted)", fontSize: "0.75rem" }}>
                    {s.variant_attributes ? JSON.stringify(s.variant_attributes) : "—"}
                  </td>
                  <td>
                    <div style={{ display: "flex", gap: "0.375rem" }}>
                      <button
                        className={`btn ${s.is_active ? "btn-danger" : "btn-success"}`}
                        style={{ padding: "0.25rem 0.625rem", fontSize: "0.75rem" }}
                        onClick={() => toggleActive(s)}
                        disabled={toggling === s.id}
                      >
                        {toggling === s.id ? "…" : s.is_active ? "Deactivate" : "Activate"}
                      </button>
                      <button
                        className="btn btn-danger"
                        style={{ padding: "0.25rem 0.625rem", fontSize: "0.75rem" }}
                        onClick={() => handleDelete(s.id)}
                        disabled={deleting === s.id}
                      >
                        {deleting === s.id ? "…" : "Delete"}
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {pages > 1 && (
        <div style={{ display: "flex", alignItems: "center", gap: "0.5rem", marginTop: "1rem", justifyContent: "flex-end" }}>
          <button className="btn btn-ghost" disabled={page === 0} onClick={() => setPage((p) => p - 1)}>← Prev</button>
          <span style={{ fontSize: "0.8125rem", color: "var(--text-2)" }}>Page {page + 1} / {pages}</span>
          <button className="btn btn-ghost" disabled={page >= pages - 1} onClick={() => setPage((p) => p + 1)}>Next →</button>
        </div>
      )}
    </div>
  );
}
