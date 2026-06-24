import type { Metadata } from "next";
import HeroCarousel from "@/components/HeroCarousel";
import CategoryGrid from "@/components/CategoryGrid";
import ProductCard from "@/components/ProductCard";
import BlogCard from "@/components/BlogCard";
import GlossaryEntry from "@/components/GlossaryEntry";
import { getAllPosts, getAllTerms } from "@/lib/mdx";
import Link from "next/link";
import { publicImageUrl } from "@/lib/images";

export const metadata: Metadata = {
  title: "ZapMarket — Shop Smarter",
  description: "India's fastest online marketplace.",
};

export const revalidate = 300;

const GW = process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";

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
    const r = await fetch(`${GW}/v1/categories`, { cache: "no-store" });
    if (!r.ok) return [];
    const d = await r.json();
    return d.data ?? d ?? [];
  } catch { return []; }
}

async function fetchNewArrivals() {
  try {
    const r = await fetch(`${GW}/v1/products?limit=8&sort_by=created_at&sort_order=desc`, { cache: "no-store" });
    if (!r.ok) return [];
    const d = await r.json();
    const products: Record<string, unknown>[] = d.data ?? d ?? [];
    await Promise.all(
      products.map(async (p) => {
        try {
          const ir = await fetch(`${GW}/v1/products/${p.id}/images`, { next: { revalidate: 300 } });
          if (!ir.ok) return;
          const id = await ir.json();
          const imgs: { url: string; position: number }[] = id.data ?? [];
          if (imgs.length > 0) {
            imgs.sort((a, b) => a.position - b.position);
            p.images = [{ url: publicImageUrl(imgs[0].url) }];
          }
        } catch { /* keep placeholder */ }
      })
    );
    return products;
  } catch { return []; }
}

function SectionHeader({ title, href, label }: { title: string; href: string; label: string }) {
  return (
    <div className="flex items-center justify-between mb-5">
      <h2
        className="text-xl font-extrabold"
        style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}
      >
        {title}
      </h2>
      <Link
        href={href}
        className="text-sm font-semibold transition-colors hover:opacity-70"
        style={{ color: "#FF2D78" }}
      >
        {label} →
      </Link>
    </div>
  );
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
          <SectionHeader title="Today's Deals" href="/deals/summer-sale" label="See all deals" />
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
            {newArrivals.slice(0, 4).map((p: Record<string, unknown>) => (
              <ProductCard key={p.id as string} product={p as { id: string; name: string; price_amount?: number; currency?: string; images?: { url: string }[] }} />
            ))}
          </div>
        </section>
      )}

      {newArrivals.length > 0 && (
        <section className="max-w-7xl mx-auto px-4 py-8">
          <SectionHeader title="New Arrivals" href="/products?sort_by=created_at&sort_order=desc" label="See all" />
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
            {newArrivals.map((p: Record<string, unknown>) => (
              <ProductCard key={p.id as string} product={p as { id: string; name: string; price_amount?: number; currency?: string; images?: { url: string }[] }} />
            ))}
          </div>
        </section>
      )}

      {posts.length > 0 && (
        <section className="max-w-7xl mx-auto px-4 py-8">
          <SectionHeader title="From the blog" href="/blog" label="All posts" />
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            {posts.map((p) => <BlogCard key={p.slug} post={p} />)}
          </div>
        </section>
      )}

      {featuredTerms.length > 0 && (
        <section className="max-w-7xl mx-auto px-4 py-8">
          <SectionHeader title="Browse the glossary" href="/glossary" label="Full glossary" />
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-6 gap-3">
            {featuredTerms.map((t) => <GlossaryEntry key={t.slug} term={t} />)}
          </div>
        </section>
      )}
    </>
  );
}
