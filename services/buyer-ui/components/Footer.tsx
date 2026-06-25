import Link from "next/link";
import { Zap, Instagram, Twitter, Facebook, Youtube, Mail } from "lucide-react";

const LINKS: Record<string, { label: string; href: string }[]> = {
  Company: [
    { label: "About us",       href: "/about" },
    { label: "Careers",        href: "/careers" },
    { label: "Press",          href: "/press" },
    { label: "Blog",           href: "/blog" },
  ],
  Help: [
    { label: "FAQs",           href: "/faq" },
    { label: "Order tracking", href: "/account/orders" },
    { label: "Returns",        href: "/returns" },
    { label: "Contact us",     href: "/contact" },
  ],
  Sellers: [
    { label: "Sell on ZapMarket",  href: "/sell" },
    { label: "Seller portal",      href: "/seller" },
    { label: "Seller guidelines",  href: "/seller/guidelines" },
    { label: "Success stories",    href: "/seller/stories" },
  ],
  Policies: [
    { label: "Privacy policy",    href: "/privacy" },
    { label: "Terms of service",  href: "/terms" },
    { label: "Cookie policy",     href: "/cookies" },
    { label: "Refund policy",     href: "/refunds" },
  ],
};

const SOCIALS = [
  { icon: Instagram, href: "#", label: "Instagram" },
  { icon: Twitter,   href: "#", label: "Twitter / X" },
  { icon: Facebook,  href: "#", label: "Facebook" },
  { icon: Youtube,   href: "#", label: "YouTube" },
];

const PAYMENT_METHODS = ["Visa", "Mastercard", "UPI", "NetBanking", "EMI", "COD"];

export default function Footer() {
  return (
    <footer className="bg-[#0F0A04] text-white">
      {/* Newsletter */}
      <div className="border-b border-white/10">
        <div className="container-zap py-9 flex flex-col sm:flex-row items-center gap-6 justify-between">
          <div>
            <h3 className="font-display font-bold text-lg">Stay in the loop</h3>
            <p className="text-white/50 text-sm mt-1">
              Deals, new arrivals & insider picks — no spam.
            </p>
          </div>
          <form
            className="flex gap-2 w-full sm:w-auto"
            action="#"
          >
            <div className="relative flex-1 sm:w-72">
              <Mail className="absolute left-3.5 top-1/2 -translate-y-1/2 h-4 w-4 text-white/30" />
              <input
                type="email"
                placeholder="your@email.com"
                className="w-full h-11 pl-10 pr-4 bg-white/10 border border-white/15 rounded-xl text-sm text-white placeholder-white/30 outline-none focus:border-[#E91E8C] transition-colors"
              />
            </div>
            <button
              type="submit"
              className="h-11 px-5 bg-[#E91E8C] hover:bg-[#FF5A35] text-white text-sm font-bold rounded-xl transition-colors shrink-0"
            >
              Subscribe
            </button>
          </form>
        </div>
      </div>

      {/* Main links */}
      <div className="container-zap py-12">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-10">
          {/* Brand */}
          <div className="lg:col-span-1">
            <Link href="/" className="flex items-center gap-2 mb-4">
              <div className="h-8 w-8 bg-[#E91E8C] rounded-xl flex items-center justify-center">
                <Zap className="h-4 w-4 text-white" strokeWidth={2.5} />
              </div>
              <span className="font-display font-bold text-lg">ZapMarket</span>
            </Link>
            <p className="text-white/45 text-sm leading-relaxed mb-5">
              India&apos;s fastest-growing marketplace for verified products and trusted sellers.
            </p>
            <div className="flex items-center gap-2">
              {SOCIALS.map(({ icon: Icon, href, label }) => (
                <a
                  key={label}
                  href={href}
                  aria-label={label}
                  className="h-8 w-8 rounded-lg bg-white/10 hover:bg-[#E91E8C] flex items-center justify-center transition-colors"
                >
                  <Icon className="h-3.5 w-3.5" />
                </a>
              ))}
            </div>
          </div>

          {/* Link columns */}
          {Object.entries(LINKS).map(([group, items]) => (
            <div key={group}>
              <h4 className="font-bold text-sm tracking-wide mb-4">{group}</h4>
              <ul className="space-y-2.5">
                {items.map(({ label, href }) => (
                  <li key={label}>
                    <Link
                      href={href}
                      className="text-sm text-white/45 hover:text-white transition-colors"
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
        <div className="container-zap py-5 flex flex-col sm:flex-row items-center justify-between gap-3">
          <p className="text-white/35 text-xs">
            © {new Date().getFullYear()} ZapMarket Pvt. Ltd. · Made with ♥ in India
          </p>
          <div className="flex items-center gap-2 flex-wrap justify-center">
            {PAYMENT_METHODS.map(method => (
              <span
                key={method}
                className="px-2.5 py-1 bg-white/8 text-white/45 text-xs rounded-md font-medium"
              >
                {method}
              </span>
            ))}
          </div>
        </div>
      </div>
    </footer>
  );
}
