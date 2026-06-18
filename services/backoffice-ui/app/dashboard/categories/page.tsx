"use client";

import { useEffect, useState, useCallback } from "react";
import { getToken } from "@/lib/auth";
import {
  getCategories, createCategory, updateCategory, deleteCategory,
  type Category,
} from "@/lib/api";
import { TableSkeleton } from "@/app/components/Skeleton";

type FormState = { name: string; slug: string; parent_id: string };
const EMPTY: FormState = { name: "", slug: "", parent_id: "" };

function slugify(s: string) {
  return s.toLowerCase().replace(/\s+/g, "-").replace(/[^a-z0-9-]/g, "");
}

export default function CategoriesPage() {
  const [rows, setRows]         = useState<Category[]>([]);
  const [total, setTotal]       = useState(0);
  const [loading, setLoading]   = useState(true);
  const [modal, setModal]       = useState<"create" | "edit" | null>(null);
  const [editing, setEditing]   = useState<Category | null>(null);
  const [form, setForm]         = useState<FormState>(EMPTY);
  const [saving, setSaving]     = useState(false);
  const [error, setError]       = useState("");
  const [deleting, setDeleting] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    getCategories()
      .then((r) => { setRows(r.data); setTotal(r.total ?? r.data.length); })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  function openCreate() {
    setEditing(null);
    setForm(EMPTY);
    setError("");
    setModal("create");
  }

  function openEdit(cat: Category) {
    setEditing(cat);
    setForm({ name: cat.name, slug: cat.slug, parent_id: cat.parent_id ?? "" });
    setError("");
    setModal("edit");
  }

  function handleNameChange(name: string) {
    setForm((f) => ({ ...f, name, slug: f.slug || slugify(name) }));
  }

  async function handleSave() {
    setError("");
    setSaving(true);
    const token = getToken();
    if (!token) { setError("Not authenticated"); setSaving(false); return; }
    const body = { name: form.name.trim(), slug: form.slug.trim(), parent_id: form.parent_id || undefined };
    try {
      if (modal === "create") {
        await createCategory(token, body);
      } else if (editing) {
        await updateCategory(token, editing.id, body);
      }
      setModal(null);
      load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: string) {
    if (!confirm("Delete this category?")) return;
    const token = getToken();
    if (!token) return;
    setDeleting(id);
    try { await deleteCategory(token, id); load(); }
    catch (e: unknown) { alert(e instanceof Error ? e.message : "Delete failed"); }
    finally { setDeleting(null); }
  }

  function parentName(parentId?: string) {
    if (!parentId) return <span style={{ color: "var(--muted)" }}>—</span>;
    return rows.find((r) => r.id === parentId)?.name ?? <span className="mono" style={{ color: "var(--muted)", fontSize: "0.75rem" }}>{parentId.slice(0, 8)}</span>;
  }

  return (
    <div style={{ padding: "2rem" }}>
      <div className="page-header">
        <div>
          <h1 className="page-title">Categories</h1>
          <p className="page-subtitle">{total} categor{total === 1 ? "y" : "ies"}</p>
        </div>
        <button className="btn btn-primary" onClick={openCreate}>+ New category</button>
      </div>

      <div className="card" style={{ padding: 0, overflow: "hidden" }}>
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Slug</th>
              <th>Parent</th>
              <th>Created</th>
              <th style={{ width: 120 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <TableSkeleton rows={6} cols={5} />
            ) : rows.length === 0 ? (
              <tr>
                <td colSpan={5}>
                  <div className="empty-state">
                    <p className="empty-state-title">No categories yet</p>
                    <p className="empty-state-body">Create your first category to start organizing products.</p>
                  </div>
                </td>
              </tr>
            ) : (
              rows.map((cat) => (
                <tr key={cat.id}>
                  <td style={{ fontWeight: 500 }}>{cat.name}</td>
                  <td><span className="mono" style={{ color: "var(--text-2)" }}>{cat.slug}</span></td>
                  <td>{parentName(cat.parent_id)}</td>
                  <td style={{ color: "var(--text-2)" }}>{new Date(cat.created_at).toLocaleDateString()}</td>
                  <td>
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
              ))
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
