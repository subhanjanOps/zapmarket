"use client";
import Link from "next/link";
import { Search, Zap } from "lucide-react";
import CartButton from "./CartButton";

export default function Navbar() {
  return (
    <header
      className="bg-white sticky top-0 z-50"
      style={{ borderBottom: "3px solid #FF2D78" }}
    >
      <div className="max-w-7xl mx-auto px-4 py-3 flex items-center gap-4">
        <Link
          href="/"
          className="shrink-0 text-2xl font-extrabold tracking-tight leading-none"
          style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}
        >
          <Zap size={20} fill="#FF2D78" stroke="#FF2D78" className="inline-block mr-0.5" /><span style={{ color: "#FF2D78" }}>Zap</span>Market
        </Link>

        <div
          className="flex-1 flex items-center rounded-full overflow-hidden max-w-2xl"
          style={{ border: "2px solid #F0EDE8", background: "#FFFCF5" }}
        >
          <input
            type="text"
            placeholder="Find anything…"
            className="flex-1 px-4 py-2 text-sm outline-none bg-transparent placeholder-gray-400"
            style={{ color: "#1A1208" }}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                const q = (e.target as HTMLInputElement).value.trim();
                if (q) window.location.href = `/products?search=${encodeURIComponent(q)}`;
              }
            }}
          />
          <button
            className="mr-1 p-2 rounded-full text-white transition-opacity hover:opacity-80"
            style={{ background: "#00736A" }}
            aria-label="Search"
          >
            <Search size={15} />
          </button>
        </div>

        <nav className="flex items-center gap-5 text-sm font-medium shrink-0">
          <Link
            href="/account/orders"
            className="transition-colors hover:text-[#FF2D78]"
            style={{ color: "#1A1208" }}
          >
            Orders
          </Link>
          <Link
            href="/login"
            className="transition-colors hover:text-[#FF2D78]"
            style={{ color: "#1A1208" }}
          >
            Sign in
          </Link>
          <CartButton />
        </nav>
      </div>
    </header>
  );
}
