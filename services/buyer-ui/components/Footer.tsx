import Link from "next/link";
import { Zap } from "lucide-react";
import NewsletterForm from "./NewsletterForm";

export default function Footer() {
  return (
    <footer style={{ background: "#1A1208", color: "#F0EDE8" }} className="mt-20">
      {/* Newsletter strip */}
      <div
        className="py-10 px-4"
        style={{ borderBottom: "1px solid rgba(255,255,255,0.06)", background: "#0F0B05" }}
      >
        <div className="max-w-7xl mx-auto flex flex-col sm:flex-row items-center justify-between gap-5">
          <div>
            <p className="text-sm font-bold" style={{ color: "#FF2D78", fontFamily: "var(--font-syne)" }}>
              Stay in the loop
            </p>
            <p className="text-xs mt-0.5" style={{ color: "#6B6052" }}>
              New deals and products, straight to your inbox.
            </p>
          </div>
          <NewsletterForm />
        </div>
      </div>

      {/* Main footer links */}
      <div className="max-w-7xl mx-auto px-4 py-12 grid grid-cols-2 md:grid-cols-4 gap-8 text-sm">
        <div>
          <h3 className="font-extrabold text-sm mb-4 uppercase tracking-wider"
            style={{ fontFamily: "var(--font-syne)", color: "#FF2D78" }}>
            Shop
          </h3>
          <ul className="space-y-2.5" style={{ color: "#6B6052" }}>
            <li><Link href="/products" className="hover:text-white transition-colors duration-150">All products</Link></li>
            <li><Link href="/deals/summer-sale" className="hover:text-white transition-colors duration-150">Today&apos;s deals</Link></li>
          </ul>
        </div>
        <div>
          <h3 className="font-extrabold text-sm mb-4 uppercase tracking-wider"
            style={{ fontFamily: "var(--font-syne)", color: "#FF2D78" }}>
            Learn
          </h3>
          <ul className="space-y-2.5" style={{ color: "#6B6052" }}>
            <li><Link href="/blog" className="hover:text-white transition-colors duration-150">Blog</Link></li>
            <li><Link href="/glossary" className="hover:text-white transition-colors duration-150">Glossary</Link></li>
          </ul>
        </div>
        <div>
          <h3 className="font-extrabold text-sm mb-4 uppercase tracking-wider"
            style={{ fontFamily: "var(--font-syne)", color: "#FF2D78" }}>
            Account
          </h3>
          <ul className="space-y-2.5" style={{ color: "#6B6052" }}>
            <li><Link href="/login" className="hover:text-white transition-colors duration-150">Sign in</Link></li>
            <li><Link href="/register" className="hover:text-white transition-colors duration-150">Register</Link></li>
            <li><Link href="/account/orders" className="hover:text-white transition-colors duration-150">My orders</Link></li>
          </ul>
        </div>
        <div>
          <div className="flex items-center gap-1.5 mb-3">
            <Zap size={16} fill="#FF2D78" stroke="#FF2D78" />
            <h3 className="font-extrabold text-sm uppercase tracking-wider"
              style={{ fontFamily: "var(--font-syne)", color: "#FF2D78" }}>
              ZapMarket
            </h3>
          </div>
          <p className="text-xs leading-relaxed" style={{ color: "#4A4030" }}>
            The fastest marketplace in India. Real sellers, real products, zapped to your door.
          </p>
        </div>
      </div>

      {/* Bottom bar */}
      <div
        className="text-center py-4 text-xs"
        style={{ borderTop: "1px solid #2A2012", color: "#4A4030" }}
      >
        © {new Date().getFullYear()} ZapMarket. All rights reserved.
      </div>
    </footer>
  );
}
