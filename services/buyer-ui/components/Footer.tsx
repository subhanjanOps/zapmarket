import Link from "next/link";

export default function Footer() {
  return (
    <footer style={{ background: "#1A1208", color: "#F0EDE8" }} className="mt-16">
      <div className="max-w-7xl mx-auto px-4 py-12 grid grid-cols-2 md:grid-cols-4 gap-8 text-sm">
        <div>
          <h3
            className="font-extrabold text-base mb-4"
            style={{ fontFamily: "var(--font-syne)", color: "#FF2D78" }}
          >
            Shop
          </h3>
          <ul className="space-y-2" style={{ color: "#9CA3AF" }}>
            <li><Link href="/products" className="hover:text-white transition-colors">All products</Link></li>
            <li><Link href="/deals/summer-sale" className="hover:text-white transition-colors">Today&apos;s deals</Link></li>
          </ul>
        </div>
        <div>
          <h3
            className="font-extrabold text-base mb-4"
            style={{ fontFamily: "var(--font-syne)", color: "#FF2D78" }}
          >
            Learn
          </h3>
          <ul className="space-y-2" style={{ color: "#9CA3AF" }}>
            <li><Link href="/blog" className="hover:text-white transition-colors">Blog</Link></li>
            <li><Link href="/glossary" className="hover:text-white transition-colors">Glossary</Link></li>
          </ul>
        </div>
        <div>
          <h3
            className="font-extrabold text-base mb-4"
            style={{ fontFamily: "var(--font-syne)", color: "#FF2D78" }}
          >
            Account
          </h3>
          <ul className="space-y-2" style={{ color: "#9CA3AF" }}>
            <li><Link href="/login" className="hover:text-white transition-colors">Sign in</Link></li>
            <li><Link href="/register" className="hover:text-white transition-colors">Register</Link></li>
            <li><Link href="/account/orders" className="hover:text-white transition-colors">My orders</Link></li>
          </ul>
        </div>
        <div>
          <h3
            className="font-extrabold text-base mb-4"
            style={{ fontFamily: "var(--font-syne)", color: "#FF2D78" }}
          >
            ⚡ ZapMarket
          </h3>
          <p className="text-xs" style={{ color: "#6B6052" }}>
            The fastest marketplace in India. Real sellers, real products, zapped to your door.
          </p>
        </div>
      </div>
      <div
        className="text-center py-4 text-xs"
        style={{ borderTop: "1px solid #2A2012", color: "#6B6052" }}
      >
        © {new Date().getFullYear()} ZapMarket. All rights reserved.
      </div>
    </footer>
  );
}
