"use client";

import { useEffect, useState, useCallback } from "react";
import { getToken } from "@/lib/auth";
import {
  getCategories, createCategory, updateCategory, deleteCategory, bulkCreateCategories,
  type Category,
} from "@/lib/api";
import { TableSkeleton } from "@/app/components/Skeleton";
import { ExportButton, ImportButton } from "@/app/components/BulkIO";
import { showAlert, showConfirm } from "@/app/components/Dialog";

const PAGE_SIZE = 20;
const CAT_EXPORT_HEADERS = ["id", "name", "slug", "parent_id", "created_at"];
const CAT_IMPORT_HEADERS = ["name", "slug", "parent_name"];
const CAT_TEMPLATE = { name: "Electronics", slug: "electronics", parent_name: "" };

type FormState = { name: string; slug: string; parent_id: string };
const EMPTY: FormState = { name: "", slug: "", parent_id: "" };

function slugify(s: string) {
  return s.toLowerCase().replace(/\s+/g, "-").replace(/[^a-z0-9-]/g, "");
}

// ── Pagination bar ────────────────────────────────────────────────────────────

function Pagination({ page, pages, onChange }: { page: number; pages: number; onChange: (p: number) => void }) {
  if (pages <= 1) return null;

  // Build the page number window: always show first, last, current ±1, with ellipsis gaps
  const range: (number | "…")[] = [];
  const add = (n: number) => { if (!range.includes(n)) range.push(n); };
  add(1);
  if (page > 3) range.push("…");
  for (let i = Math.max(2, page - 1); i <= Math.min(pages - 1, page + 1); i++) add(i);
  if (page < pages - 2) range.push("…");
  add(pages);

  const btn: React.CSSProperties = {
    minWidth: 34, height: 34, padding: "0 0.5rem",
    display: "inline-flex", alignItems: "center", justifyContent: "center",
    borderRadius: 6, fontSize: "0.8125rem", cursor: "pointer",
    border: "1px solid var(--border)", background: "var(--surface2)",
    color: "var(--text)", lineHeight: 1,
  };
  const active: React.CSSProperties = {
    ...btn, background: "var(--accent)", color: "#fff", borderColor: "var(--accent)", fontWeight: 600,
  };
  const disabled: React.CSSProperties = {
    ...btn, opacity: 0.4, cursor: "default",
  };

  return (
    <div style={{ display: "flex", alignItems: "center", gap: "0.25rem", justifyContent: "flex-end", marginTop: "1rem" }}>
      <button
        style={page === 1 ? disabled : btn}
        disabled={page === 1}
        onClick={() => onChange(page - 1)}
      >
        ←
      </button>
      {range.map((r, i) =>
        r === "…" ? (
          <span key={`ellipsis-${i}`} style={{ padding: "0 0.25rem", color: "var(--muted)", fontSize: "0.8125rem" }}>…</span>
        ) : (
          <button
            key={r}
            style={r === page ? active : btn}
            onClick={() => onChange(r as number)}
          >
            {r}
          </button>
        )
      )}
      <button
        style={page === pages ? disabled : btn}
        disabled={page === pages}
        onClick={() => onChange(page + 1)}
      >
        →
      </button>
    </div>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function CategoriesPage() {
  const [rows, setRows]           = useState<Category[]>([]);
  const [total, setTotal]         = useState(0);
  const [page, setPage]           = useState(1);
  const [loading, setLoading]     = useState(true);
  const [modal, setModal]         = useState<"create" | "edit" | null>(null);
  const [editing, setEditing]     = useState<Category | null>(null);
  const [form, setForm]           = useState<FormState>(EMPTY);
  const [saving, setSaving]       = useState(false);
  const [error, setError]         = useState("");
  const [deleting, setDeleting]   = useState<string | null>(null);
  const [selected, setSelected]   = useState<Set<string>>(new Set());
  const [bulkDeleting, setBulkDeleting] = useState(false);

  const pages = Math.ceil(total / PAGE_SIZE) || 1;

  const load = useCallback((p = page) => {
    setLoading(true);
    getCategories({ limit: PAGE_SIZE, offset: (p - 1) * PAGE_SIZE })
      .then((r) => { setRows(r.data); setTotal(r.total ?? r.data.length); })
      .catch(console.error)
      .finally(() => setLoading(false));
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page]);

  useEffect(() => { load(); }, [load]);
  useEffect(() => { setSelected(new Set()); }, [rows]);

  function goTo(p: number) {
    setPage(p);
    setSelected(new Set());
  }

  function openCreate() {
    setEditing(null); setForm(EMPTY); setError(""); setModal("create");
  }

  function openEdit(cat: Category) {
    setEditing(cat);
    setForm({ name: cat.name, slug: cat.slug, parent_id: cat.parent_id ?? "" });
    setError(""); setModal("edit");
  }

  function handleNameChange(name: string) {
    setForm((f) => ({ ...f, name, slug: f.slug || slugify(name) }));
  }

  async function handleSave() {
    setError(""); setSaving(true);
    const token = getToken();
    if (!token) { setError("Not authenticated"); setSaving(false); return; }
    const body = { name: form.name.trim(), slug: form.slug.trim(), parent_id: form.parent_id || undefined };
    try {
      if (modal === "create") await createCategory(token, body);
      else if (editing) await updateCategory(token, editing.id, body);
      setModal(null); load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally { setSaving(false); }
  }

  async function handleDelete(id: string) {
    if (!await showConfirm("Delete this category?")) return;
    const token = getToken();
    if (!token) return;
    setDeleting(id);
    try { await deleteCategory(token, id); load(); }
    catch (e: unknown) { await showAlert(e instanceof Error ? e.message : "Delete failed"); }
    finally { setDeleting(null); }
  }

  async function handleBulkDelete() {
    const ids = [...selected];
    if (!ids.length) return;

    const hasParents = ids.some((id) => rows.some((r) => r.parent_id === id));
    const msg = hasParents
      ? `Delete ${ids.length} selected categor${ids.length === 1 ? "y" : "ies"}?\n\nWarning: some selected categories have subcategories — those subcategories will become root-level.`
      : `Delete ${ids.length} selected categor${ids.length === 1 ? "y" : "ies"}?`;
    if (!await showConfirm(msg)) return;

    const token = getToken();
    if (!token) return;
    setBulkDeleting(true);
    const errors: string[] = [];
    for (const id of ids) {
      try { await deleteCategory(token, id); }
      catch (e: unknown) {
        const name = rows.find((r) => r.id === id)?.name ?? id.slice(0, 8);
        errors.push(`${name}: ${e instanceof Error ? e.message : "failed"}`);
      }
    }
    setBulkDeleting(false);
    if (errors.length) await showAlert(`${ids.length - errors.length} deleted, ${errors.length} failed:\n${errors.join("\n")}`);
    load();
  }

  function toggleRow(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  }

  function toggleAll() {
    setSelected((prev) =>
      prev.size === rows.length ? new Set() : new Set(rows.map((r) => r.id))
    );
  }

  function parentName(parentId?: string) {
    if (!parentId) return <span style={{ color: "var(--muted)" }}>—</span>;
    return rows.find((r) => r.id === parentId)?.name
      ?? <span className="mono" style={{ color: "var(--muted)", fontSize: "0.75rem" }}>{parentId.slice(0, 8)}</span>;
  }

  const allSelected = rows.length > 0 && selected.size === rows.length;
  const someSelected = selected.size > 0 && !allSelected;

  return (
    <div style={{ padding: "2rem" }}>
      <div className="page-header">
        <div>
          <h1 className="page-title">Categories</h1>
          <p className="page-subtitle">{total} categor{total === 1 ? "y" : "ies"}</p>
        </div>
        <div style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
          <ExportButton
            filename="categories.csv"
            headers={CAT_EXPORT_HEADERS}
            fetchAll={async () => {
              const r = await getCategories({ limit: 10000, offset: 0 });
              return r.data as unknown as Record<string, unknown>[];
            }}
          />
          <ImportButton
            title="Categories"
            expectedHeaders={CAT_IMPORT_HEADERS}
            templateRow={CAT_TEMPLATE}
            importAll={async (csvRows) => {
              const token = getToken();
              if (!token) throw new Error("Not authenticated");

              // Seed name→id from categories already in the DB
              const nameToId = new Map<string, string>(rows.map((c) => [c.name, c.id]));

              // Pass 1: roots (no parent_name)
              const roots = csvRows.filter((r) => !r.parent_name?.trim());
              const children = csvRows.filter((r) => !!r.parent_name?.trim());

              const errors: { row: number; message: string }[] = [];
              let ok = 0;

              if (roots.length > 0) {
                try {
                  const created = await bulkCreateCategories(token, roots.map((r) => ({ name: r.name, slug: r.slug })));
                  created.forEach((c) => nameToId.set(c.name, c.id));
                  ok += created.length;
                } catch (e) {
                  errors.push({ row: 0, message: `Root categories: ${e instanceof Error ? e.message : "failed"}` });
                }
              }

              // Pass 2: children — resolve parent_id from the map built in pass 1
              if (children.length > 0) {
                const resolved: { name: string; slug: string; parent_id?: string }[] = [];
                children.forEach((r, i) => {
                  const parentName = r.parent_name.trim();
                  const parent_id = nameToId.get(parentName);
                  if (!parent_id) {
                    errors.push({ row: roots.length + i + 2, message: `Parent "${parentName}" not found` });
                  } else {
                    resolved.push({ name: r.name, slug: r.slug, parent_id });
                  }
                });

                if (resolved.length > 0) {
                  try {
                    const created = await bulkCreateCategories(token, resolved);
                    ok += created.length;
                  } catch (e) {
                    errors.push({ row: 0, message: `Subcategories: ${e instanceof Error ? e.message : "failed"}` });
                  }
                }
              }

              return { ok, errors };
            }}
            onDone={load}
          />
          <button className="btn btn-primary" onClick={openCreate}>+ New category</button>
        </div>
      </div>

      {/* Bulk action bar */}
      {selected.size > 0 && (
        <div className="bulk-bar" style={{
          display: "flex", alignItems: "center", gap: "0.75rem",
          padding: "0.625rem 1rem", marginBottom: "0.75rem",
          background: "var(--accent-bg)", border: "1px solid color-mix(in srgb, var(--accent) 30%, transparent)",
          borderRadius: 8, fontSize: "0.8125rem",
        }}>
          <span style={{ fontWeight: 600, color: "var(--accent)" }}>{selected.size} selected</span>
          <button
            className="btn btn-danger"
            style={{ padding: "0.3rem 0.875rem", fontSize: "0.8125rem" }}
            onClick={handleBulkDelete}
            disabled={bulkDeleting}
          >
            {bulkDeleting ? "Deleting…" : `Delete ${selected.size}`}
          </button>
          <button
            className="btn btn-ghost"
            style={{ padding: "0.3rem 0.75rem", fontSize: "0.8125rem" }}
            onClick={() => setSelected(new Set())}
          >
            Clear
          </button>
        </div>
      )}

      <div className="card" style={{ padding: 0, overflow: "hidden" }}>
        <table>
          <thead>
            <tr>
              <th style={{ width: 36, padding: "0.625rem 0.75rem" }}>
                <input
                  type="checkbox"
                  checked={allSelected}
                  ref={(el) => { if (el) el.indeterminate = someSelected; }}
                  onChange={toggleAll}
                  style={{ cursor: "pointer" }}
                />
              </th>
              <th>Name</th>
              <th>Slug</th>
              <th>Parent</th>
              <th>Created</th>
              <th style={{ width: 120 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <TableSkeleton rows={PAGE_SIZE} cols={6} />
            ) : rows.length === 0 ? (
              <tr>
                <td colSpan={6}>
                  <div className="empty-state">
                    <p className="empty-state-title">No categories yet</p>
                    <p className="empty-state-body">Create your first category to start organizing products.</p>
                  </div>
                </td>
              </tr>
            ) : (
              rows.map((cat) => {
                const checked = selected.has(cat.id);
                return (
                  <tr
                    key={cat.id}
                    style={{ background: checked ? "var(--accent-bg)" : undefined, cursor: "pointer" }}
                    onClick={() => toggleRow(cat.id)}
                  >
                    <td style={{ padding: "0.625rem 0.75rem" }} onClick={(e) => e.stopPropagation()}>
                      <input
                        type="checkbox"
                        checked={checked}
                        onChange={() => toggleRow(cat.id)}
                        style={{ cursor: "pointer" }}
                      />
                    </td>
                    <td style={{ fontWeight: 500 }}>{cat.name}</td>
                    <td><span className="mono" style={{ color: "var(--text-2)" }}>{cat.slug}</span></td>
                    <td>{parentName(cat.parent_id)}</td>
                    <td style={{ color: "var(--text-2)" }}>{new Date(cat.created_at).toLocaleDateString()}</td>
                    <td onClick={(e) => e.stopPropagation()}>
                      <div style={{ display: "flex", gap: "0.375rem" }}>
                        <button className="btn btn-ghost" style={{ padding: "0.25rem 0.625rem", fontSize: "0.75rem" }} onClick={() => openEdit(cat)}>Edit</button>
                        <button
                          className="btn btn-danger"
                          style={{ padding: "0.25rem 0.625rem", fontSize: "0.75rem" }}
                          onClick={() => handleDelete(cat.id)}
                          disabled={deleting === cat.id}
                        >
                          {deleting === cat.id ? "…" : "Delete"}
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginTop: "1rem" }}>
        <span style={{ fontSize: "0.8125rem", color: "var(--muted)" }}>
          {total > 0
            ? `${(page - 1) * PAGE_SIZE + 1}–${Math.min(page * PAGE_SIZE, total)} of ${total}`
            : ""}
        </span>
        <Pagination page={page} pages={pages} onChange={goTo} />
      </div>

      {/* Modal */}
      {modal && (
        <div className="modal-overlay" onClick={() => setModal(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2 className="modal-title">{modal === "create" ? "New category" : "Edit category"}</h2>
              <p className="modal-subtitle">{modal === "create" ? "Add a product category" : `Editing "${editing?.name}"`}</p>
            </div>
            <div className="modal-body">
              <div style={{ display: "flex", flexDirection: "column", gap: "0.875rem" }}>
                <div className="form-group" style={{ marginBottom: 0 }}>
                  <label className="form-label">Name</label>
                  <input className="input" value={form.name} onChange={(e) => handleNameChange(e.target.value)} placeholder="Electronics" autoFocus />
                </div>
                <div className="form-group" style={{ marginBottom: 0 }}>
                  <label className="form-label">Slug</label>
                  <input className="input" value={form.slug} onChange={(e) => setForm((f) => ({ ...f, slug: e.target.value }))} placeholder="electronics" />
                </div>
                <div className="form-group" style={{ marginBottom: 0 }}>
                  <label className="form-label">Parent category (optional)</label>
                  <select className="input" value={form.parent_id} onChange={(e) => setForm((f) => ({ ...f, parent_id: e.target.value }))}>
                    <option value="">None (root)</option>
                    {rows.filter((r) => r.id !== editing?.id).map((r) => (
                      <option key={r.id} value={r.id}>{r.name}</option>
                    ))}
                  </select>
                </div>
                {error && <p style={{ color: "var(--danger)", fontSize: "0.8rem", margin: 0 }}>{error}</p>}
                <div style={{ display: "flex", gap: "0.625rem", justifyContent: "flex-end", marginTop: "0.25rem" }}>
                  <button className="btn btn-ghost" onClick={() => setModal(null)}>Cancel</button>
                  <button className="btn btn-primary" onClick={handleSave} disabled={saving || !form.name || !form.slug}>
                    {saving ? "Saving…" : modal === "create" ? "Create" : "Save changes"}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
