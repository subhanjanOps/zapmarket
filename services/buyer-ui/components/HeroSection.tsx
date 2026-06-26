"use client";

import Link from "next/link";
import Image from "next/image";
import { motion } from "framer-motion";
import { Smartphone, Shirt, Home, Sparkles } from "lucide-react";

interface Campaign {
  id: string;
  title: string;
  subtitle?: string;
  image_url?: string;
  cta_text?: string;
  cta_url?: string;
  bg_color?: string;
}

const CATEGORY_TEASERS = [
  { icon: Smartphone, name: "Electronics", href: "/products?category=electronics" },
  { icon: Shirt,       name: "Fashion",     href: "/products?category=fashion" },
  { icon: Home,        name: "Home",        href: "/products?category=home" },
  { icon: Sparkles,    name: "Beauty",      href: "/products?category=beauty" },
];

const TRUST_ITEMS = [
  "Free delivery on orders above ₹499",
  "Easy 30-day returns",
  "Secure payments",
];

export function HeroSection({ campaigns }: { campaigns: Campaign[] }) {
  const primary = campaigns[0];

  return (
    <div className="bg-white">
      {/* Main hero */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-14 sm:py-20">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 lg:gap-16 items-center">

          {/* Left — text */}
          <motion.div
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.5, ease: [0.25, 0.1, 0.25, 1] }}
          >
            <h1
              className="font-bold text-[#111111] leading-tight"
              style={{ fontSize: "clamp(2rem, 4vw, 2.75rem)", letterSpacing: "-0.02em" }}
            >
              Discover<br />Amazing Products
            </h1>

            <p className="mt-4 text-lg text-[#555555] leading-relaxed max-w-md">
              Shop from thousands of verified sellers. Fast delivery, easy returns.
            </p>

            <div className="mt-8 flex flex-wrap items-center gap-3">
              <motion.div whileTap={{ scale: 0.97 }}>
                <Link
                  href="/products"
                  className="inline-flex items-center justify-center h-11 px-8 rounded-md bg-[#E91E8C] hover:bg-[#C2187A] text-white text-sm font-medium transition-colors"
                >
                  Shop Now
                </Link>
              </motion.div>

              <motion.div whileTap={{ scale: 0.97 }}>
                <Link
                  href="/deals"
                  className="inline-flex items-center justify-center h-11 px-8 rounded-md border border-[#E8E8E8] hover:bg-[#F6F6F6] text-[#111111] text-sm font-medium transition-colors"
                >
                  Explore Deals
                </Link>
              </motion.div>
            </div>

            <motion.p
              className="mt-6 text-xs text-[#999999]"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: 0.35 }}
            >
              2M+ Customers&nbsp;&nbsp;·&nbsp;&nbsp;50K+ Products&nbsp;&nbsp;·&nbsp;&nbsp;99% On-Time Delivery
            </motion.p>
          </motion.div>

          {/* Right — visual */}
          <motion.div
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.5, delay: 0.1, ease: [0.25, 0.1, 0.25, 1] }}
          >
            {primary?.image_url ? (
              /* Campaign with image */
              <div className="relative w-full aspect-[4/3] rounded-lg overflow-hidden border border-[#E8E8E8] shadow-sm">
                <Image
                  src={primary.image_url}
                  alt={primary.title}
                  fill
                  className="object-cover"
                  priority
                />
              </div>
            ) : primary ? (
              /* Campaign card — no image */
              <div
                className="rounded-lg border border-[#E8E8E8] p-8 shadow-sm"
                style={{ backgroundColor: primary.bg_color ?? "#F6F6F6" }}
              >
                <h2
                  className="font-semibold text-[#111111] leading-tight"
                  style={{ fontSize: "1.5rem", letterSpacing: "-0.02em" }}
                >
                  {primary.title}
                </h2>
                {primary.subtitle && (
                  <p className="mt-2 text-sm text-[#555555]">{primary.subtitle}</p>
                )}
                {primary.cta_url && (
                  <motion.div className="mt-6" whileTap={{ scale: 0.97 }}>
                    <Link
                      href={primary.cta_url}
                      className="inline-flex items-center justify-center h-10 px-6 rounded-md bg-[#111111] hover:bg-[#222222] text-white text-sm font-medium transition-colors"
                    >
                      {primary.cta_text ?? "Shop Now"}
                    </Link>
                  </motion.div>
                )}
              </div>
            ) : (
              /* Fallback — 2×2 category grid */
              <div className="grid grid-cols-2 gap-4">
                {CATEGORY_TEASERS.map(({ icon: Icon, name, href }) => (
                  <motion.div
                    key={name}
                    whileHover={{ y: -4 }}
                    transition={{ duration: 0.2 }}
                  >
                    <Link
                      href={href}
                      className="flex flex-col items-start gap-3 p-4 rounded-lg bg-[#F6F6F6] border border-[#E8E8E8] shadow-sm hover:shadow-md hover:border-[#D0D0D0] transition-all"
                    >
                      <div className="flex items-center justify-center h-9 w-9 rounded-md bg-white border border-[#E8E8E8]">
                        <Icon className="h-4 w-4 text-[#111111]" strokeWidth={1.75} />
                      </div>
                      <span className="text-sm font-semibold text-[#111111]">{name}</span>
                    </Link>
                  </motion.div>
                ))}
              </div>
            )}
          </motion.div>

        </div>
      </div>

      {/* Trust strip */}
      <div className="bg-[#F6F6F6] py-3 border-t border-[#E8E8E8]">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex items-center justify-center flex-wrap gap-x-6 gap-y-1">
          {TRUST_ITEMS.map((item, i) => (
            <span key={item} className="flex items-center gap-3 text-xs text-[#555555]">
              {i > 0 && <span className="text-[#D0D0D0] select-none hidden sm:inline">|</span>}
              {item}
            </span>
          ))}
        </div>
      </div>
    </div>
  );
}
