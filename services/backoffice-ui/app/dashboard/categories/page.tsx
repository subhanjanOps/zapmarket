"use client";

import { useEffect, useState, useCallback } from "react";
import { getToken } from "@/lib/auth";
import {
  getCategories, createCategory, updateCategory, deleteCategory,
  type Category,
} from "@/lib/api";
import { TableSkeleton } from "@/app/components/Skeleton";
import { ExportButton, ImportButton } from "@/app/components/BulkIO";
import { showAlert, showConfirm } from "@/app/components/Dialog";

const CAT_EXPORT_HEADERS = ["id", "name", "slug", "parent_id", "created_at"];
const CAT_IMPORT_HEADERS = ["name", "slug", "parent_name"];
const CAT_TEMPLATE = { name: "Electronics", slug: "electronics", parent_name: "" };

type FormState = { name: string; slug: string; parent_id: string };
const EMPTY: FormState = { name: "", slug: "", parent_id: "" };

function slugify(s: string) {
  return s.toLowerCase().replace(/\s+/g, "-").replace(/[^a-z0-9-]/g, "");
}

export default function CategoriesPage() {
  const [rows, setRows]           = useState<Category[]>([]);
  const [total, setTotal]         = useState(0);
  const [loading, setLoading]     = useState(true);
  const [modal, setModal]         = useState<"create" | "edit" | null>(null);
  const [editing, setEditing]     = useState<Category | null>(null);
  const [form, setForm]           = useState<FormState>(EMPTY);
  const [saving, setSaving]       = useState(false);
  const [error, setError]         = useState("");
  const [deleting, setDeleting]   = useState<string | null>(null);
  const [selected, setSelected]   = useState<Set<string>>(new Set());
  const [bulkDeleting, setBulkDeleting] = useState(false);

  const load = useCallback(() => {
    setLoading(true);
    getCategories()
      .then((r) => { setRows(r.data); setTotal(r.total ?? r.data.length); })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);
  // Clear selection when rows reload
  useEffect(() => { setSelected(new Set()); }, [rows]);

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

    // Warn: deleting a parent also orphans its children
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
              const r = await getCategories();
              return r.data as unknown as Record<string, unknown>[];
            }}
          />
          <ImportButton
            title="Categories"
            expectedHeaders={CAT_IMPORT_HEADERS}
            templateRow={CAT_TEMPLATE}
            importRow={(() => {
              const nameToId = new Map<string, string>(rows.map((c) => [c.name, c.id]));
              return async (row: Record<string, string>) => {
                const token = getToken();
                if (!token) throw new Error("Not authenticated");
                const parentName = row.parent_name?.trim();
                const parent_id = parentName ? nameToId.get(parentName) : undefined;
                if (parentName && !parent_id) throw new Error(`Parent "${parentName}" not found`);
                const created = await createCategory(token, { name: row.name, slug: row.slug, parent_id });
                nameToId.set(created.name, created.id);
              };
            })()}
            onDone={load}
          />
          <button className="btn btn-primary" onClick={openCreate}>+ New category</button>
        </div>
      </div>

      {/* Bulk action bar — appears when rows are checked */}
      {selected.size > 0 && (
        <div style={{
          display: "flex", alignItems: "center", gap: "0.75rem",
          padding: "0.625rem 1rem", marginBottom: "0.75rem",
          background: "var(--accent-bg)", border: "1px solid color-mix(in srgb, var(--accent) 30%, transparent)",
          borderRadius: 8, fontSize: "0.8125rem",
        }}>
          <span style={{ fontWeight: 600, color: "var(--accent)" }}>
            {selected.size} selected
          </span>
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
              <TableSkeleton rows={6} cols={6} />
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
