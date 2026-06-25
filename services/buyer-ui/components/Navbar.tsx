"use client";
import { useState, useEffect } from "react";
import Link from "next/link";
import { Search, Zap, User, Package } from "lucide-react";
import CartButton from "./CartButton";

export default function Navbar() {
  const [scrolled, setScrolled] = useState(false);
  const [query, setQuery] = useState("");

  useEffect(() => {
    const handler = () => setScrolled(window.scrollY > 12);
    window.addEventListener("scroll", handler, { passive: true });
    return () => window.removeEventListener("scroll", handler);
  }, []);

  function handleSearch(e: React.FormEvent) {
    e.preventDefault();
    const q = query.trim();
    if (q) window.location.href = `/products?search=${encodeURIComponent(q)}`;
  }

  return (
    <header
      className="sticky top-0 z-50 transition-all duration-300"
      style={{
        background: scrolled ? "rgba(255,252,245,0.88)" : "#FFFCF5",
        backdropFilter: scrolled ? "blur(12px)" : "none",
        WebkitBackdropFilter: scrolled ? "blur(12px)" : "none",
        borderBottom: scrolled ? "1px solid rgba(240,237,232,0.8)" : "3px solid #FF2D78",
        boxShadow: scrolled ? "0 2px 20px rgba(26,18,8,0.06)" : "none",
      }}
    >
      <div className="max-w-7xl mx-auto px-4 py-3 flex items-center gap-4">
        {/* Logo */}
        <Link
          href="/"
          className="shrink-0 flex items-center gap-1 text-2xl font-extrabold tracking-tight leading-none group"
          style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}
        >
          <Zap
            size={22}
            fill="#FF2D78"
            stroke="#FF2D78"
            className="transition-transform duration-200 group-hover:scale-110 group-hover:rotate-12"
          />
          <span style={{ color: "#FF2D78" }}>Zap</span>Market
        </Link>

        {/* Search */}
        <form
          onSubmit={handleSearch}
          className="flex-1 flex items-center rounded-full overflow-hidden max-w-2xl transition-all duration-200"
          style={{
            border: "2px solid #F0EDE8",
            background: "#fff",
            boxShadow: "0 1px 4px rgba(26,18,8,0.04)",
          }}
          onFocus={(e) => { (e.currentTarget as HTMLElement).style.borderColor = "#FF2D78"; (e.currentTarget as HTMLElement).style.boxShadow = "0 0 0 3px rgba(255,45,120,0.08)"; }}
          onBlur={(e) => { (e.currentTarget as HTMLElement).style.borderColor = "#F0EDE8"; (e.currentTarget as HTMLElement).style.boxShadow = "0 1px 4px rgba(26,18,8,0.04)"; }}
        >
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Find anything…"
            className="flex-1 px-4 py-2.5 text-sm outline-none bg-transparent"
            style={{ color: "#1A1208" }}
          />
          <button
            type="submit"
            className="mr-1.5 p-2 rounded-full text-white transition-all duration-200 hover:scale-105 active:scale-95 cursor-pointer"
            style={{ background: "#00736A" }}
            aria-label="Search"
          >
            <Search size={15} />
          </button>
        </form>

        {/* Nav links */}
        <nav className="flex items-center gap-5 text-sm font-medium shrink-0">
          <Link
            href="/account/orders"
            className="hidden sm:flex items-center gap-1.5 transition-all duration-150 hover:text-[#FF2D78] hover:-translate-y-px"
            style={{ color: "#1A1208" }}
          >
            <Package size={16} />
            <span className="hidden md:inline">Orders</span>
          </Link>
          <Link
            href="/login"
            className="hidden sm:flex items-center gap-1.5 transition-all duration-150 hover:text-[#FF2D78] hover:-translate-y-px"
            style={{ color: "#1A1208" }}
          >
            <User size={16} />
            <span className="hidden md:inline">Sign in</span>
          </Link>
          <CartButton />
        </nav>
      </div>
    </header>
  );
}
