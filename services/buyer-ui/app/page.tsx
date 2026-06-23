import type { Metadata } from "next";
import HeroCarousel from "@/components/HeroCarousel";
import CategoryGrid from "@/components/CategoryGrid";
import ProductCard from "@/components/ProductCard";
import BlogCard from "@/components/BlogCard";
import GlossaryEntry from "@/components/GlossaryEntry";
import { getAllPosts, getAllTerms } from "@/lib/mdx";
import Link from "next/link";

export const metadata: Metadata = {
  title: "ZapMarket — Shop Smarter",
  description: "India's fastest online marketplace.",
};

export const revalidate = 300;

const GW = process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";

async function fetchCampaigns() {
  try {
    const r = await fetch(`${GW}/v1/campaigns`, { next: { revalidate: 300 } });
    if (!r.ok) return [];
    const d = await r.json();
    return d.data ?? d ?? [];
  } catch { return []; }
}

async function fetchCategories() {
  try {
    const r = await fetch(`${GW}/v1/categories`, { next: { revalidate: 3600 } });
    if (!r.ok) return [];
    const d = await r.json();
    return d.data ?? d ?? [];
  } catch { return []; }
}

async function fetchNewArrivals() {
  try {
    const r = await fetch(`${GW}/v1/products?limit=8&sort_by=created_at&sort_order=desc`, { next: { revalidate: 300 } });
    if (!r.ok) return [];
    const d = await r.json();
    return d.data ?? d ?? [];
  } catch { return []; }
}

export default async function HomePage() {
  const [campaigns, categories, newArrivals] = await Promise.all([
    fetchCampaigns(),
    fetchCategories(),
    fetchNewArrivals(),
  ]);

  const posts = getAllPosts().slice(0, 3);
  const featuredTerms = getAllTerms().slice(0, 6);

  return (
    <>
      <HeroCarousel campaigns={campaigns} />
      <CategoryGrid categories={categories} />

      {newArrivals.length > 0 && (
        <section className="max-w-7xl mx-auto px-4 py-8">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-bold">Today&apos;s Deals</h2>
            <Link href="/deals/summer-sale" className="text-sm text-[#FF9900] hover:underline">See all deals →</Link>
          </div>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
            {newArrivals.slice(0, 4).map((p: Record<string, unknown>) => (
              <ProductCard key={p.id as string} product={p as { id: string; name: string; price_amount?: number; currency?: string; images?: { url: string }[] }} />
            ))}
          </div>
        </section>
      )}

      {newArrivals.length > 0 && (
        <section className="max-w-7xl mx-auto px-4 py-8">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-bold">New Arrivals</h2>
            <Link href="/products?sort_by=created_at&sort_order=desc" className="text-sm text-[#FF9900] hover:underline">See all →</Link>
          </div>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
            {newArrivals.map((p: Record<string, unknown>) => (
              <ProductCard key={p.id as string} product={p as { id: string; name: string; price_amount?: number; currency?: string; images?: { url: string }[] }} />
            ))}
          </div>
        </section>
      )}

      {posts.length > 0 && (
        <section className="max-w-7xl mx-auto px-4 py-8">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-bold">From the Blog</h2>
            <Link href="/blog" className="text-sm text-[#FF9900] hover:underline">All posts →</Link>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            {posts.map((p) => <BlogCard key={p.slug} post={p} />)}
          </div>
        </section>
      )}

      {featuredTerms.length > 0 && (
        <section className="max-w-7xl mx-auto px-4 py-8">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-bold">Browse the Glossary</h2>
            <Link href="/glossary" className="text-sm text-[#FF9900] hover:underline">Full glossary →</Link>
          </div>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-6 gap-3">
            {featuredTerms.map((t) => <GlossaryEntry key={t.slug} term={t} />)}
          </div>
        </section>
      )}
    </>
  );
}
