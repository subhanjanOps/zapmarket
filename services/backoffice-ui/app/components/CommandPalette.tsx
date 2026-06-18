"use client";

import { useState, useEffect, useRef } from "react";
import { useRouter } from "next/navigation";
import {
  Search,
  LayoutDashboard,
  Tag,
  Package,
  Layers,
  Users,
  Store,
  ClipboardList,
  ShieldCheck,
  Plus,
  CornerDownLeft,
  type LucideIcon,
} from "lucide-react";

interface Cmd {
  id: string;
  group: "Navigate" | "Create";
  label: string;
  Icon: LucideIcon;
  href: string;
  shortcut?: string[];
}

const ALL: Cmd[] = [
  { id: "go-dashboard",  group: "Navigate", label: "Dashboard",    Icon: LayoutDashboard, href: "/dashboard" },
  { id: "go-categories", group: "Navigate", label: "Categories",   Icon: Tag,             href: "/dashboard/categories" },
  { id: "go-products",   group: "Navigate", label: "Products",     Icon: Package,         href: "/dashboard/products" },
  { id: "go-skus",       group: "Navigate", label: "SKUs",         Icon: Layers,          href: "/dashboard/skus" },
  { id: "go-users",      group: "Navigate", label: "Users",        Icon: Users,           href: "/dashboard/users" },
  { id: "go-sellers",    group: "Navigate", label: "Sellers",      Icon: Store,           href: "/dashboard/sellers" },
  { id: "go-orders",     group: "Navigate", label: "Orders",       Icon: ClipboardList,   href: "/dashboard/orders" },
  { id: "go-moderation", group: "Navigate", label: "Moderation",   Icon: ShieldCheck,     href: "/dashboard/moderation" },
  { id: "new-category",  group: "Create",   label: "New category", Icon: Plus, href: "/dashboard/categories", shortcut: ["N", "C"] },
  { id: "new-product",   group: "Create",   label: "New product",  Icon: Plus, href: "/dashboard/products",   shortcut: ["N", "P"] },
];

export function CommandPalette() {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [activeIdx, setActiveIdx] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const itemRefs = useRef<(HTMLButtonElement | null)[]>([]);

  // ⌘K / Ctrl+K to toggle
  useEffect(() => {
    function onGlobalKey(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        setOpen((prev) => {
          if (!prev) { setQuery(""); setActiveIdx(0); }
          return !prev;
        });
      }
    }
    window.addEventListener("keydown", onGlobalKey);
    return () => window.removeEventListener("keydown", onGlobalKey);
  }, []);

  // Auto-focus input on open
  useEffect(() => {
    if (open) {
      const t = setTimeout(() => inputRef.current?.focus(), 10);
      return () => clearTimeout(t);
    }
  }, [open]);

  // Reset active index when query changes
  useEffect(() => { setActiveIdx(0); }, [query]);

  const filtered = query.trim()
    ? ALL.filter((c) => c.label.toLowerCase().includes(query.toLowerCase().trim()))
    : ALL;

  const groups: { label: string; items: Cmd[] }[] = query.trim()
    ? [{ label: "Results", items: filtered }]
    : [
        { label: "Navigate", items: filtered.filter((c) => c.group === "Navigate") },
        { label: "Create",   items: filtered.filter((c) => c.group === "Create") },
      ].filter((g) => g.items.length > 0);

  // Flat ordered list for index tracking
  const flat = groups.flatMap((g) => g.items);

  function close() { setOpen(false); setQuery(""); }

  function execute(cmd: Cmd) {
    router.push(cmd.href);
    close();
  }

  function onInputKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    switch (e.key) {
      case "ArrowDown": {
        e.preventDefault();
        const next = Math.min(activeIdx + 1, flat.length - 1);
        setActiveIdx(next);
        itemRefs.current[next]?.scrollIntoView({ block: "nearest" });
        break;
      }
      case "ArrowUp": {
        e.preventDefault();
        const prev = Math.max(activeIdx - 1, 0);
        setActiveIdx(prev);
        itemRefs.current[prev]?.scrollIntoView({ block: "nearest" });
        break;
      }
      case "Enter": {
        const cmd = flat[activeIdx];
        if (cmd) execute(cmd);
        break;
      }
      case "Escape":
        close();
        break;
    }
  }

  if (!open) return null;

  return (
    <>
      <div className="cmdk-overlay" onClick={close} />
      <div
        className="cmdk"
        role="dialog"
        aria-modal="true"
        aria-label="Command palette"
      >
        {/* Search */}
        <div className="cmdk-input-wrap">
          <Search size={15} strokeWidth={1.75} className="cmdk-search-icon" />
          <input
            ref={inputRef}
            className="cmdk-input"
            placeholder="Go to, search, or create…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={onInputKeyDown}
            autoComplete="off"
            spellCheck={false}
          />
          {query && (
            <button
              onClick={() => setQuery("")}
              style={{
                background: "none",
                border: "none",
                color: "var(--muted)",
                cursor: "pointer",
                fontSize: "0.6875rem",
                padding: "0 0.25rem",
                fontFamily: "inherit",
              }}
            >
              Clear
            </button>
          )}
        </div>

        {/* Results */}
        <div className="cmdk-results">
          {flat.length === 0 ? (
            <div className="cmdk-empty">No results for &quot;{query}&quot;</div>
          ) : (
            groups.map((group) => (
              <div key={group.label}>
                {!query.trim() && (
                  <div className="cmdk-section-label">{group.label}</div>
                )}
                {group.items.map((cmd) => {
                  const flatIdx = flat.indexOf(cmd);
                  const isActive = flatIdx === activeIdx;
                  return (
                    <button
                      key={cmd.id}
                      ref={(el) => { itemRefs.current[flatIdx] = el; }}
                      className={`cmdk-item${isActive ? " cmdk-item-active" : ""}`}
                      onClick={() => execute(cmd)}
                      onMouseEnter={() => setActiveIdx(flatIdx)}
                    >
                      <cmd.Icon size={14} strokeWidth={1.75} className="cmdk-item-icon" />
                      <span className="cmdk-item-label">{cmd.label}</span>
                      {cmd.shortcut && (
                        <span className="cmdk-item-shortcut">
                          {cmd.shortcut.map((k) => (
                            <span key={k} className="kbd">{k}</span>
                          ))}
                        </span>
                      )}
                    </button>
                  );
                })}
              </div>
            ))
          )}
        </div>

        {/* Footer */}
        <div className="cmdk-footer">
          <span className="cmdk-footer-hint">
            <span className="kbd">↑</span><span className="kbd">↓</span> navigate
          </span>
          <span className="cmdk-footer-hint">
            <CornerDownLeft size={10} /> open
          </span>
          <span className="cmdk-footer-hint">
            <span className="kbd">esc</span> close
          </span>
        </div>
      </div>
    </>
  );
}
