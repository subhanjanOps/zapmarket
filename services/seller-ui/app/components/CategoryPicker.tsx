"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { ChevronDown, ChevronRight, Check, Layers, Tag, X } from "lucide-react";
import { getCategories, type Category } from "@/lib/api";

const PAGE_SIZE = 20;

type FetchFn = (query: string, offset: number) => Promise<{ items: Category[]; total: number }>;

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

// ── Single-level searchable combobox with lazy + paginated API fetch ──────────

interface ComboProps {
  placeholder: string;
  fetchFn: FetchFn;
  value: string;
  selectedLabel?: string;
  onSelect: (cat: Category | null) => void;
  disabled?: boolean;
  icon?: React.ReactNode;
  label?: string;
}

function Combobox({ placeholder, fetchFn, value, selectedLabel, onSelect, disabled, icon, label }: ComboProps) {
  const [open, setOpen]             = useState(false);
  const [query, setQuery]           = useState("");
  const [items, setItems]           = useState<Category[]>([]);
  const [total, setTotal]           = useState(0);
  const [loading, setLoading]       = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const wrapRef                     = useRef<HTMLDivElement>(null);
  const inputRef                    = useRef<HTMLInputElement>(null);
  const sentinelRef                 = useRef<HTMLDivElement>(null);
  const fetchSeqRef                 = useRef(0);
  const activeQueryRef              = useRef("");

  const load = useCallback(async (q: string, offset: number) => {
    const seq = ++fetchSeqRef.current;
    const isAppend = offset > 0;
    if (isAppend) setLoadingMore(true); else setLoading(true);

    try {
      const result = await fetchFn(q, offset);
      if (seq !== fetchSeqRef.current) return; // stale
      setTotal(result.total);
      if (isAppend) {
        setItems((prev) => [...prev, ...result.items]);
      } else {
        setItems(result.items);
      }
    } catch {
      if (seq === fetchSeqRef.current) {
        if (!isAppend) setItems([]);
      }
    } finally {
      if (seq === fetchSeqRef.current) {
        if (isAppend) setLoadingMore(false); else setLoading(false);
      }
    }
  }, [fetchFn]);

  // Fetch on open
  useEffect(() => {
    if (!open) return;
    activeQueryRef.current = "";
    setQuery("");
    setItems([]);
    setTotal(0);
    load("", 0);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  // Debounced search
  useEffect(() => {
    if (!open) return;
    const trimmed = query.trim();
    if (trimmed === activeQueryRef.current) return;
    const t = setTimeout(() => {
      activeQueryRef.current = trimmed;
      setItems([]);
      setTotal(0);
      load(trimmed, 0);
    }, 300);
    return () => clearTimeout(t);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query, open]);

  // IntersectionObserver for infinite scroll
  useEffect(() => {
    if (!sentinelRef.current) return;
    const obs = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && !loading && !loadingMore && items.length < total) {
          load(activeQueryRef.current, items.length);
        }
      },
      { threshold: 0.1 },
    );
    obs.observe(sentinelRef.current);
    return () => obs.disconnect();
  }, [items.length, total, loading, loadingMore, load]);

  // Close on outside click
  useEffect(() => {
    if (!open) return;
    function onOut(e: MouseEvent) {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) {
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

  const hasValue = Boolean(value);
  const displayText = open ? query : (selectedLabel ?? "");

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
          e.preventDefault();
          if (open) {
            setOpen(false);
            setQuery("");
          } else {
            setOpen(true);
            setTimeout(() => inputRef.current?.focus(), 0);
          }
        }}
      >
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

      {/* Dropdown via portal */}
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
            {!loading && items.length === 0 && (
              <div style={{ padding: "0.875rem 1rem", color: "var(--muted)", fontSize: "0.8125rem", fontStyle: "italic" }}>
                {query ? `No matches for "${query}"` : "Nothing here"}
              </div>
            )}
            {!loading && items.map((cat, i) => {
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
                    borderBottom: i < items.length - 1 ? "1px solid var(--border)" : "none",
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

            {/* Sentinel for infinite scroll */}
            <div ref={sentinelRef} style={{ height: 1 }} />

            {loadingMore && (
              <div style={{ padding: "0.5rem 1rem", color: "var(--muted)", fontSize: "0.75rem", textAlign: "center" }}>
                <span className="spinner" style={{ width: 12, height: 12, borderWidth: 2, display: "inline-block", verticalAlign: "middle", marginRight: 6 }} />
                Loading more…
              </div>
            )}
          </div>

          {!loading && items.length > 0 && (
            <div style={{ padding: "0.3rem 0.875rem", borderTop: "1px solid var(--border)", fontSize: "0.7rem", color: "var(--muted)" }}>
              {items.length < total
                ? `${items.length} of ${total} — scroll for more`
                : `${items.length} result${items.length !== 1 ? "s" : ""}`}
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
  const [selectedRoot, setSelectedRoot] = useState<Category | null>(null);
  const [selectedSub,  setSelectedSub]  = useState<Category | null>(null);
  const [subsExist,    setSubsExist]    = useState(false);
  const prefillDoneRef                  = useRef(false);

  // Resolve pre-filled value (only runs when value is set on mount)
  useEffect(() => {
    if (!value || prefillDoneRef.current) return;
    prefillDoneRef.current = true;
    getCategories({ limit: 200, offset: 0 }).then((allRes) => {
      const cats = allRes.categories;
      const roots = cats.filter((c) => !c.parent_id);
      const asRoot = roots.find((r) => r.id === value);
      if (asRoot) {
        setSelectedRoot(asRoot);
        // Check if it has subcategories
        getCategories({ parent_id: asRoot.id, limit: 1 }).then((r) => {
          setSubsExist(r.total > 0);
        }).catch(() => {});
        return;
      }
      const sub = cats.find((c) => c.id === value);
      if (sub?.parent_id) {
        const parent = roots.find((r) => r.id === sub.parent_id);
        if (parent) {
          setSelectedRoot(parent);
          setSelectedSub(sub);
          setSubsExist(true);
        }
      }
    }).catch(() => {});
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value]);

  // fetchFn for root combobox: load all roots once, filter in memory.
  // Roots are bounded (~10-50), so one fetch + client-side search is correct.
  // API search can't be used here because it returns all levels — subcategories
  // would be filtered out by !parent_id, giving misleading empty results.
  const rootsCacheRef = useRef<Category[]>([]);
  const rootFetchFn: FetchFn = useCallback(async (query, _offset) => {
    if (rootsCacheRef.current.length === 0) {
      const r = await getCategories({ limit: 300, offset: 0 });
      rootsCacheRef.current = r.categories.filter((c) => !c.parent_id);
    }
    const q = query.toLowerCase();
    const filtered = q
      ? rootsCacheRef.current.filter((c) => c.name.toLowerCase().includes(q))
      : rootsCacheRef.current;
    return { items: filtered, total: filtered.length };
  }, []);

  // fetchFn for subcategory combobox: paginated under selected root
  const subFetchFn: FetchFn = useCallback(async (query, offset) => {
    if (!selectedRoot) return { items: [], total: 0 };
    const r = await getCategories({ parent_id: selectedRoot.id, search: query || undefined, limit: PAGE_SIZE, offset });
    return { items: r.categories, total: r.total };
  }, [selectedRoot]);

  const handleRootSelect = useCallback(async (cat: Category | null) => {
    setSelectedRoot(cat);
    setSelectedSub(null);
    setSubsExist(false);
    if (!cat) { onChange(""); return; }
    onChange(cat.id);
    // Peek if subcategories exist
    try {
      const r = await getCategories({ parent_id: cat.id, limit: 1 });
      setSubsExist(r.total > 0);
    } catch { /* ignore */ }
  }, [onChange]);

  const handleSubSelect = useCallback((cat: Category | null) => {
    setSelectedSub(cat);
    if (cat) onChange(cat.id);
    else if (selectedRoot) onChange(selectedRoot.id);
    else onChange("");
  }, [onChange, selectedRoot]);

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
      {/* Level 1 — root/parent */}
      <Combobox
        label="Category"
        icon={<Layers size={11} />}
        placeholder="Search category…"
        fetchFn={rootFetchFn}
        value={selectedRoot?.id ?? ""}
        selectedLabel={selectedRoot?.name}
        onSelect={handleRootSelect}
      />

      {/* Connector line */}
      {selectedRoot && subsExist && (
        <div style={{ display: "flex", alignItems: "center", gap: "0.5rem", padding: "0 0.25rem", animation: "sub-slide 0.2s ease" }}>
          <div style={{ width: 1, height: 16, background: "var(--border)", marginLeft: 18 }} />
          <ChevronRight size={11} style={{ color: "var(--muted)" }} />
          <span style={{ fontSize: "0.7rem", color: "var(--muted)" }}>then pick a subcategory</span>
        </div>
      )}

      {/* Level 2 — subcategory */}
      {selectedRoot && subsExist && (
        <div style={{ paddingLeft: "1.25rem", borderLeft: "2px solid var(--accent-bg)", animation: "sub-slide 0.22s cubic-bezier(0.16,1,0.3,1) both" }}>
          <Combobox
            label="Subcategory"
            icon={<Tag size={11} />}
            placeholder="Search subcategory (optional)…"
            fetchFn={subFetchFn}
            value={selectedSub?.id ?? ""}
            selectedLabel={selectedSub?.name}
            onSelect={handleSubSelect}
          />
        </div>
      )}

      {/* No-subs hint */}
      {selectedRoot && !subsExist && (
        <p style={{ margin: "0.125rem 0 0", fontSize: "0.75rem", color: "var(--muted)", display: "flex", alignItems: "center", gap: "0.3rem" }}>
          <Check size={11} style={{ color: "var(--success)" }} />
          Using &ldquo;{selectedRoot.name}&rdquo; directly — no subcategories.
        </p>
      )}
    </div>
  );
}
