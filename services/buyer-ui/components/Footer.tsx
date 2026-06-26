import Link from "next/link";
import { Zap, Instagram, Twitter, Facebook, Youtube } from "lucide-react";

const LINKS: Record<string, { label: string; href: string }[]> = {
  Company: [
    { label: "About us",      href: "/about" },
    { label: "Careers",       href: "/careers" },
    { label: "Press",         href: "/press" },
    { label: "Blog",          href: "/blog" },
  ],
  Help: [
    { label: "FAQs",          href: "/faq" },
    { label: "Order tracking", href: "/account/orders" },
    { label: "Returns",       href: "/returns" },
    { label: "Contact us",    href: "/contact" },
  ],
  Sellers: [
    { label: "Sell on ZapMarket",  href: "/sell" },
    { label: "Seller portal",      href: "/seller" },
    { label: "Seller guidelines",  href: "/seller/guidelines" },
    { label: "Success stories",    href: "/seller/stories" },
  ],
  Policies: [
    { label: "Privacy policy",   href: "/privacy" },
    { label: "Terms of service", href: "/terms" },
    { label: "Cookie policy",    href: "/cookies" },
    { label: "Refund policy",    href: "/refunds" },
  ],
};

const SOCIALS = [
  { icon: Instagram, href: "#", label: "Instagram" },
  { icon: Twitter,   href: "#", label: "Twitter / X" },
  { icon: Facebook,  href: "#", label: "Facebook" },
  { icon: Youtube,   href: "#", label: "YouTube" },
];

export default function Footer() {
  return (
    <footer className="bg-[#111111] text-[#AAAAAA]">

      {/* Top section — logo + newsletter */}
      <div className="border-b border-white/10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-10 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-8">

          {/* Logo + tagline + socials */}
          <div className="flex flex-col gap-4">
            <Link href="/" className="flex items-center gap-1.5">
              <Zap className="h-5 w-5 text-white" strokeWidth={2.5} />
              <span className="text-white font-bold text-lg tracking-tight">
                Zap<span className="text-[#E91E8C]">Market</span>
              </span>
            </Link>
            <p className="text-sm text-[#AAAAAA] leading-relaxed max-w-xs">
              India&apos;s fastest-growing marketplace for verified products and trusted sellers.
            </p>
            <div className="flex items-center gap-2">
              {SOCIALS.map(({ icon: Icon, href, label }) => (
                <a
                  key={label}
                  href={href}
                  aria-label={label}
                  className="h-8 w-8 bg-white/10 hover:bg-white/20 rounded-md flex items-center justify-center transition-colors duration-200"
                >
                  <Icon className="h-4 w-4 text-[#AAAAAA]" />
                </a>
              ))}
            </div>
          </div>

          {/* Newsletter */}
          <div className="flex flex-col gap-2 w-full sm:w-auto">
            <p className="text-xs font-semibold text-white uppercase tracking-wider">
              Stay in the loop
            </p>
            <p className="text-sm text-[#AAAAAA]">
              Deals, new arrivals and insider picks — no spam.
            </p>
            <div className="flex gap-2 mt-1">
              <input
                type="email"
                placeholder="your@email.com"
                className="flex-1 sm:w-64 h-9 px-3 bg-white/10 border border-white/15 rounded-md text-sm text-white placeholder-white/40 outline-none focus:border-white/30 transition-colors duration-200"
              />
              <button
                type="button"
                className="h-9 px-5 bg-[#E91E8C] hover:bg-[#C2187A] text-white text-sm font-medium rounded-md transition-colors duration-200 shrink-0"
              >
                Subscribe
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Main links grid */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-8">
          {Object.entries(LINKS).map(([group, items]) => (
            <div key={group}>
              <h4 className="text-xs font-semibold text-white uppercase tracking-wider mb-3">
                {group}
              </h4>
              <ul className="space-y-2.5">
                {items.map(({ label, href }) => (
                  <li key={label}>
                    <Link
                      href={href}
                      className="text-sm text-[#AAAAAA] hover:text-white transition-colors duration-200"
                    >
                      {label}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </div>

      {/* Bottom bar */}
      <div className="border-t border-white/10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-5 flex flex-col sm:flex-row items-center justify-between gap-3">
          <p className="text-xs text-[#666666]">
            &copy; 2024 ZapMarket Pvt. Ltd. All rights reserved.
          </p>
          <p className="text-xs text-[#666666]">
            Visa &middot; Mastercard &middot; UPI &middot; Net Banking
          </p>
        </div>
      </div>

    </footer>
  );
}
