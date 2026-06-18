"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { getCategories, type Category } from "@/lib/api";

// ── Mini searchable combobox used for each level ──────────────────────────────

interface ComboboxProps {
  placeholder: string;
  items: Category[];
  loading: boolean;
  value: string;
  onSelect: (cat: Category | null) => void;
  onSearch?: (q: string) => void;
  disabled?: boolean;
}

function Combobox({ placeholder, items, loading, value, onSelect, onSearch, disabled }: ComboboxProps) {
  const [open, setOpen]   = useState(false);
  const [query, setQuery] = useState("");
  const ref               = useRef<HTMLDivElement>(null);
  const selected          = items.find((c) => c.id === value);

  useEffect(() => {
    function onOut(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
        setQuery("");
      }
    }
    document.addEventListener("mousedown", onOut);
    return () => document.removeEventListener("mousedown", onOut);
  }, []);

  const displayed = open ? query : (selected?.name ?? "");

  const filtered = query.trim()
    ? items.filter((c) => c.name.toLowerCase().includes(query.toLowerCase()))
    : items;

  function handleSelect(cat: Category) {
    onSelect(cat);
    setOpen(false);
    setQuery("");
  }

  function handleClear(e: React.MouseEvent) {
    e.stopPropagation();
    onSelect(null);
    setQuery("");
    setOpen(false);
  }

  return (
    <div ref={ref} style={{ position: "relative" }}>
      <div style={{ position: "relative" }}>
        <input
          className="input"
          style={{ width: "100%", boxSizing: "border-box", paddingRight: "2rem", opacity: disabled ? 0.5 : 1 }}
          placeholder={placeholder}
          value={displayed}
          disabled={disabled}
          onChange={(e) => {
            setQuery(e.target.value);
            onSearch?.(e.target.value);
            if (!open) setOpen(true);
          }}
          onFocus={() => { if (!disabled) setOpen(true); }}
          autoComplete="off"
        />
        <span
          style={{ position: "absolute", right: "0.625rem", top: "50%", transform: "translateY(-50%)", color: "var(--muted)", cursor: disabled ? "default" : "pointer", fontSize: "0.75rem", userSelect: "none" }}
          onMouseDown={(e) => {
            e.preventDefault();
            if (disabled) return;
            if (value) handleClear(e);
            else setOpen((o) => !o);
          }}
        >
          {value ? "✕" : "▾"}
        </span>
      </div>

      {open && !disabled && (
        <div style={{
          position: "absolute", top: "calc(100% + 4px)", left: 0, right: 0, zIndex: 1000,
          background: "var(--surface)", border: "1px solid var(--border)", borderRadius: 8,
          boxShadow: "0 8px 24px rgba(0,0,0,0.18)", maxHeight: 240, overflowY: "auto",
          fontSize: "0.8125rem",
        }}>
          {loading && <div style={{ padding: "0.75rem 1rem", color: "var(--muted)" }}>Loading…</div>}
          {!loading && filtered.length === 0 && (
            <div style={{ padding: "0.75rem 1rem", color: "var(--muted)", fontStyle: "italic" }}>
              {query ? `No results for "${query}"` : "No options"}
            </div>
          )}
          {!loading && filtered.map((cat) => (
            <div
              key={cat.id}
              onMouseDown={(e) => { e.preventDefault(); handleSelect(cat); }}
              style={{
                padding: "0.5rem 1rem", cursor: "pointer",
                background: cat.id === value ? "var(--accent-bg)" : "transparent",
                color: cat.id === value ? "var(--accent)" : "var(--text)",
                fontWeight: cat.id === value ? 500 : 400,
              }}
              onMouseEnter={(e) => { if (cat.id !== value) (e.currentTarget as HTMLElement).style.background = "var(--surface2)"; }}
              onMouseLeave={(e) => { if (cat.id !== value) (e.currentTarget as HTMLElement).style.background = "transparent"; }}
            >
              {cat.name}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ── CategoryPicker: two-step root → subcategory ───────────────────────────────

interface Props {
  value: string;
  onChange: (id: string) => void;
}

export default function CategoryPicker({ value, onChange }: Props) {
  const [roots, setRoots]           = useState<Category[]>([]);
  const [rootsLoading, setRootsLoading] = useState(true);
  const [selectedRoot, setSelectedRoot] = useState<Category | null>(null);

  const [subs, setSubs]             = useState<Category[]>([]);
  const [subsLoading, setSubsLoading] = useState(false);
  const [selectedSub, setSelectedSub] = useState<Category | null>(null);

  // Load all root categories once
  useEffect(() => {
    setRootsLoading(true);
    getCategories({ limit: 200, offset: 0 })
      .then((r) => setRoots(r.categories.filter((c) => !c.parent_id)))
      .catch(() => {})
      .finally(() => setRootsLoading(false));
  }, []);

  // When value is pre-filled externally, resolve root + sub
  useEffect(() => {
    if (!value) { setSelectedRoot(null); setSelectedSub(null); return; }
    // If it matches a root we already have
    const root = roots.find((r) => r.id === value);
    if (root) { setSelectedRoot(root); setSelectedSub(null); return; }
    // Otherwise it might be a subcategory — fetch it
    getCategories({ limit: 1, offset: 0 })
      .then(async (r) => {
        // Try fetching the full list to find the sub
        const sub = r.categories.find((c) => c.id === value);
        if (sub?.parent_id) {
          const parentRoot = roots.find((r) => r.id === sub.parent_id);
          if (parentRoot) {
            setSelectedRoot(parentRoot);
            const subsRes = await getCategories({ parent_id: parentRoot.id, limit: 200 });
            setSubs(subsRes.categories);
            setSelectedSub(sub);
          }
        }
      })
      .catch(() => {});
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value, roots]);

  // Load subcategories when root changes
  async function handleRootSelect(cat: Category | null) {
    setSelectedRoot(cat);
    setSelectedSub(null);
    setSubs([]);
    if (!cat) { onChange(""); return; }
    // Immediately set to root; will update to sub if one is chosen
    onChange(cat.id);
    setSubsLoading(true);
    try {
      const r = await getCategories({ parent_id: cat.id, limit: 200 });
      setSubs(r.categories);
    } catch { /* ignore */ }
    finally { setSubsLoading(false); }
  }

  function handleSubSelect(cat: Category | null) {
    setSelectedSub(cat);
    if (cat) onChange(cat.id);
    else if (selectedRoot) onChange(selectedRoot.id);
    else onChange("");
  }

  const hasSubs = subs.length > 0 || subsLoading;

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: "0.625rem" }}>
      <Combobox
        placeholder="Search parent category…"
        items={roots}
        loading={rootsLoading}
        value={selectedRoot?.id ?? ""}
        onSelect={handleRootSelect}
      />
      {selectedRoot && hasSubs && (
        <Combobox
          placeholder="Search subcategory (optional)…"
          items={subs}
          loading={subsLoading}
          value={selectedSub?.id ?? ""}
          onSelect={handleSubSelect}
        />
      )}
      {selectedRoot && !subsLoading && subs.length === 0 && (
        <p style={{ margin: 0, fontSize: "0.75rem", color: "var(--muted)" }}>
          No subcategories — "{selectedRoot.name}" will be used directly.
        </p>
      )}
    </div>
  );
}
