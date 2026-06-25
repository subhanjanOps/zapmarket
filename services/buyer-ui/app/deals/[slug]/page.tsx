import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Tag } from "lucide-react";
import ProductCard from "@/components/ProductCard";
import CountdownTimer from "@/components/CountdownTimer";
import { GW } from "@/lib/api";

export const revalidate = 60;

function publicImageUrl(raw: string): string {
  if (!raw) return "/placeholder-product.png";
  if (raw.startsWith("http")) return raw;
  const gwBase = (process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000").replace(/\/$/, "");
  return `${gwBase}/${raw.replace(/^\//, "")}`;
}

export async function generateMetadata({ params }: { params: Promise<{ slug: string }> }): Promise<Metadata> {
  const { slug } = await params;
  try {
    const r = await fetch(`${GW}/v1/campaigns/${slug}`, { next: { revalidate: 60 } });
    if (!r.ok) return { title: "Deal not found" };
    const c = await r.json();
    return { title: c.title };
  } catch { return { title: "Deal" }; }
}

export default async function DealPage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  let campaign: Record<string, unknown> | null = null;
  try {
    const r = await fetch(`${GW}/v1/campaigns/${slug}`, { next: { revalidate: 60 } });
    if (r.ok) campaign = await r.json();
  } catch { /* fall through */ }

  if (!campaign) notFound();

  const productIds: string[] = (campaign.product_ids as string[]) ?? [];
  const products: Record<string, unknown>[] = [];

  if (productIds.length) {
    const results = await Promise.allSettled(
      productIds.slice(0, 16).map(async (id) => {
        const [pRes, imgRes] = await Promise.allSettled([
          fetch(`${GW}/v1/products/${id}`, { next: { revalidate: 300 } }),
          fetch(`${GW}/v1/products/${id}/images`, { next: { revalidate: 300 } }),
        ]);
        const p = pRes.status === "fulfilled" && pRes.value.ok ? await pRes.value.json() : null;
        if (!p) return null;
        const imgs = imgRes.status === "fulfilled" && imgRes.value.ok ? await imgRes.value.json() : [];
        const imageArr: { url: string }[] = Array.isArray(imgs)
          ? imgs.map((img: { url: string }) => ({ url: publicImageUrl(img.url) }))
          : [];
        return { ...p, images: imageArr };
      })
    );
    results.forEach((r) => { if (r.status === "fulfilled" && r.value) products.push(r.value); });
  }

  return (
    <div>
      {/* Deal hero */}
      <div
        className="relative py-24 text-center overflow-hidden"
        style={{ background: "linear-gradient(135deg, #00736A 0%, #005952 100%)" }}
      >
        {typeof campaign.banner_image_url === "string" && campaign.banner_image_url && (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={campaign.banner_image_url}
            alt={campaign.title as string}
            className="absolute inset-0 w-full h-full object-cover opacity-15"
          />
        )}
        <div className="relative z-10 px-4">
          <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full mb-4 text-xs font-bold uppercase tracking-widest"
            style={{ background: "rgba(255,255,255,0.15)", color: "rgba(255,255,255,0.9)" }}>
            <Tag size={11} /> Limited Time Deal
          </div>
          <h1
            className="text-4xl sm:text-5xl font-extrabold text-white leading-tight"
            style={{ fontFamily: "var(--font-syne)" }}
          >
            {campaign.title as string}
          </h1>
          {typeof campaign.headline === "string" && campaign.headline && (
            <p className="mt-4 text-lg max-w-xl mx-auto" style={{ color: "rgba(255,255,255,0.75)" }}>
              {campaign.headline}
            </p>
          )}
          {typeof campaign.valid_until === "string" && campaign.valid_until && (
            <div className="mt-6 flex justify-center">
              <CountdownTimer validUntil={campaign.valid_until} />
            </div>
          )}
        </div>
      </div>

      {/* Products */}
      <div className="max-w-7xl mx-auto px-4 py-10">
        {products.length > 0 ? (
          <>
            <p className="text-sm font-semibold mb-5" style={{ color: "#6B6052" }}>
              {products.length} product{products.length !== 1 ? "s" : ""} in this deal
            </p>
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
              {products.map((p) => (
                <ProductCard
                  key={p.id as string}
                  product={p as { id: string; name: string; base_price: number; price_amount?: number; currency?: string; images?: { url: string }[] }}
                />
              ))}
            </div>
          </>
        ) : (
          <p className="text-center py-16" style={{ color: "#9CA3AF" }}>No products in this deal yet.</p>
        )}
      </div>
    </div>
  );
}
