import type { Metadata } from "next";
import Link from "next/link";
import { Truck, RotateCcw, ShieldCheck, Package } from "lucide-react";

import ProductInteractions from "@/components/ProductInteractions";
import ProductImageGallery from "@/components/ProductImageGallery";
import ProductCard from "@/components/ProductCard";
import AnimateIn from "@/components/AnimateIn";
import { publicImageUrl } from "@/lib/images";

import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

const GW =
  process.env.GATEWAY_URL ??
  process.env.NEXT_PUBLIC_GATEWAY_URL ??
  "http://localhost:8000";

export async function generateMetadata({
  params,
}: {
  params: Promise<{ id: string }>;
}): Promise<Metadata> {
  const { id } = await params;
  try {
    const r = await fetch(`${GW}/v1/products/${id}`, {
      next: { revalidate: 300 },
    });
    if (!r.ok) return { title: "Product" };
    const p = await r.json();
    return { title: p.name ?? "Product", description: p.description };
  } catch {
    return { title: "Product" };
  }
}

async function fetchRelated(
  categoryId: string | undefined,
  currentId: string
) {
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
    await Promise.all(
      products.map(async (p) => {
        try {
          const ir = await fetch(`${GW}/v1/products/${p.id}/images`, {
            next: { revalidate: 300 },
          });
          if (!ir.ok) return;
          const id = await ir.json();
          const imgs: { url: string; position: number }[] = id.data ?? [];
          if (imgs.length > 0) {
            imgs.sort((a, b) => a.position - b.position);
            p.images = imgs.map((img) => ({ url: publicImageUrl(img.url) }));
          }
        } catch {
          /* skip */
        }
      })
    );
    return products.slice(0, 4);
  } catch {
    return [];
  }
}

export default async function ProductDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;

  const [productRes, skusRes, imagesRes] = await Promise.all([
    fetch(`${GW}/v1/products/${id}`, { next: { revalidate: 300 } })
      .then((r) => (r.ok ? r.json() : null))
      .catch(() => null),
    fetch(`${GW}/v1/skus?product_id=${id}`, { next: { revalidate: 300 } })
      .then((r) => (r.ok ? r.json() : { data: [] }))
      .catch(() => ({ data: [] })),
    fetch(`${GW}/v1/products/${id}/images`, { next: { revalidate: 300 } })
      .then((r) => (r.ok ? r.json() : { data: [] }))
      .catch(() => ({ data: [] })),
  ]);

  if (!productRes) {
    return (
      <div className="text-center py-24">
        <div className="w-16 h-16 rounded-2xl mx-auto mb-5 flex items-center justify-center bg-muted">
          <Package size={28} className="text-muted-foreground" />
        </div>
        <p className="text-lg font-semibold mb-2">Product not found.</p>
        <p className="text-sm mb-6 text-muted-foreground">
          It may have been removed or the link is wrong.
        </p>
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

  const basePrice: number | undefined =
    skus.length > 0 ? (skus[0] as Record<string, unknown>).price_amount as number : undefined;

  return (
    <div className="bg-background">
      <div className="max-w-6xl mx-auto px-4 py-8">

        {/* ── Breadcrumb ── */}
        <Breadcrumb className="mb-7">
          <BreadcrumbList>
            <BreadcrumbItem>
              <BreadcrumbLink render={<Link href="/" />}>Home</BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator />
            <BreadcrumbItem>
              <BreadcrumbLink render={<Link href="/products" />}>Products</BreadcrumbLink>
            </BreadcrumbItem>
            {productRes.category_name && (
              <>
                <BreadcrumbSeparator />
                <BreadcrumbItem>
                  <BreadcrumbLink render={<Link href={`/products?category_id=${productRes.category_id}`} />}>
                    {productRes.category_name}
                  </BreadcrumbLink>
                </BreadcrumbItem>
              </>
            )}
            <BreadcrumbSeparator />
            <BreadcrumbItem>
              <BreadcrumbPage className="line-clamp-1 max-w-[200px]">
                {productRes.name}
              </BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>

        {/* ── Main two-column grid ── */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-10 lg:gap-14">

          {/* Left: Image gallery */}
          <div className="animate-scale-in">
            <ProductImageGallery images={allImages} productName={productRes.name} />
          </div>

          {/* Right: Product info panel */}
          <div className="animate-fade-up delay-100 flex flex-col gap-5">

            {/* Category badge */}
            {productRes.category_name && (
              <div>
                <Badge variant="secondary" className="uppercase tracking-widest text-[10px] font-bold">
                  {productRes.category_name}
                </Badge>
              </div>
            )}

            {/* Product name */}
            <h1
              className="text-2xl font-bold leading-tight"
              style={{ fontFamily: "var(--font-syne)" }}
            >
              {productRes.name}
            </h1>

            {/* Price */}
            {basePrice !== undefined && (
              <p className="text-primary text-3xl font-bold">
                {productRes.currency ?? "USD"}{" "}
                {(basePrice / 100).toFixed(2)}
              </p>
            )}

            <Separator />

            {/* Interactions: SKU selector + Add to cart */}
            {skus.length > 0 ? (
              <ProductInteractions
                skus={skus}
                productName={productRes.name}
                productImage={firstImage}
              />
            ) : (
              <Card className="border-amber-200 bg-amber-50">
                <CardContent className="py-4 text-sm text-amber-800">
                  This product is currently unavailable. Check back soon.
                </CardContent>
              </Card>
            )}

            <Separator />

            {/* Trust strip */}
            <div className="flex flex-wrap gap-2">
              <Badge variant="outline" className="flex items-center gap-1.5 py-1.5 px-3 text-xs font-medium">
                <Truck size={13} className="text-primary" />
                Free shipping over $50
              </Badge>
              <Badge variant="outline" className="flex items-center gap-1.5 py-1.5 px-3 text-xs font-medium">
                <RotateCcw size={13} className="text-primary" />
                30-day returns
              </Badge>
              <Badge variant="outline" className="flex items-center gap-1.5 py-1.5 px-3 text-xs font-medium">
                <ShieldCheck size={13} className="text-primary" />
                Buyer protection
              </Badge>
            </div>
          </div>
        </div>

        {/* ── Details / Shipping tabs ── */}
        <AnimateIn animation="fade-up" delay={150} className="mt-12">
          <Tabs defaultValue="details">
            <TabsList className="mb-4">
              <TabsTrigger value="details">Details</TabsTrigger>
              <TabsTrigger value="shipping">Shipping</TabsTrigger>
            </TabsList>

            <TabsContent value="details">
              <Card>
                <CardContent className="py-5 text-sm leading-relaxed text-muted-foreground">
                  {productRes.description ? (
                    <p>{productRes.description}</p>
                  ) : (
                    <p>No additional details available for this product.</p>
                  )}
                  {skus.length > 0 && (
                    <p className="mt-3 text-foreground font-medium">
                      {skus.length} variant{skus.length !== 1 ? "s" : ""} available
                    </p>
                  )}
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="shipping">
              <Card>
                <CardContent className="py-5 text-sm leading-relaxed text-muted-foreground space-y-2">
                  <p>Standard shipping: 5–7 business days.</p>
                  <p>Express shipping: 2–3 business days (additional fee applies).</p>
                  <p>Free standard shipping on orders over $50.</p>
                  <p>We ship to all major regions. Delivery times may vary by location.</p>
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </AnimateIn>

        {/* ── Related products ── */}
        {related.length > 0 && (
          <section className="mt-14">
            <AnimateIn animation="slide-left">
              <div className="flex items-center justify-between mb-6">
                <h2
                  className="text-xl font-extrabold"
                  style={{ fontFamily: "var(--font-syne)" }}
                >
                  You might also like
                </h2>
                {productRes.category_id && (
                  <Link
                    href={`/products?category_id=${productRes.category_id}`}
                    className="text-sm font-semibold text-primary hover:opacity-70 transition-opacity"
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
                    product={
                      p as {
                        id: string;
                        name: string;
                        price_amount?: number;
                        currency?: string;
                        images?: { url: string }[];
                      }
                    }
                  />
                </AnimateIn>
              ))}
            </div>
          </section>
        )}

        <div className="h-16" />
      </div>
    </div>
  );
}
