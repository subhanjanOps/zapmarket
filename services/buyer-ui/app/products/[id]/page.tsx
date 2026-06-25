import type { Metadata } from "next";
import Link from "next/link";
import ProductInteractions from "@/components/ProductInteractions";
import ProductImageGallery from "@/components/ProductImageGallery";
import ProductCard from "@/components/ProductCard";
import AnimateIn from "@/components/AnimateIn";
import { publicImageUrl } from "@/lib/images";
import { Package, Tag, BarChart2, Layers } from "lucide-react";

const GW = process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";

export async function generateMetadata({ params }: { params: Promise<{ id: string }> }): Promise<Metadata> {
  const { id } = await params;
  try {
    const r = await fetch(`${GW}/v1/products/${id}`, { next: { revalidate: 300 } });
    if (!r.ok) return { title: "Product" };
    const p = await r.json();
    return { title: p.name ?? "Product", description: p.description };
  } catch { return { title: "Product" }; }
}

async function fetchRelated(categoryId: string | undefined, currentId: string) {
  if (!categoryId) return [];
  try {
    const r = await fetch(
      `${GW}/v1/products?category_id=${categoryId}&limit=4`,
      { next: { revalidate: 300 } }
    );
    if (!r.ok) return [];
    const d = await r.json();
    const products: Record<string, unknown>[] = (d.data ?? d ?? []).filter(
      (p: Record<string, unknown>) => p.id !== currentId
    );
    // Fetch first image for each
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
        } catch { /* skip */ }
      })
    );
    return products.slice(0, 4);
  } catch { return []; }
}

export default async function ProductDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;

  const [productRes, skusRes, imagesRes] = await Promise.all([
    fetch(`${GW}/v1/products/${id}`, { next: { revalidate: 300 } }).then((r) => r.ok ? r.json() : null).catch(() => null),
    fetch(`${GW}/v1/skus?product_id=${id}`, { next: { revalidate: 300 } }).then((r) => r.ok ? r.json() : { data: [] }).catch(() => ({ data: [] })),
    fetch(`${GW}/v1/products/${id}/images`, { next: { revalidate: 300 } }).then((r) => r.ok ? r.json() : { data: [] }).catch(() => ({ data: [] })),
  ]);

  if (!productRes) {
    return (
      <div className="text-center py-24">
        <div className="w-16 h-16 rounded-2xl mx-auto mb-5 flex items-center justify-center" style={{ background: "#F0EDE8" }}>
          <Package size={28} style={{ color: "#9CA3AF" }} />
        </div>
        <p className="text-lg font-semibold mb-2" style={{ color: "#1A1208" }}>Product not found.</p>
        <p className="text-sm mb-6" style={{ color: "#9CA3AF" }}>It may have been removed or the link is wrong.</p>
        <Link
          href="/products"
          className="inline-block font-bold px-6 py-3 rounded-xl text-white transition-all duration-200 hover:scale-105 active:scale-95"
          style={{ background: "#FF2D78", fontFamily: "var(--font-syne)" }}
        >
          Back to shopping
        </Link>
      </div>
    );
  }

  const skus = skusRes.data ?? skusRes ?? [];
  const rawImages: { url: string; position: number }[] = imagesRes.data ?? [];
  rawImages.sort((a, b) => a.position - b.position);
  const allImages = rawImages.map((img) => publicImageUrl(img.url));
  const firstImage = allImages[0] ?? "/placeholder-product.png";

  const related = await fetchRelated(productRes.category_id, id);

  // Build highlight tiles from available product data
  const highlights = [
    productRes.category_name && { icon: Tag,      label: "Category",  value: productRes.category_name },
    skus.length > 0          && { icon: Layers,   label: "Variants",  value: `${skus.length} option${skus.length !== 1 ? "s" : ""}` },
    productRes.sku            && { icon: BarChart2,label: "SKU",       value: productRes.sku },
  ].filter(Boolean) as { icon: React.ElementType; label: string; value: string }[];

  return (
    <div style={{ background: "#FFFCF5" }}>
      <div className="max-w-6xl mx-auto px-4 py-8">

        {/* Breadcrumb */}
        <nav className="text-xs mb-7 flex items-center gap-1.5 flex-wrap" style={{ color: "#9CA3AF" }}>
          <Link href="/" className="hover:text-[#FF2D78] transition-colors">Home</Link>
          <span>/</span>
          <Link href="/products" className="hover:text-[#FF2D78] transition-colors">Products</Link>
          {productRes.category_name && (
            <>
              <span>/</span>
              <Link
                href={`/products?category_id=${productRes.category_id}`}
                className="hover:text-[#FF2D78] transition-colors"
              >
                {productRes.category_name}
              </Link>
            </>
          )}
          <span>/</span>
          <span style={{ color: "#1A1208" }} className="font-medium line-clamp-1">{productRes.name}</span>
        </nav>

        {/* Main two-column layout */}
        <div className="flex flex-col lg:flex-row gap-10 lg:gap-14">

          {/* ── Left: Image gallery ── */}
          <div className="lg:w-[48%] animate-scale-in">
            <ProductImageGallery images={allImages} productName={productRes.name} />
          </div>

          {/* ── Right: Product info ── */}
          <div className="lg:w-[52%] animate-fade-up delay-100">

            {/* Category badge */}
            {productRes.category_name && (
              <span
                className="inline-flex items-center gap-1 text-[10px] font-bold uppercase tracking-widest px-3 py-1.5 rounded-full mb-4"
                style={{ background: "#FF2D7812", color: "#FF2D78" }}
              >
                <Tag size={10} />
                {productRes.category_name}
              </span>
            )}

            {/* Title */}
            <h1
              className="text-2xl sm:text-3xl font-extrabold leading-tight"
              style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}
            >
              {productRes.name}
            </h1>

            {/* Description */}
            {productRes.description && (
              <p
                className="mt-4 text-sm leading-relaxed"
                style={{ color: "#6B6052", maxWidth: "44ch" }}
              >
                {productRes.description}
              </p>
            )}

            {/* Divider */}
            <div className="mt-6" style={{ borderTop: "1px solid #F0EDE8" }} />

            {/* Interactions: price, SKU selector, qty, CTA, trust */}
            {skus.length > 0 ? (
              <div className="mt-6">
                <ProductInteractions
                  skus={skus}
                  productName={productRes.name}
                  productImage={firstImage}
                />
              </div>
            ) : (
              <div
                className="mt-6 rounded-2xl p-5 text-sm"
                style={{ background: "#FFF8EC", border: "1px solid #FFE0A0", color: "#92400E" }}
              >
                This product is currently unavailable. Check back soon.
              </div>
            )}
          </div>
        </div>

        {/* ── Highlights strip ── */}
        {highlights.length > 0 && (
          <AnimateIn animation="fade-up" delay={150} className="mt-12">
            <div
              className="rounded-2xl p-6 grid grid-cols-2 sm:grid-cols-3 gap-5"
              style={{ background: "#fff", border: "1px solid #F0EDE8", boxShadow: "0 2px 12px rgba(26,18,8,0.04)" }}
            >
              {highlights.map(({ icon: Icon, label, value }) => (
                <div key={label} className="flex items-start gap-3">
                  <div
                    className="w-9 h-9 rounded-xl flex items-center justify-center shrink-0"
                    style={{ background: "#E6F4F2" }}
                  >
                    <Icon size={16} style={{ color: "#00736A" }} />
                  </div>
                  <div>
                    <p className="text-[10px] font-bold uppercase tracking-wider" style={{ color: "#9CA3AF" }}>{label}</p>
                    <p className="text-sm font-semibold mt-0.5" style={{ color: "#1A1208" }}>{value}</p>
                  </div>
                </div>
              ))}
            </div>
          </AnimateIn>
        )}

        {/* ── Related products ── */}
        {related.length > 0 && (
          <section className="mt-14">
            <AnimateIn animation="slide-left">
              <div className="flex items-center justify-between mb-6">
                <h2
                  className="text-xl font-extrabold"
                  style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}
                >
                  You might also like
                </h2>
                {productRes.category_id && (
                  <Link
                    href={`/products?category_id=${productRes.category_id}`}
                    className="text-sm font-semibold hover:opacity-70 transition-opacity"
                    style={{ color: "#FF2D78" }}
                  >
                    See all →
                  </Link>
                )}
              </div>
            </AnimateIn>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
              {related.map((p, i) => (
                <AnimateIn key={p.id as string} animation="scale-in" delay={i * 60}>
                  <ProductCard
                    product={p as { id: string; name: string; price_amount?: number; currency?: string; images?: { url: string }[] }}
                  />
                </AnimateIn>
              ))}
            </div>
          </section>
        )}

        {/* Bottom padding */}
        <div className="h-16" />
      </div>
    </div>
  );
}
