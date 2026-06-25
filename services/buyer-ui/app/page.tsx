import type { Metadata } from "next";
import HeroCarousel from "@/components/HeroCarousel";
import CategoryGrid from "@/components/CategoryGrid";
import ProductCard from "@/components/ProductCard";
import BlogCard from "@/components/BlogCard";
import GlossaryEntry from "@/components/GlossaryEntry";
import AnimateIn from "@/components/AnimateIn";
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
    const r = await fetch(`${GW}/v1/categories`, { next: { revalidate: 300 } });
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
            p.images = imgs.map((img) => ({ url: publicImageUrl(img.url) }));
          }
        } catch { /* keep placeholder */ }
      })
    );
    return products;
  } catch { return []; }
}

function SectionHeader({ title, href, label }: { title: string; href: string; label: string }) {
  return (
    <div className="flex items-center justify-between mb-6">
      <h2
        className="text-xl font-extrabold"
        style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}
      >
        {title}
      </h2>
      <Link
        href={href}
        className="text-sm font-semibold transition-all duration-150 hover:gap-2 hover:opacity-80 flex items-center gap-1"
        style={{ color: "#FF2D78" }}
      >
        {label} →
      </Link>
    </div>
  );
}

function ProductGrid({ products }: { products: Record<string, unknown>[] }) {
  return (
    <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
      {products.map((p, i) => (
        <AnimateIn key={p.id as string} animation="scale-in" delay={i * 50}>
          <ProductCard product={p as { id: string; name: string; price_amount?: number; currency?: string; images?: { url: string }[] }} />
        </AnimateIn>
      ))}
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
        <section className="max-w-7xl mx-auto px-4 py-10">
          <AnimateIn animation="slide-left">
            <SectionHeader title="Today's Deals" href="/deals/summer-sale" label="See all deals" />
          </AnimateIn>
          <ProductGrid products={newArrivals.slice(0, 4)} />
        </section>
      )}

      {newArrivals.length > 0 && (
        <section className="max-w-7xl mx-auto px-4 py-10">
          <AnimateIn animation="slide-left">
            <SectionHeader title="New Arrivals" href="/products?sort_by=created_at&sort_order=desc" label="See all" />
          </AnimateIn>
          <ProductGrid products={newArrivals} />
        </section>
      )}

      {posts.length > 0 && (
        <section className="max-w-7xl mx-auto px-4 py-10">
          <AnimateIn animation="slide-left">
            <SectionHeader title="From the blog" href="/blog" label="All posts" />
          </AnimateIn>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            {posts.map((p, i) => (
              <AnimateIn key={p.slug} animation="fade-up" delay={i * 80}>
                <BlogCard post={p} />
              </AnimateIn>
            ))}
          </div>
        </section>
      )}

      {featuredTerms.length > 0 && (
        <section className="max-w-7xl mx-auto px-4 py-10">
          <AnimateIn animation="slide-left">
            <SectionHeader title="Browse the glossary" href="/glossary" label="Full glossary" />
          </AnimateIn>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-6 gap-3">
            {featuredTerms.map((t, i) => (
              <AnimateIn key={t.slug} animation="scale-in" delay={i * 40}>
                <GlossaryEntry term={t} />
              </AnimateIn>
            ))}
          </div>
        </section>
      )}
    </>
  );
}
