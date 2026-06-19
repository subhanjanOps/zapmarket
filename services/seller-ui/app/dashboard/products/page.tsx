"use client";
import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { RefreshCw, Plus, Pencil, Trash2 } from "lucide-react";
import { getProducts, deleteProduct, Product } from "@/lib/api";
import { StatusBadge } from "@/app/components/StatusBadge";
import { SkeletonTableCard } from "@/app/components/Skeleton";
import { showAlert, showConfirm } from "@/app/components/Dialog";

const STATUSES = ["All", "ACTIVE", "DRAFT", "ARCHIVED"] as const;
const PAGE_SIZE = 20;

export default function ProductsPage() {
  const [products, setProducts] = useState<Product[]>([]);
  const [total, setTotal]       = useState(0);
  const [loading, setLoading]   = useState(true);
  const [error, setError]       = useState("");
  const [search, setSearch]     = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [status, setStatus]     = useState<typeof STATUSES[number]>("All");
  const [offset, setOffset]     = useState(0);
  const [deleting, setDeleting] = useState<string | null>(null);

  // Debounce search input by 300ms to avoid firing on every keystroke
  useEffect(() => {
    const t = setTimeout(() => setDebouncedSearch(search), 300);
    return () => clearTimeout(t);
  }, [search]);

  const load = useCallback(() => {
    setLoading(true);
    getProducts({
      status: status === "All" ? undefined : status,
      search: debouncedSearch || undefined,
      limit: PAGE_SIZE,
      offset,
    })
      .then((r) => { setProducts(r.products ?? []); setTotal(r.total ?? 0); setLoading(false); })
      .catch((e) => { setError(e.message); setLoading(false); });
  }, [status, debouncedSearch, offset]);

  useEffect(() => { load(); }, [load]);

  async function handleDelete(id: string, name: string) {
    if (!await showConfirm(`Delete "${name}"? This cannot be undone.`)) return;
    setDeleting(id);
    try {
      await deleteProduct(id);
      load();
    } catch (e) {
      await showAlert(e instanceof Error ? e.message : "Delete failed");
    } finally {
      setDeleting(null);
    }
  }

  const pages = Math.ceil(total / PAGE_SIZE);
  const page  = Math.floor(offset / PAGE_SIZE) + 1;

  return (
    <>
      <div className="page-header">
        <div>
          <h1 className="page-title">Products</h1>
          <p className="page-subtitle">{total} total</p>
        </div>
        <div style={{ display: "flex", gap: "0.5rem" }}>
          <button className="btn btn-ghost" onClick={load}><RefreshCw size={13} /></button>
          <Link href="/dashboard/products/new" className="btn btn-primary" style={{ textDecoration: "none" }}>
            <Plus size={13} /> New Product
          </Link>
        </div>
      </div>

      {/* Filters */}
      <div style={{ display: "flex", gap: "0.75rem", marginBottom: "1.25rem", alignItems: "center", flexWrap: "wrap" }}>
        <div style={{ display: "flex", gap: "2px", background: "var(--surface2)", borderRadius: 8, padding: 2 }}>
          {STATUSES.map((s) => (
            <button
              key={s}
              onClick={() => { setStatus(s); setOffset(0); }}
              style={{ padding: "0.3125rem 0.75rem", borderRadius: 6, border: "none", cursor: "pointer", fontSize: "0.8125rem", fontFamily: "inherit", background: status === s ? "var(--surface)" : "transparent", color: status === s ? "var(--text)" : "var(--muted)", fontWeight: status === s ? 500 : 400, boxShadow: status === s ? "var(--shadow-sm)" : "none", transition: "all 0.1s" }}
            >
              {s === "All" ? "All" : s.charAt(0) + s.slice(1).toLowerCase()}
            </button>
          ))}
        </div>
        <label htmlFor="product-search" style={{ position: "absolute", width: 1, height: 1, overflow: "hidden", clip: "rect(0,0,0,0)" }}>Search products</label>
        <input
          id="product-search"
          className="input"
          style={{ width: "14rem" }}
          placeholder="Search products…"
          value={search}
          onChange={(e) => { setSearch(e.target.value); setOffset(0); }}
        />
      </div>

      {error && <p style={{ color: "var(--danger)", fontSize: "0.8125rem", marginBottom: "1rem" }}>{error}</p>}

      {loading ? (
        <SkeletonTableCard cols={6} rows={8} />
      ) : (
        <div className="card" style={{ padding: 0, overflow: "hidden" }}>
          <div style={{ overflowX: "auto", WebkitOverflowScrolling: "touch" }}>
            <table>
              <thead>
                <tr><th>Name</th><th>Category</th><th>Status</th><th>Created</th><th style={{ textAlign: "right" }}>Actions</th></tr>
              </thead>
              <tbody>
                {products.length === 0 ? (
                  <tr><td colSpan={5} className="empty-state"><p className="empty-state-title">No products found</p></td></tr>
                ) : products.map((p) => (
                  <tr key={p.id}>
                    <td>
                      <Link href={`/dashboard/products/${p.id}`} style={{ color: "var(--text)", textDecoration: "none", fontWeight: 500 }}>{p.name}</Link>
                      <div className="mono" style={{ fontSize: "0.75rem", color: "var(--muted)" }}>{p.slug}</div>
                    </td>
                    <td style={{ color: "var(--text-2)" }}>{p.category_name ?? "—"}</td>
                    <td><StatusBadge status={p.status} /></td>
                    <td style={{ color: "var(--muted)" }}>{new Date(p.created_at).toLocaleDateString()}</td>
                    <td style={{ textAlign: "right" }}>
                      <div style={{ display: "flex", gap: "0.375rem", justifyContent: "flex-end" }}>
                        <Link href={`/dashboard/products/${p.id}`} className="btn btn-ghost" style={{ padding: "0.3rem 0.5rem", textDecoration: "none" }}>
                          <Pencil size={13} />
                        </Link>
                        <button className="btn btn-danger" style={{ padding: "0.3rem 0.5rem" }} disabled={deleting === p.id} onClick={() => handleDelete(p.id, p.name)}>
                          <Trash2 size={13} />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {pages > 1 && (
            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", padding: "0.75rem 1.125rem", borderTop: "1px solid var(--border)" }}>
              <span style={{ fontSize: "0.8125rem", color: "var(--muted)" }}>Page {page} of {pages}</span>
              <div style={{ display: "flex", gap: "0.375rem" }}>
                <button className="btn btn-ghost" disabled={offset === 0} onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}>← Prev</button>
                <button className="btn btn-ghost" disabled={offset + PAGE_SIZE >= total} onClick={() => setOffset(offset + PAGE_SIZE)}>Next →</button>
              </div>
            </div>
          )}
        </div>
      )}
    </>
  );
}
