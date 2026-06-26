"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { getProducts, updateProduct, type Product } from "@/lib/api";
import StatusBadge from "@/app/components/StatusBadge";
import { TableSkeleton } from "@/app/components/Skeleton";
import { showAlert } from "@/app/components/Dialog";

const PAGE_SIZE = 20;

export default function ModerationPage() {
  const [rows, setRows]         = useState<Product[]>([]);
  const [total, setTotal]       = useState(0);
  const [page, setPage]         = useState(0);
  const [loading, setLoading]   = useState(true);
  const [updating, setUpdating] = useState<string | null>(null);
  const [error, setError]       = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    getProducts({ status: "DRAFT", limit: PAGE_SIZE, offset: page * PAGE_SIZE })
      .then((r) => { setRows(r.data); setTotal(r.total); setError(null); })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : "Failed to load products"))
      .finally(() => setLoading(false));
  }, [page]);

  useEffect(() => { load(); }, [load]);

  async function approve(p: Product) {
    setUpdating(p.id);
    try {
      await updateProduct(p.id, { status: "ACTIVE" });
      load();
    } catch (e: unknown) {
      await showAlert(e instanceof Error ? e.message : "Failed");
    } finally {
      setUpdating(null);
    }
  }

  async function reject(p: Product) {
    setUpdating(p.id);
    try {
      await updateProduct(p.id, { status: "ARCHIVED" });
      load();
    } catch (e: unknown) {
      await showAlert(e instanceof Error ? e.message : "Failed");
    } finally {
      setUpdating(null);
    }
  }

  const pages = Math.ceil(total / PAGE_SIZE);

  return (
    <div className="page-content">
      {error && (
        <div style={{ marginBottom: "1rem", padding: "0.75rem 1rem", background: "#FEF2F2", border: "1px solid #FCA5A5", borderRadius: "0.5rem", color: "#DC2626", fontSize: "0.875rem" }}>
          {error}
        </div>
      )}
      <div className="page-header">
        <div>
          <h1 className="page-title">Moderation queue</h1>
          <p className="page-subtitle">{total} product{total === 1 ? "" : "s"} pending review (DRAFT)</p>
        </div>
      </div>

      <div style={{ border: "1px solid var(--border)", borderRadius: 7, overflow: "hidden" }}>
        <table>
          <thead>
            <tr>
              <th>Product</th>
              <th>Seller ID</th>
              <th>Status</th>
              <th>Created</th>
              <th style={{ width: 160 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <TableSkeleton rows={8} cols={5} />
            ) : rows.length === 0 ? (
              <tr>
                <td colSpan={5}>
                  <div className="empty-state">
                    <p className="empty-state-title">Queue is clear</p>
                    <p className="empty-state-body">No DRAFT products awaiting review.</p>
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
                    {p.description && (
                      <div style={{ fontSize: "0.75rem", color: "var(--muted)", marginTop: 2, maxWidth: 320, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                        {p.description}
                      </div>
                    )}
                  </td>
                  <td><span className="mono" style={{ fontSize: "0.75rem", color: "var(--muted)" }}>{p.seller_id.slice(0, 12)}…</span></td>
                  <td><StatusBadge status={p.status} /></td>
                  <td style={{ color: "var(--text-2)" }}>{new Date(p.created_at).toLocaleDateString()}</td>
                  <td>
                    <div style={{ display: "flex", gap: "0.375rem" }}>
                      <button
                        className="btn btn-success"
                        style={{ padding: "0.25rem 0.75rem", fontSize: "0.75rem" }}
                        onClick={() => approve(p)}
                        disabled={updating === p.id}
                      >
                        {updating === p.id ? "…" : "Approve"}
                      </button>
                      <button
                        className="btn btn-danger"
                        style={{ padding: "0.25rem 0.75rem", fontSize: "0.75rem" }}
                        onClick={() => reject(p)}
                        disabled={updating === p.id}
                      >
                        {updating === p.id ? "…" : "Reject"}
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
