import Link from "next/link";

export default function Footer() {
  return (
    <footer className="bg-[#232F3E] text-gray-300 mt-16">
      <div className="max-w-7xl mx-auto px-4 py-10 grid grid-cols-2 md:grid-cols-4 gap-8 text-sm">
        <div>
          <h3 className="text-white font-semibold mb-3">Shop</h3>
          <ul className="space-y-2">
            <li><Link href="/products" className="hover:text-white">All Products</Link></li>
            <li><Link href="/deals/summer-sale" className="hover:text-white">Today&apos;s Deals</Link></li>
          </ul>
        </div>
        <div>
          <h3 className="text-white font-semibold mb-3">Learn</h3>
          <ul className="space-y-2">
            <li><Link href="/blog" className="hover:text-white">Blog</Link></li>
            <li><Link href="/glossary" className="hover:text-white">Glossary</Link></li>
          </ul>
        </div>
        <div>
          <h3 className="text-white font-semibold mb-3">Account</h3>
          <ul className="space-y-2">
            <li><Link href="/login" className="hover:text-white">Sign In</Link></li>
            <li><Link href="/register" className="hover:text-white">Register</Link></li>
            <li><Link href="/account/orders" className="hover:text-white">My Orders</Link></li>
          </ul>
        </div>
        <div>
          <h3 className="text-white font-semibold mb-3">ZapMarket</h3>
          <p className="text-xs text-gray-400">The fastest marketplace in India.</p>
        </div>
      </div>
      <div className="border-t border-gray-700 text-center py-4 text-xs text-gray-500">
        © {new Date().getFullYear()} ZapMarket. All rights reserved.
      </div>
    </footer>
  );
}
