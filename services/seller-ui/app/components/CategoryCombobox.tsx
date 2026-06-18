"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { getCategories, type Category } from "@/lib/api";

const PAGE_SIZE = 20;
const SEARCH_THRESHOLD = 3; // characters before API search fires

interface Props {
  value: string;
  onChange: (id: string) => void;
  style?: React.CSSProperties;
  placeholder?: string;
}

export default function CategoryCombobox({ value, onChange, style, placeholder = "Search or select category…" }: Props) {
  const [query, setQuery]         = useState("");
  const [open, setOpen]           = useState(false);
  const [items, setItems]         = useState<Category[]>([]);
  const [total, setTotal]         = useState(0);
  const [loading, setLoading]     = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [selectedName, setSelectedName] = useState("");

  const containerRef  = useRef<HTMLDivElement>(null);
  const inputRef      = useRef<HTMLInputElement>(null);
  const listRef       = useRef<HTMLDivElement>(null);
  const debounceTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const activeSearch  = useRef("");   // tracks what was fetched so we can discard stale responses

  // ── Fetch first page ────────────────────────────────────────────────────────

  const fetchFirst = useCallback(async (search: string) => {
    activeSearch.current = search;
    setLoading(true);
    setItems([]);
    setTotal(0);
    try {
      const params = search.length >= SEARCH_THRESHOLD ? { search, limit: PAGE_SIZE, offset: 0 } : { limit: PAGE_SIZE, offset: 0 };
      const r = await getCategories(params);
      if (activeSearch.current !== search) return; // stale
      setItems(r.categories);
      setTotal(r.total);
    } catch { /* ignore */ }
    finally { setLoading(false); }
  }, []);

  // ── Fetch next page (lazy scroll) ──────────────────────────────────────────

  const fetchMore = useCallback(async () => {
    if (loadingMore || items.length >= total) return;
    setLoadingMore(true);
    try {
      const search = activeSearch.current;
      const params = search.length >= SEARCH_THRESHOLD
        ? { search, limit: PAGE_SIZE, offset: items.length }
        : { limit: PAGE_SIZE, offset: items.length };
      const r = await getCategories(params);
      if (activeSearch.current !== search) return;
      setItems((prev) => {
        const ids = new Set(prev.map((c) => c.id));
        return [...prev, ...r.categories.filter((c) => !ids.has(c.id))];
      });
      setTotal(r.total);
    } catch { /* ignore */ }
    finally { setLoadingMore(false); }
  }, [items.length, total, loadingMore]);

  // ── Debounced query change ──────────────────────────────────────────────────

  function handleQueryChange(q: string) {
    setQuery(q);
    if (!q) onChange(""); // clear selection

    if (debounceTimer.current) clearTimeout(debounceTimer.current);

    if (q.length === 0 || q.length >= SEARCH_THRESHOLD) {
      debounceTimer.current = setTimeout(() => fetchFirst(q), q.length === 0 ? 0 : 350);
    }
    // 1–2 chars: wait silently for the 3rd character
  }

  // ── Infinite scroll sentinel ────────────────────────────────────────────────

  useEffect(() => {
    if (!open || !listRef.current) return;
    const list = listRef.current;
    function onScroll() {
      if (list.scrollHeight - list.scrollTop - list.clientHeight < 80) {
        fetchMore();
      }
    }
    list.addEventListener("scroll", onScroll);
    return () => list.removeEventListener("scroll", onScroll);
  }, [open, fetchMore]);

  // ── Open/close behaviour ────────────────────────────────────────────────────

  function handleFocus() {
    setOpen(true);
    setQuery("");
    fetchFirst("");
  }

  useEffect(() => {
    function onOutside(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
        setQuery("");
      }
    }
    document.addEventListener("mousedown", onOutside);
    return () => document.removeEventListener("mousedown", onOutside);
  }, []);

  // ── Resolve initial selected name ───────────────────────────────────────────

  useEffect(() => {
    if (!value) { setSelectedName(""); return; }
    // Check already-loaded items first
    const found = items.find((c) => c.id === value);
    if (found) { setSelectedName(found.name); return; }
    // Fetch by ID if not in list (e.g. on page load with pre-filled value)
    getCategories({ limit: 1, offset: 0 })
      .then((r) => {
        const match = r.categories.find((c) => c.id === value);
        if (match) setSelectedName(match.name);
      })
      .catch(() => {});
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value]);

  // ── Select an item ──────────────────────────────────────────────────────────

  function select(cat: Category) {
    onChange(cat.id);
    setSelectedName(cat.name);
    setQuery("");
    setOpen(false);
  }

  // ── Render ──────────────────────────────────────────────────────────────────

  const displayText = open ? query : selectedName;
  const parentMap = new Map(items.map((c) => [c.id, c.name]));

  // Group: roots + their children that are in the list
  const roots = items.filter((c) => !c.parent_id);
  const childrenOf = (id: string) => items.filter((c) => c.parent_id === id);
  const orphans = items.filter((c) => c.parent_id && !items.find((p) => p.id === c.parent_id));

  const hint = query.length > 0 && query.length < SEARCH_THRESHOLD
    ? `Type ${SEARCH_THRESHOLD - query.length} more character${SEARCH_THRESHOLD - query.length === 1 ? "" : "s"} to search…`
    : null;

  return (
    <div ref={containerRef} style={{ position: "relative", ...style }}>
      <div style={{ position: "relative" }}>
        <input
          ref={inputRef}
          value={displayText}
          onChange={(e) => handleQueryChange(e.target.value)}
          onFocus={handleFocus}
          placeholder={placeholder}
          autoComplete="off"
          className="input"
          style={{ width: "100%", boxSizing: "border-box", paddingRight: "2rem", ...(style ?? {}) }}
        />
        <span
          style={{ position: "absolute", right: "0.625rem", top: "50%", transform: "translateY(-50%)", color: "var(--muted)", cursor: "pointer", fontSize: "0.75rem", userSelect: "none" }}
          onMouseDown={(e) => {
            e.preventDefault();
            if (value) { onChange(""); setSelectedName(""); setQuery(""); inputRef.current?.focus(); }
            else { open ? setOpen(false) : (setOpen(true), fetchFirst("")); inputRef.current?.focus(); }
          }}
        >
          {value ? "✕" : "▾"}
        </span>
      </div>

      {open && (
        <div
          ref={listRef}
          style={{
            position: "absolute", top: "calc(100% + 4px)", left: 0, right: 0, zIndex: 999,
            background: "var(--surface)", border: "1px solid var(--border)", borderRadius: 8,
            boxShadow: "0 8px 24px rgba(0,0,0,0.15)", maxHeight: 280, overflowY: "auto",
            fontSize: "0.8125rem",
          }}
        >
          {/* Search hint */}
          {hint && (
            <div style={{ padding: "0.5rem 1rem", color: "var(--muted)", fontStyle: "italic", borderBottom: "1px solid var(--border)" }}>
              {hint}
            </div>
          )}

          {/* Loading first page */}
          {loading && (
            <div style={{ padding: "0.75rem 1rem", color: "var(--muted)" }}>Loading…</div>
          )}

          {/* No results */}
          {!loading && items.length === 0 && !hint && (
            <div style={{ padding: "0.75rem 1rem", color: "var(--muted)", fontStyle: "italic" }}>
              {query.length >= SEARCH_THRESHOLD ? `No categories match "${query}"` : "No categories"}
            </div>
          )}

          {/* Grouped list */}
          {!loading && roots.map((root) => {
            const children = childrenOf(root.id);
            return (
              <div key={root.id}>
                <Item cat={root} selected={value === root.id} onSelect={select} bold />
                {children.map((child) => (
                  <Item key={child.id} cat={child} selected={value === child.id} onSelect={select} indent />
                ))}
              </div>
            );
          })}

          {/* Orphans (parent not in current page) */}
          {!loading && orphans.map((c) => (
            <Item
              key={c.id}
              cat={c}
              selected={value === c.id}
              onSelect={select}
              prefix={`${parentMap.get(c.parent_id ?? "") ?? "…"} /`}
            />
          ))}

          {/* Load-more spinner */}
          {loadingMore && (
            <div style={{ padding: "0.5rem 1rem", color: "var(--muted)", textAlign: "center", fontSize: "0.75rem" }}>
              Loading more…
            </div>
          )}

          {/* Total count footer */}
          {!loading && items.length > 0 && (
            <div style={{ padding: "0.375rem 1rem", borderTop: "1px solid var(--border)", color: "var(--muted)", fontSize: "0.7rem" }}>
              {items.length} of {total} categor{total === 1 ? "y" : "ies"}
              {query.length >= SEARCH_THRESHOLD ? ` matching "${query}"` : ""}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ── Row component ─────────────────────────────────────────────────────────────

function Item({ cat, selected, onSelect, bold, indent, prefix }: {
  cat: Category;
  selected: boolean;
  onSelect: (c: Category) => void;
  bold?: boolean;
  indent?: boolean;
  prefix?: string;
}) {
  return (
    <div
      onMouseDown={(e) => { e.preventDefault(); onSelect(cat); }}
      style={{
        padding: indent ? "0.375rem 1rem 0.375rem 1.75rem" : "0.5rem 1rem",
        cursor: "pointer",
        fontWeight: bold ? 600 : 400,
        color: selected ? "var(--accent)" : indent ? "var(--text-2)" : "var(--text)",
        background: selected ? "var(--accent-bg)" : "transparent",
        borderLeft: indent ? "2px solid var(--border)" : "none",
        marginLeft: indent ? "1rem" : 0,
        transition: "background 0.08s",
      }}
      onMouseEnter={(e) => { if (!selected) (e.currentTarget as HTMLElement).style.background = "var(--surface2)"; }}
      onMouseLeave={(e) => { if (!selected) (e.currentTarget as HTMLElement).style.background = "transparent"; }}
    >
      {prefix && <span style={{ color: "var(--muted)", fontSize: "0.7rem", marginRight: "0.25rem" }}>{prefix}</span>}
      {cat.name}
    </div>
  );
}
