"use client";
import { useState, useEffect } from "react";
import Link from "next/link";

interface Campaign {
  id: string;
  slug: string;
  title: string;
  headline: string;
  banner_image_url: string | null;
}

function HeroShell({
  bgImage,
  children,
  dots,
}: {
  bgImage?: string | null;
  children: React.ReactNode;
  dots?: React.ReactNode;
}) {
  return (
    <div
      className="relative overflow-hidden"
      style={{
        background: "#00736A",
        clipPath: "polygon(0 0, 100% 0, 100% 91%, 0 100%)",
        paddingBottom: "5rem",
      }}
    >
      {bgImage && (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={bgImage}
          alt=""
          className="absolute inset-0 w-full h-full object-cover opacity-20"
        />
      )}
      {/* Decorative orbs */}
      <div
        className="absolute -top-24 -right-24 w-[28rem] h-[28rem] rounded-full pointer-events-none"
        style={{ background: "#FF2D78", opacity: 0.12 }}
      />
      <div
        className="absolute bottom-0 -left-16 w-64 h-64 rounded-full pointer-events-none"
        style={{ background: "#FF8C00", opacity: 0.1 }}
      />
      <div className="relative z-10 max-w-4xl mx-auto px-8 py-20 text-white">
        {children}
      </div>
      {dots && (
        <div className="absolute bottom-10 left-0 right-0 flex justify-center gap-2 z-10">
          {dots}
        </div>
      )}
    </div>
  );
}

export default function HeroCarousel({ campaigns }: { campaigns: Campaign[] }) {
  const [idx, setIdx] = useState(0);
  useEffect(() => {
    if (campaigns.length <= 1) return;
    const t = setInterval(() => setIdx((i) => (i + 1) % campaigns.length), 5000);
    return () => clearInterval(t);
  }, [campaigns.length]);

  if (!campaigns.length) {
    return (
      <HeroShell>
        <p
          className="text-xs font-semibold uppercase tracking-[0.2em] mb-4"
          style={{ color: "#FF8C00" }}
        >
          India&apos;s liveliest marketplace
        </p>
        <h1
          className="text-5xl md:text-6xl font-extrabold leading-tight"
          style={{ fontFamily: "var(--font-syne)" }}
        >
          Everything you need,
          <br />
          <span style={{ color: "#FF2D78" }}>⚡ zapped</span> to your door.
        </h1>
        <p className="mt-4 text-lg max-w-lg" style={{ color: "rgba(255,255,255,0.75)" }}>
          Thousands of products. Real sellers. Fast delivery. No nonsense.
        </p>
        <Link
          href="/products"
          className="mt-8 inline-block font-bold px-8 py-4 rounded-full text-white transition-transform hover:scale-105 active:scale-95"
          style={{ background: "#FF2D78", fontFamily: "var(--font-syne)" }}
        >
          Shop now →
        </Link>
      </HeroShell>
    );
  }

  const c = campaigns[idx];
  return (
    <HeroShell
      bgImage={c.banner_image_url}
      dots={campaigns.length > 1 ? campaigns.map((_, i) => (
        <button
          key={i}
          onClick={() => setIdx(i)}
          className="w-2 h-2 rounded-full transition-all"
          style={{ background: i === idx ? "#FF2D78" : "rgba(255,255,255,0.35)" }}
          aria-label={`Slide ${i + 1}`}
        />
      )) : undefined}
    >
      <h1
        className="text-5xl font-extrabold leading-tight"
        style={{ fontFamily: "var(--font-syne)" }}
      >
        {c.title}
      </h1>
      <p className="mt-3 text-xl" style={{ color: "rgba(255,255,255,0.8)" }}>
        {c.headline}
      </p>
      <Link
        href={`/deals/${c.slug}`}
        className="mt-8 inline-block font-bold px-8 py-4 rounded-full text-white transition-transform hover:scale-105 active:scale-95"
        style={{ background: "#FF2D78", fontFamily: "var(--font-syne)" }}
      >
        Shop the deal →
      </Link>
    </HeroShell>
  );
}
