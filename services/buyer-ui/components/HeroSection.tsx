"use client";

import Link from "next/link";
import Image from "next/image";
import { motion } from "framer-motion";
import { ArrowRight, Zap, ShieldCheck, RefreshCw, Headphones } from "lucide-react";

interface Campaign {
  id: string;
  title: string;
  subtitle?: string;
  image_url?: string;
  cta_text?: string;
  cta_url?: string;
  bg_color?: string;
}

const TRUST_PILLS = [
  { icon: ShieldCheck, label: "Secure payments" },
  { icon: RefreshCw,   label: "Easy returns" },
  { icon: Headphones,  label: "24/7 support" },
];

const STAT_ITEMS = [
  { value: "2M+",  label: "Happy customers" },
  { value: "50K+", label: "Products" },
  { value: "99%",  label: "Delivery success" },
];

const FALLBACK_SECONDARY: Campaign[] = [
  { id: "fs", title: "Flash Sale", subtitle: "Up to 60% off today", cta_text: "Shop deals", cta_url: "/deals" },
  { id: "na", title: "New Arrivals", subtitle: "Fresh picks just landed", cta_text: "Explore", cta_url: "/products?sort_by=created_at" },
];

export function HeroSection({ campaigns }: { campaigns: Campaign[] }) {
  const primary = campaigns[0];
  const secondaries = campaigns.slice(1, 3).length > 0 ? campaigns.slice(1, 3) : FALLBACK_SECONDARY;

  return (
    <section className="container-zap py-5 sm:py-6">
      <div className="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-4">
        {/* Primary hero */}
        <motion.div
          className="relative overflow-hidden rounded-3xl bg-gradient-to-br from-[#0F0A04] via-[#1C0F08] to-[#2D1015] min-h-[320px] sm:min-h-[400px] flex flex-col justify-end p-7 sm:p-10"
          initial={{ opacity: 0, scale: 0.97 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.7, ease: [0.16, 1, 0.3, 1] }}
        >
          {primary?.image_url && (
            <Image
              src={primary.image_url}
              alt={primary.title ?? "Campaign"}
              fill
              className="object-cover opacity-25 mix-blend-luminosity"
              priority
            />
          )}

          {/* Glow orbs */}
          <div className="absolute top-10 right-10 h-40 w-40 bg-[#E91E8C]/15 rounded-full blur-3xl pointer-events-none" />
          <div className="absolute top-6 right-24 h-20 w-20 bg-[#FF5A35]/15 rounded-full blur-2xl pointer-events-none" />
          <div className="absolute bottom-0 left-0 h-32 w-48 bg-[#E91E8C]/8 rounded-full blur-3xl pointer-events-none" />

          {/* Promo badge */}
          <div className="absolute top-5 left-5 sm:top-7 sm:left-7">
            <div className="flex items-center gap-1.5 px-3 py-1.5 bg-[#E91E8C] rounded-full shadow-[0_2px_12px_rgba(233,30,140,0.4)]">
              <Zap className="h-3 w-3 text-white" strokeWidth={2.5} />
              <span className="text-xs font-bold text-white tracking-wide">Limited time offer</span>
            </div>
          </div>

          {/* Content */}
          <div className="relative z-10">
            <motion.h1
              className="font-display font-extrabold text-white leading-[1.1] mb-3"
              style={{ fontSize: "clamp(1.75rem, 4vw, 2.75rem)" }}
              initial={{ opacity: 0, y: 24 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.2, duration: 0.6, ease: [0.16, 1, 0.3, 1] }}
            >
              {primary?.title ?? "Discover Amazing\nDeals Today"}
            </motion.h1>

            {primary?.subtitle && (
              <motion.p
                className="text-white/65 text-base mb-6 max-w-sm leading-relaxed"
                initial={{ opacity: 0, y: 16 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.32, duration: 0.55 }}
              >
                {primary.subtitle}
              </motion.p>
            )}

            <motion.div
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.42, duration: 0.5 }}
              className="flex flex-wrap items-center gap-3"
            >
              <Link
                href={primary?.cta_url ?? "/products"}
                className="inline-flex items-center gap-2 px-6 py-3 bg-[#E91E8C] hover:bg-[#FF5A35] text-white font-bold rounded-2xl transition-colors shadow-[0_4px_20px_rgba(233,30,140,0.4)] hover:shadow-[0_4px_20px_rgba(255,90,53,0.4)]"
              >
                {primary?.cta_text ?? "Shop now"}
                <ArrowRight className="h-4 w-4" />
              </Link>
              <Link
                href="/products"
                className="inline-flex items-center gap-2 px-5 py-3 bg-white/10 hover:bg-white/15 text-white font-semibold rounded-2xl transition-colors border border-white/20 text-sm"
              >
                Browse all
              </Link>
            </motion.div>
          </div>

          {/* Stats — desktop */}
          <motion.div
            className="absolute bottom-7 right-7 hidden sm:flex items-center gap-5"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: 0.65 }}
          >
            {STAT_ITEMS.map(({ value, label }) => (
              <div key={label} className="text-right">
                <div className="font-display font-bold text-white text-xl tabular-nums">{value}</div>
                <div className="text-white/40 text-xs">{label}</div>
              </div>
            ))}
          </motion.div>
        </motion.div>

        {/* Secondary campaign cards */}
        <div className="flex flex-row lg:flex-col gap-4">
          {secondaries.map((camp, i) => (
            <motion.div
              key={camp.id}
              className="relative flex-1 overflow-hidden rounded-2xl min-h-[120px] sm:min-h-[155px] flex flex-col justify-end p-5 group cursor-pointer"
              style={{
                background: i === 0
                  ? "linear-gradient(135deg, #FDE8F4 0%, #FFF0EC 100%)"
                  : "linear-gradient(135deg, #F0F9FF 0%, #EEF2FF 100%)",
              }}
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: 0.2 + i * 0.1, duration: 0.6, ease: [0.16, 1, 0.3, 1] }}
              whileHover={{ scale: 1.02 }}
            >
              {camp.image_url && (
                <Image
                  src={camp.image_url}
                  alt={camp.title}
                  fill
                  className="object-cover opacity-15 group-hover:opacity-25 transition-opacity"
                />
              )}
              <div className="relative z-10">
                <h3 className="font-display font-bold text-[#0F0A04] text-lg leading-tight">
                  {camp.title}
                </h3>
                {camp.subtitle && (
                  <p className="text-sm text-[#7A6856] mt-0.5">{camp.subtitle}</p>
                )}
                <Link
                  href={camp.cta_url ?? "/products"}
                  className="mt-3 inline-flex items-center gap-1 text-sm font-bold text-[#E91E8C] hover:text-[#B5166E] transition-colors"
                >
                  {camp.cta_text ?? "Shop now"}
                  <ArrowRight className="h-3.5 w-3.5 transition-transform group-hover:translate-x-0.5" />
                </Link>
              </div>
            </motion.div>
          ))}
        </div>
      </div>

      {/* Trust pills */}
      <div className="flex items-center justify-center gap-6 sm:gap-10 mt-5 py-1">
        {TRUST_PILLS.map(({ icon: Icon, label }) => (
          <div key={label} className="flex items-center gap-2 text-sm text-[#7A6856]">
            <Icon className="h-4 w-4 text-[#E91E8C]" />
            <span className="font-medium hidden sm:block">{label}</span>
          </div>
        ))}
      </div>
    </section>
  );
}
