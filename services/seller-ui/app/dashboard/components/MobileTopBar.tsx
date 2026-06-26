"use client";
import { Menu } from "lucide-react";

interface Props {
  onMenuOpen: () => void;
}

export function MobileTopBar({ onMenuOpen }: Props) {
  return (
    <header
      className="dash-topbar"
      style={{
        display: "none", alignItems: "center", justifyContent: "space-between",
        padding: "0.75rem 1rem",
        background: "var(--surface)",
        borderBottom: "1px solid var(--border)",
        position: "sticky", top: 0, zIndex: 20,
      }}
    >
      <button
        onClick={onMenuOpen}
        aria-label="Open menu"
        style={{ background: "none", border: "none", cursor: "pointer", color: "var(--text)", padding: 4, display: "flex", alignItems: "center" }}
      >
        <Menu size={22} />
      </button>
      <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
        <div style={{ width: 28, height: 28, borderRadius: 7, background: "var(--accent)", display: "flex", alignItems: "center", justifyContent: "center", fontSize: 11, fontWeight: 800, color: "var(--accent-text)", letterSpacing: "-0.04em" }}>ZM</div>
        <span style={{ fontWeight: 700, fontSize: "0.9375rem", color: "var(--text)", letterSpacing: "-0.02em", fontFamily: '"Rubik", "Outfit", system-ui, sans-serif' }}>ZapMarket</span>
      </div>
      <div style={{ width: 30 }} />
    </header>
  );
}
