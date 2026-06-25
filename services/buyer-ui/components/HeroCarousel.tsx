"use client";
import { useState, useEffect } from "react";
import Link from "next/link";
import { Truck, ShieldCheck, Zap as ZapIcon } from "lucide-react";

interface Campaign {
  id: string;
  slug: string;
  title: string;
  headline: string;
  banner_image_url: string | null;
}

const TRUST_BADGES = [
  { icon: Truck, label: "Fast delivery" },
  { icon: ShieldCheck, label: "Secure payments" },
  { icon: ZapIcon, label: "Real sellers" },
];

function TrustStrip() {
  return (
    <div
      className="py-2.5 flex items-center justify-center gap-8 text-xs font-semibold"
      style={{ background: "#1A1208", color: "#F0EDE8" }}
    >
      {TRUST_BADGES.map(({ icon: Icon, label }) => (
        <span key={label} className="flex items-center gap-1.5">
          <Icon size={13} stroke="#FF8C00" />
          {label}
        </span>
      ))}
    </div>
  );
}

function SlideContent({ campaign, key: _key }: { campaign?: Campaign; key: string }) {
  const title = campaign?.title ?? "Everything you need,";
  const subtitle = campaign?.headline ?? "Thousands of products. Real sellers. Fast delivery.";
  const href = campaign ? `/deals/${campaign.slug}` : "/products";
  const cta = campaign ? "Shop the deal →" : "Shop now →";

  return (
    <>
      <p
        className="text-xs font-semibold uppercase tracking-[0.22em] mb-4 animate-fade-in"
        style={{ color: "#FF8C00" }}
      >
        India&apos;s liveliest marketplace
      </p>
      <h1
        className="text-4xl sm:text-5xl md:text-6xl font-extrabold leading-[1.08] mb-2 animate-fade-up delay-75"
        style={{ fontFamily: "var(--font-syne)" }}
      >
        {title}
      </h1>
      {!campaign && (
        <h1
          className="text-4xl sm:text-5xl md:text-6xl font-extrabold leading-[1.08] animate-fade-up delay-150"
          style={{ fontFamily: "var(--font-syne)", color: "#FF2D78" }}
        >
          ⚡ zapped to your door.
        </h1>
      )}
      <p
        className="mt-5 text-base sm:text-lg max-w-lg animate-fade-up delay-300"
        style={{ color: "rgba(255,255,255,0.75)" }}
      >
        {subtitle}
      </p>
      <div className="mt-8 flex flex-wrap gap-3 animate-fade-up delay-400">
        <Link
          href={href}
          className="inline-block font-bold px-8 py-4 rounded-full text-white cursor-pointer
                     transition-all duration-200 hover:scale-105 hover:shadow-lg active:scale-95
                     btn-shimmer"
          style={{ fontFamily: "var(--font-syne)" }}
        >
          {cta}
        </Link>
        <Link
          href="/products"
          className="inline-block font-semibold px-8 py-4 rounded-full transition-all duration-200
                     hover:bg-white/20 active:scale-95 cursor-pointer"
          style={{ border: "2px solid rgba(255,255,255,0.35)", color: "#fff" }}
        >
          Browse all
        </Link>
      </div>
    </>
  );
}

export default function HeroCarousel({ campaigns }: { campaigns: Campaign[] }) {
  const [idx, setIdx] = useState(0);
  const [slideKey, setSlideKey] = useState("0");

  useEffect(() => {
    if (campaigns.length <= 1) return;
    const t = setInterval(() => {
      setIdx((i) => {
        const next = (i + 1) % campaigns.length;
        setSlideKey(String(next));
        return next;
      });
    }, 5500);
    return () => clearInterval(t);
  }, [campaigns.length]);

  function goTo(i: number) {
    setIdx(i);
    setSlideKey(String(i) + Date.now());
  }

  const bgImage = campaigns[idx]?.banner_image_url;

  return (
    <>
      <div
        className="relative overflow-hidden"
        style={{
          background: "linear-gradient(135deg, #00736A 0%, #005f58 100%)",
          clipPath: "polygon(0 0, 100% 0, 100% 91%, 0 100%)",
          paddingBottom: "5rem",
        }}
      >
        {/* Background image */}
        {bgImage && (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={bgImage}
            alt=""
            className="absolute inset-0 w-full h-full object-cover opacity-15 transition-opacity duration-700"
          />
        )}

        {/* Decorative orbs */}
        <div
          className="absolute -top-32 -right-32 w-[32rem] h-[32rem] rounded-full pointer-events-none animate-float"
          style={{ background: "#FF2D78", opacity: 0.10 }}
        />
        <div
          className="absolute -bottom-16 -left-20 w-72 h-72 rounded-full pointer-events-none"
          style={{ background: "#FF8C00", opacity: 0.08 }}
        />
        <div
          className="absolute top-1/2 right-1/4 w-40 h-40 rounded-full pointer-events-none animate-float"
          style={{ background: "#fff", opacity: 0.04, animationDelay: "2s" }}
        />

        {/* Geometric accent ring */}
        <div
          className="absolute -top-12 -right-12 w-[22rem] h-[22rem] rounded-full pointer-events-none animate-spin-slow"
          style={{ border: "1px solid rgba(255,255,255,0.06)" }}
        />

        {/* Content */}
        <div
          key={slideKey}
          className="relative z-10 max-w-4xl mx-auto px-6 sm:px-10 py-16 sm:py-20 text-white"
        >
          {campaigns.length > 0 ? (
            <SlideContent campaign={campaigns[idx]} key={slideKey} />
          ) : (
            <SlideContent key="default" />
          )}
        </div>

        {/* Dot navigation */}
        {campaigns.length > 1 && (
          <div className="absolute bottom-12 left-0 right-0 flex justify-center gap-2 z-10">
            {campaigns.map((_, i) => (
              <button
                key={i}
                onClick={() => goTo(i)}
                className="rounded-full transition-all duration-300 cursor-pointer"
                style={{
                  width: i === idx ? "24px" : "8px",
                  height: "8px",
                  background: i === idx ? "#FF2D78" : "rgba(255,255,255,0.3)",
                }}
                aria-label={`Slide ${i + 1}`}
              />
            ))}
          </div>
        )}
      </div>

      <TrustStrip />
    </>
  );
}
