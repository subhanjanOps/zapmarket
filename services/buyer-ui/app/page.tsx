import type { Metadata } from "next";
import { HeroSection } from "@/components/HeroSection";
import { FeaturedCategories } from "@/components/FeaturedCategories";
import { ProductCard } from "@/components/ProductCard";
import { TrustBanner } from "@/components/TrustBanner";
import { SectionHeader } from "@/components/SectionHeader";
import { MotionWrapper, MotionChild } from "@/components/MotionWrapper";
import BlogCard from "@/components/BlogCard";
import GlossaryEntry from "@/components/GlossaryEntry";
import { getAllPosts, getAllTerms } from "@/lib/mdx";
import { publicImageUrl } from "@/lib/images";

export const metadata: Metadata = {
  title: "ZapMarket — Shop Smarter",
  description: "India's fastest online marketplace. Browse products, read guides, and shop deals.",
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

async function fetchProductsWithImages(query: string) {
  try {
    const r = await fetch(`${GW}/v1/products?${query}`, { next: { revalidate: 60 } });
    if (!r.ok) return [];
    const d = await r.json();
    const products: Record<string, unknown>[] = d.data ?? d.products ?? d ?? [];
    await Promise.all(
      products.map(async p => {
        try {
          const ir = await fetch(`${GW}/v1/products/${p.id}/images`, { next: { revalidate: 300 } });
          if (!ir.ok) return;
          const id = await ir.json();
          const imgs: { url: string; position: number }[] = id.data ?? id.images ?? [];
          if (imgs.length > 0) {
            imgs.sort((a, b) => a.position - b.position);
            p.images = imgs.map(img => ({ url: publicImageUrl(img.url) }));
          }
        } catch { /* no images */ }
      })
    );
    return products;
  } catch { return []; }
}

export default async function HomePage() {
  const [campaigns, categories, newArrivals, deals] = await Promise.all([
    fetchCampaigns(),
    fetchCategories(),
    fetchProductsWithImages("limit=8&sort_by=created_at&sort_order=desc"),
    fetchProductsWithImages("limit=4&sort_by=base_price&sort_order=asc"),
  ]);

  const posts = getAllPosts().slice(0, 3);
  const featuredTerms = getAllTerms().slice(0, 6);

  return (
    <main>
      {/* Hero */}
      <HeroSection campaigns={campaigns} />

      {/* Categories */}
      {categories.length > 0 && (
        <FeaturedCategories categories={categories} />
      )}

      {/* Today's Deals */}
      {deals.length > 0 && (
        <section className="container-zap py-8 sm:py-10">
          <SectionHeader
            title="Today's Deals"
            subtitle="Handpicked savings, updated daily"
            href="/products"
            viewAllLabel="All deals"
            accent
            className="mb-6 sm:mb-8"
          />
          <MotionWrapper stagger className="grid grid-cols-2 sm:grid-cols-2 md:grid-cols-4 gap-4">
            {deals.slice(0, 4).map((p, i) => (
              <MotionChild key={p.id as string}>
                <ProductCard
                  product={p as Parameters<typeof ProductCard>[0]["product"]}
                  priority={i < 2}
                />
              </MotionChild>
            ))}
          </MotionWrapper>
        </section>
      )}

      {/* Trust Banner */}
      <TrustBanner />

      {/* New Arrivals */}
      {newArrivals.length > 0 && (
        <section className="container-zap py-10 sm:py-12">
          <SectionHeader
            title="New Arrivals"
            subtitle="The freshest products, just landed"
            href="/products?sort_by=created_at&sort_order=desc"
            className="mb-6 sm:mb-8"
          />
          <MotionWrapper stagger className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
            {newArrivals.slice(0, 8).map((p, i) => (
              <MotionChild key={p.id as string}>
                <ProductCard
                  product={p as Parameters<typeof ProductCard>[0]["product"]}
                  priority={i < 4}
                />
              </MotionChild>
            ))}
          </MotionWrapper>
        </section>
      )}

      {/* Blog */}
      {posts.length > 0 && (
        <section className="bg-[#F3F0EB] py-10 sm:py-12">
          <div className="container-zap">
            <SectionHeader
              title="From the Blog"
              subtitle="Shopping guides, tips, and insider knowledge"
              href="/blog"
              className="mb-6 sm:mb-8"
            />
            <MotionWrapper stagger className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
              {posts.map(post => (
                <MotionChild key={post.slug}>
                  <BlogCard post={post} />
                </MotionChild>
              ))}
            </MotionWrapper>
          </div>
        </section>
      )}

      {/* Glossary */}
      {featuredTerms.length > 0 && (
        <section className="container-zap py-10 sm:py-12">
          <SectionHeader
            title="Marketplace Glossary"
            subtitle="Know exactly what you're buying"
            href="/glossary"
            viewAllLabel="Full glossary"
            className="mb-6 sm:mb-8"
          />
          <MotionWrapper stagger className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-6 gap-3">
            {featuredTerms.map(t => (
              <MotionChild key={t.slug}>
                <GlossaryEntry term={t} />
              </MotionChild>
            ))}
          </MotionWrapper>
        </section>
      )}
    </main>
  );
}
