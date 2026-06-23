"use client";
import Link from "next/link";
import { Search } from "lucide-react";
import CartButton from "./CartButton";

export default function Navbar() {
  return (
    <header className="bg-[#232F3E] text-white sticky top-0 z-50">
      <div className="max-w-7xl mx-auto px-4 py-3 flex items-center gap-4">
        <Link href="/" className="text-xl font-bold text-[#FF9900]">
          ZapMarket
        </Link>
        <div className="flex-1 flex items-center bg-white rounded overflow-hidden max-w-2xl">
          <input
            type="text"
            placeholder="Search products…"
            className="flex-1 px-3 py-2 text-gray-900 text-sm outline-none"
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                const q = (e.target as HTMLInputElement).value.trim();
                if (q) window.location.href = `/products?search=${encodeURIComponent(q)}`;
              }
            }}
          />
          <button className="bg-[#FF9900] px-4 py-2 text-gray-900 hover:bg-[#e68900]">
            <Search size={18} />
          </button>
        </div>
        <nav className="flex items-center gap-4 text-sm">
          <Link href="/account/orders" className="hover:text-[#FF9900]">Orders</Link>
          <Link href="/login" className="hover:text-[#FF9900]">Sign In</Link>
          <CartButton />
        </nav>
      </div>
    </header>
  );
}
