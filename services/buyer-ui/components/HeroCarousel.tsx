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

export default function HeroCarousel({ campaigns }: { campaigns: Campaign[] }) {
  const [idx, setIdx] = useState(0);
  useEffect(() => {
    if (campaigns.length <= 1) return;
    const t = setInterval(() => setIdx((i) => (i + 1) % campaigns.length), 5000);
    return () => clearInterval(t);
  }, [campaigns.length]);

  if (!campaigns.length) {
    return (
      <div className="bg-[#232F3E] text-white text-center py-20">
        <h1 className="text-3xl font-bold">Welcome to ZapMarket</h1>
        <p className="mt-2 text-gray-300">Shop smarter. Shop faster.</p>
        <Link href="/products" className="mt-6 inline-block bg-[#FF9900] text-black font-semibold px-6 py-3 rounded hover:bg-[#e68900]">
          Shop Now
        </Link>
      </div>
    );
  }

  const c = campaigns[idx];
  return (
    <div className="relative bg-[#232F3E] text-white overflow-hidden" style={{ minHeight: 320 }}>
      {c.banner_image_url && (
        // eslint-disable-next-line @next/next/no-img-element
        <img src={c.banner_image_url} alt={c.title} className="absolute inset-0 w-full h-full object-cover opacity-30" />
      )}
      <div className="relative z-10 max-w-4xl mx-auto px-8 py-16 text-center">
        <h1 className="text-4xl font-bold">{c.title}</h1>
        <p className="mt-3 text-xl text-gray-200">{c.headline}</p>
        <Link href={`/deals/${c.slug}`} className="mt-6 inline-block bg-[#FF9900] text-black font-semibold px-6 py-3 rounded hover:bg-[#e68900]">
          Shop the Deal
        </Link>
      </div>
      {campaigns.length > 1 && (
        <div className="absolute bottom-4 left-0 right-0 flex justify-center gap-2">
          {campaigns.map((_, i) => (
            <button key={i} onClick={() => setIdx(i)}
              className={`w-2 h-2 rounded-full ${i === idx ? "bg-[#FF9900]" : "bg-gray-500"}`} />
          ))}
        </div>
      )}
    </div>
  );
}
