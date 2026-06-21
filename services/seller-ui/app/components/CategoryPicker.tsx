"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { ChevronDown, ChevronRight, Check, Layers, Tag, X } from "lucide-react";
import { getCategories, type Category } from "@/lib/api";

// ── Dropdown portal — renders outside card/overflow containers ────────────────

function DropdownPortal({ anchorRef, children, open }: {
  anchorRef: React.RefObject<HTMLDivElement | null>;
  children: React.ReactNode;
  open: boolean;
}) {
  const [rect, setRect] = useState<DOMRect | null>(null);

  useEffect(() => {
    if (!open || !anchorRef.current) return;
    const update = () => {
      if (anchorRef.current) setRect(anchorRef.current.getBoundingClientRect());
    };
    update();
    window.addEventListener("scroll", update, true);
    window.addEventListener("resize", update);
    return () => {
      window.removeEventListener("scroll", update, true);
      window.removeEventListener("resize", update);
    };
  }, [open, anchorRef]);

  if (!open || !rect || typeof document === "undefined") return null;

  const spaceBelow = window.innerHeight - rect.bottom;
  const dropUp = spaceBelow < 260 && rect.top > 260;

  return createPortal(
    <div
      style={{
        position: "fixed",
        top: dropUp ? undefined : rect.bottom + 4,
        bottom: dropUp ? window.innerHeight - rect.top + 4 : undefined,
        left: rect.left,
        width: rect.width,
        zIndex: 9999,
        animation: "picker-drop 0.18s cubic-bezier(0.16,1,0.3,1) both",
      }}
    >
      {children}
    </div>,
    document.body,
  );
}

// ── Single-level searchable combobox ──────────────────────────────────────────

interface ComboProps {
  placeholder: string;
  items: Category[];
  loading: boolean;
  value: string;
  onSelect: (cat: Category | null) => void;
  disabled?: boolean;
  icon?: React.ReactNode;
  label?: string;
}

function Combobox({ placeholder, items, loading, value, onSelect, disabled, icon, label }: ComboProps) {
  const [open, setOpen]   = useState(false);
  const [query, setQuery] = useState("");
  const wrapRef           = useRef<HTMLDivElement>(null);
  const inputRef          = useRef<HTMLInputElement>(null);
  const selected          = items.find((c) => c.id === value);
  const displayText       = open ? query : (selected?.name ?? "");

  const filtered = query.trim()
    ? items.filter((c) => c.name.toLowerCase().includes(query.toLowerCase()))
    : items;

  // Close on outside click
  useEffect(() => {
    if (!open) return;
    function onOut(e: MouseEvent) {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) {
        // also check if click is inside the portal
        const inPortal = (e.target as Element)?.closest("[data-picker-portal]");
        if (!inPortal) { setOpen(false); setQuery(""); }
      }
    }
    document.addEventListener("mousedown", onOut);
    return () => document.removeEventListener("mousedown", onOut);
  }, [open]);

  function handleSelect(cat: Category) {
    onSelect(cat);
    setOpen(false);
    setQuery("");
  }

  function handleClear(e: React.MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    onSelect(null);
    setQuery("");
    setOpen(false);
  }

  const hasValue = Boolean(value && selected);

  return (
    <div ref={wrapRef} style={{ position: "relative" }}>
      {label && (
        <div style={{ fontSize: "0.6875rem", fontWeight: 500, color: "var(--muted)", textTransform: "uppercase", letterSpacing: "0.06em", marginBottom: "0.3rem", display: "flex", alignItems: "center", gap: "0.3rem" }}>
          {icon} {label}
        </div>
      )}
      {/* Trigger */}
      <div
        role="combobox"
        aria-expanded={open}
        style={{
          display: "flex", alignItems: "center",
          background: hasValue ? "var(--accent-bg)" : "var(--surface2)",
          border: `1px solid ${open ? "var(--accent)" : hasValue ? "color-mix(in srgb,var(--accent) 35%,transparent)" : "var(--border)"}`,
          borderRadius: 8,
          boxShadow: open ? "var(--focus-ring)" : "none",
          padding: "0 0.5rem 0 0.75rem",
          height: 38,
          cursor: disabled ? "default" : "pointer",
          opacity: disabled ? 0.45 : 1,
          transition: "border-color 0.15s, background 0.15s, box-shadow 0.15s",
          gap: "0.5rem",
        }}
        onMouseDown={(e) => {
          if (disabled) return;
          e.preventDefault(); // prevent focus-then-click double-fire
          if (open) {
            setOpen(false);
            setQuery("");
          } else {
            setOpen(true);
            // schedule focus so input is ready after state update
            setTimeout(() => inputRef.current?.focus(), 0);
          }
        }}
      >
        {/* Search icon / category icon */}
        <span style={{ color: hasValue ? "var(--accent)" : "var(--muted)", flexShrink: 0, display: "flex" }}>
          {hasValue ? <Check size={13} strokeWidth={2.5} /> : <Tag size={13} strokeWidth={1.75} />}
        </span>

        <input
          ref={inputRef}
          value={displayText}
          placeholder={placeholder}
          disabled={disabled}
          autoComplete="off"
          onChange={(e) => {
            setQuery(e.target.value);
            if (!open) setOpen(true);
          }}
          onFocus={() => { if (!disabled && !open) setOpen(true); }}
          onClick={(e) => e.stopPropagation()}
          onKeyDown={(e) => { if (e.key === "Escape") { setOpen(false); setQuery(""); } }}
          style={{
            flex: 1, background: "transparent", border: "none", outline: "none",
            color: hasValue && !open ? "var(--accent)" : "var(--text)",
            fontSize: "0.8125rem", fontFamily: "inherit",
            fontWeight: hasValue && !open ? 500 : 400,
            minWidth: 0,
          }}
        />

        {/* Clear / chevron */}
        {hasValue ? (
          <button
            onMouseDown={handleClear}
            style={{ background: "none", border: "none", cursor: "pointer", color: "var(--muted)", display: "flex", padding: 3, borderRadius: 4, flexShrink: 0 }}
            tabIndex={-1}
            title="Clear"
          >
            <X size={12} />
          </button>
        ) : (
          <ChevronDown
            size={14}
            style={{ color: "var(--muted)", flexShrink: 0, transition: "transform 0.15s", transform: open ? "rotate(180deg)" : "none" }}
          />
        )}
      </div>

      {/* Dropdown via portal to escape overflow:hidden */}
      <DropdownPortal anchorRef={wrapRef} open={open}>
        <div
          data-picker-portal
          style={{
            background: "var(--surface)",
            border: "1px solid var(--border-strong)",
            borderRadius: 10,
            boxShadow: "var(--shadow-lg)",
            overflow: "hidden",
          }}
        >
          <div style={{ maxHeight: 240, overflowY: "auto" }}>
            {loading && (
              <div style={{ padding: "0.75rem 1rem", color: "var(--muted)", fontSize: "0.8125rem" }}>
                <span className="spinner" style={{ width: 14, height: 14, borderWidth: 2, display: "inline-block", verticalAlign: "middle", marginRight: 8 }} />
                Loading…
              </div>
            )}
            {!loading && filtered.length === 0 && (
              <div style={{ padding: "0.875rem 1rem", color: "var(--muted)", fontSize: "0.8125rem", fontStyle: "italic" }}>
                {query ? `No matches for "${query}"` : "Nothing here"}
              </div>
            )}
            {!loading && filtered.map((cat, i) => {
              const sel = cat.id === value;
              return (
                <div
                  key={cat.id}
                  onMouseDown={(e) => { e.preventDefault(); handleSelect(cat); }}
                  style={{
                    display: "flex", alignItems: "center", gap: "0.5rem",
                    padding: "0.5625rem 0.875rem",
                    cursor: "pointer",
                    background: sel ? "var(--accent-bg)" : "transparent",
                    color: sel ? "var(--accent)" : "var(--text)",
                    fontWeight: sel ? 500 : 400,
                    fontSize: "0.8125rem",
                    borderBottom: i < filtered.length - 1 ? "1px solid var(--border)" : "none",
                    transition: "background 0.08s",
                    animation: `item-in 0.1s ease ${Math.min(i * 0.02, 0.12)}s both`,
                  }}
                  onMouseEnter={(e) => { if (!sel) (e.currentTarget as HTMLElement).style.background = "var(--surface2)"; }}
                  onMouseLeave={(e) => { if (!sel) (e.currentTarget as HTMLElement).style.background = "transparent"; }}
                >
                  {sel
                    ? <Check size={13} strokeWidth={2.5} style={{ color: "var(--accent)", flexShrink: 0 }} />
                    : <div style={{ width: 13, flexShrink: 0 }} />}
                  {cat.name}
                </div>
              );
            })}
          </div>

          {!loading && items.length > 0 && (
            <div style={{ padding: "0.3rem 0.875rem", borderTop: "1px solid var(--border)", fontSize: "0.7rem", color: "var(--muted)" }}>
              {filtered.length} of {items.length}
            </div>
          )}
        </div>
      </DropdownPortal>
    </div>
  );
}

// ── CategoryPicker: parent → subcategory two-step ─────────────────────────────

interface Props {
  value: string;
  onChange: (id: string) => void;
}

export default function CategoryPicker({ value, onChange }: Props) {
  const [roots, setRoots]             = useState<Category[]>([]);
  const [rootsLoading, setRootsLoading] = useState(true);
  const [rootsError, setRootsError]   = useState("");
  const [selectedRoot, setSelectedRoot] = useState<Category | null>(null);

  const [subs, setSubs]               = useState<Category[]>([]);
  const [subsLoading, setSubsLoading] = useState(false);
  const [selectedSub, setSelectedSub] = useState<Category | null>(null);

  // Load all root categories once
  useEffect(() => {
    setRootsLoading(true);
    setRootsError("");
    getCategories({ limit: 300, offset: 0 })
      .then((r) => setRoots(r.categories.filter((c) => !c.parent_id)))
      .catch((e: unknown) => setRootsError(e instanceof Error ? e.message : "Failed to load categories"))
      .finally(() => setRootsLoading(false));
  }, []);

  // Resolve pre-filled value (e.g. when editing an existing product)
  useEffect(() => {
    if (!value || !roots.length) return;
    const asRoot = roots.find((r) => r.id === value);
    if (asRoot) { setSelectedRoot(asRoot); setSelectedSub(null); return; }
    // Could be a subcategory — fetch its parent
    getCategories({ limit: 200, offset: 0 }).then(async (allRes) => {
      const sub = allRes.categories.find((c) => c.id === value);
      if (sub?.parent_id) {
        const parent = roots.find((r) => r.id === sub.parent_id);
        if (parent) {
          setSelectedRoot(parent);
          const subsRes = await getCategories({ parent_id: parent.id, limit: 200 });
          setSubs(subsRes.categories);
          setSelectedSub(sub);
        }
      }
    }).catch(() => {});
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value, roots.length]);

  const handleRootSelect = useCallback(async (cat: Category | null) => {
    setSelectedRoot(cat);
    setSelectedSub(null);
    setSubs([]);
    if (!cat) { onChange(""); return; }
    onChange(cat.id);
    setSubsLoading(true);
    try {
      const r = await getCategories({ parent_id: cat.id, limit: 200 });
      setSubs(r.categories);
    } catch { /* ignore */ }
    finally { setSubsLoading(false); }
  }, [onChange]);

  const handleSubSelect = useCallback((cat: Category | null) => {
    setSelectedSub(cat);
    if (cat) onChange(cat.id);
    else if (selectedRoot) onChange(selectedRoot.id);
    else onChange("");
  }, [onChange, selectedRoot]);

  const hasSubs = subs.length > 0 || subsLoading;

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
      {rootsError && (
        <p style={{ margin: 0, fontSize: "0.75rem", color: "var(--danger)" }}>
          {rootsError}
        </p>
      )}
      {/* Level 1 — root/parent */}
      <Combobox
        label="Category"
        icon={<Layers size={11} />}
        placeholder="Search category…"
        items={roots}
        loading={rootsLoading}
        value={selectedRoot?.id ?? ""}
        onSelect={handleRootSelect}
      />

      {/* Connector line */}
      {selectedRoot && hasSubs && (
        <div style={{ display: "flex", alignItems: "center", gap: "0.5rem", padding: "0 0.25rem", animation: "sub-slide 0.2s ease" }}>
          <div style={{ width: 1, height: 16, background: "var(--border)", marginLeft: 18 }} />
          <ChevronRight size={11} style={{ color: "var(--muted)" }} />
          <span style={{ fontSize: "0.7rem", color: "var(--muted)" }}>then pick a subcategory</span>
        </div>
      )}

      {/* Level 2 — subcategory */}
      {selectedRoot && hasSubs && (
        <div style={{ paddingLeft: "1.25rem", borderLeft: "2px solid var(--accent-bg)", animation: "sub-slide 0.22s cubic-bezier(0.16,1,0.3,1) both" }}>
          <Combobox
            label="Subcategory"
            icon={<Tag size={11} />}
            placeholder="Search subcategory (optional)…"
            items={subs}
            loading={subsLoading}
            value={selectedSub?.id ?? ""}
            onSelect={handleSubSelect}
          />
        </div>
      )}

      {/* No-subs hint */}
      {selectedRoot && !subsLoading && subs.length === 0 && (
        <p style={{ margin: "0.125rem 0 0", fontSize: "0.75rem", color: "var(--muted)", display: "flex", alignItems: "center", gap: "0.3rem" }}>
          <Check size={11} style={{ color: "var(--success)" }} />
          Using "{selectedRoot.name}" directly — no subcategories.
        </p>
      )}
    </div>
  );
}
