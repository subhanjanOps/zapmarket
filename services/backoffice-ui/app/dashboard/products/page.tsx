"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { getToken } from "@/lib/auth";
import { getProducts, getCategories, createProduct, updateProduct, deleteProduct, type Product, type Category } from "@/lib/api";
import StatusBadge from "@/app/components/StatusBadge";
import { TableSkeleton } from "@/app/components/Skeleton";
import { ExportButton, ImportButton } from "@/app/components/BulkIO";
import { showAlert, showConfirm } from "@/app/components/Dialog";

const PROD_EXPORT_HEADERS = ["id", "name", "slug", "category_id", "seller_id", "status", "created_at"];
const PROD_IMPORT_HEADERS = ["name", "slug", "category_id", "seller_id", "status"];
const PROD_TEMPLATE = { name: "iPhone 16", slug: "iphone-16", category_id: "<uuid>", seller_id: "<uuid>", status: "DRAFT" };

const STATUSES = ["", "DRAFT", "ACTIVE", "ARCHIVED"];
const PAGE_SIZE = 20;

export default function ProductsPage() {
  const router       = useRouter();
  const searchParams = useSearchParams();

  const [rows, setRows]           = useState<Product[]>([]);
  const [total, setTotal]         = useState(0);
  const [page, setPage]           = useState(0);
  const [loading, setLoading]     = useState(true);
  const [cats, setCats]           = useState<Category[]>([]);
  const [search, setSearch]       = useState(searchParams.get("search") ?? "");
  const [status, setStatus]       = useState(searchParams.get("status") ?? "");
  const [categoryId, setCategoryId] = useState(searchParams.get("category_id") ?? "");
  const [deleting, setDeleting]   = useState<string | null>(null);

  useEffect(() => {
    getCategories().then((r) => setCats(r.data)).catch(console.error);
  }, []);

  const load = useCallback(() => {
    setLoading(true);
    const token = getToken() ?? undefined;
    getProducts({
      search: search || undefined,
      status: status || undefined,
      category_id: categoryId || undefined,
      limit: PAGE_SIZE,
      offset: page * PAGE_SIZE,
    }, token)
      .then((r) => { setRows(r.data); setTotal(r.total); })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, [search, status, categoryId, page]);

  useEffect(() => { load(); }, [load]);

  function catName(id: string) {
    return cats.find((c) => c.id === id)?.name ?? <span className="mono" style={{ fontSize: "0.75rem", color: "var(--muted)" }}>{id.slice(0, 8)}</span>;
  }

  async function toggleStatus(p: Product) {
    const token = getToken();
    if (!token) return;
    const next = p.status === "ACTIVE" ? "ARCHIVED" : "ACTIVE";
    try {
      await updateProduct(token, p.id, { status: next });
      load();
    } catch (e: unknown) {
      await showAlert(e instanceof Error ? e.message : "Update failed");
    }
  }

  async function handleDelete(id: string) {
    if (!await showConfirm("Delete this product?")) return;
    const token = getToken();
    if (!token) return;
    setDeleting(id);
    try { await deleteProduct(token, id); load(); }
    catch (e: unknown) { await showAlert(e instanceof Error ? e.message : "Delete failed"); }
    finally { setDeleting(null); }
  }

  const pages = Math.ceil(total / PAGE_SIZE);

  return (
    <div className="page-content">
      <div className="page-header">
        <div>
          <h1 className="page-title">Products</h1>
          <p className="page-subtitle">{total} product{total === 1 ? "" : "s"}</p>
        </div>
        <div style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
          <ExportButton
            filename="products.csv"
            headers={PROD_EXPORT_HEADERS}
            fetchAll={async () => {
              const token = getToken() ?? undefined;
              const PAGE = 100;
              const first = await getProducts({ limit: PAGE, offset: 0 }, token);
              const all: Product[] = [...first.data];
              const totalCount = first.total;
              let offset = PAGE;
              while (offset < totalCount) {
                const r = await getProducts({ limit: PAGE, offset }, token);
                all.push(...r.data);
                offset += PAGE;
              }
              return all as unknown as Record<string, unknown>[];
            }}
          />
          <ImportButton
            title="Products"
            expectedHeaders={PROD_IMPORT_HEADERS}
            templateRow={PROD_TEMPLATE}
            importRow={async (row) => {
              const token = getToken();
              if (!token) throw new Error("Not authenticated");
              await createProduct(token, {
                name: row.name,
                slug: row.slug,
                category_id: row.category_id,
                seller_id: row.seller_id,
                status: row.status || "DRAFT",
              });
            }}
            onDone={load}
          />
        </div>
      </div>

      {/* Filters */}
      <div style={{ display: "flex", gap: "0.625rem", marginBottom: "1.25rem", flexWrap: "wrap" }}>
        <input
          className="input"
          style={{ maxWidth: 260 }}
          placeholder="Search products…"
          value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(0); }}
        />
        <select className="input" style={{ maxWidth: 160 }} value={status} onChange={(e) => { setStatus(e.target.value); setPage(0); }}>
          {STATUSES.map((s) => <option key={s} value={s}>{s || "All statuses"}</option>)}
        </select>
        <select className="input" style={{ maxWidth: 180 }} value={categoryId} onChange={(e) => { setCategoryId(e.target.value); setPage(0); }}>
          <option value="">All categories</option>
          {cats.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
        </select>
      </div>

      <div style={{ border: "1px solid var(--border)", borderRadius: 7, overflow: "hidden" }}>
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Category</th>
              <th>Status</th>
              <th>Seller ID</th>
              <th>Created</th>
              <th style={{ width: 140 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <TableSkeleton rows={8} cols={6} />
            ) : rows.length === 0 ? (
              <tr>
                <td colSpan={6}>
                  <div className="empty-state">
                    <p className="empty-state-title">No products found</p>
                    <p className="empty-state-body">Try adjusting your filters.</p>
                  </div>
                </td>
              </tr>
            ) : (
              rows.map((p) => (
                <tr key={p.id}>
                  <td>
                    <Link href={`/dashboard/products/${p.id}`} style={{ color: "var(--accent)", textDecoration: "none", fontWeight: 500 }}>
                      {p.name}
                    </Link>
                  </td>
                  <td style={{ color: "var(--text-2)" }}>{catName(p.category_id)}</td>
                  <td><StatusBadge status={p.status} /></td>
                  <td><span className="mono" style={{ fontSize: "0.75rem", color: "var(--muted)" }}>{p.seller_id.slice(0, 8)}</span></td>
                  <td style={{ color: "var(--text-2)" }}>{new Date(p.created_at).toLocaleDateString()}</td>
                  <td>
                    <div style={{ display: "flex", gap: "0.375rem" }}>
                      <button
                        className={`btn ${p.status === "ACTIVE" ? "btn-danger" : "btn-success"}`}
                        style={{ padding: "0.25rem 0.625rem", fontSize: "0.75rem" }}
                        onClick={() => toggleStatus(p)}
                      >
                        {p.status === "ACTIVE" ? "Archive" : "Activate"}
                      </button>
                      <button
                        className="btn btn-danger"
                        style={{ padding: "0.25rem 0.625rem", fontSize: "0.75rem" }}
                        onClick={() => handleDelete(p.id)}
                        disabled={deleting === p.id}
                      >
                        {deleting === p.id ? "…" : "Delete"}
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
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
