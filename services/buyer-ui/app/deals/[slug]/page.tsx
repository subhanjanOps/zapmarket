import type { Metadata } from "next";
import { notFound } from "next/navigation";
import ProductCard from "@/components/ProductCard";
import CountdownTimer from "@/components/CountdownTimer";

const GW = process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";
export const revalidate = 60;

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
  } catch { /* fall through to notFound */ }

  if (!campaign) notFound();

  const productIds: string[] = (campaign.product_ids as string[]) ?? [];
  const products: Record<string, unknown>[] = [];
  if (productIds.length) {
    const results = await Promise.allSettled(
      productIds.slice(0, 16).map((id) =>
        fetch(`${GW}/v1/products/${id}`, { next: { revalidate: 300 } }).then((r) => r.ok ? r.json() : null)
      )
    );
    results.forEach((r) => { if (r.status === "fulfilled" && r.value) products.push(r.value); });
  }

  return (
    <div>
      <div className="relative bg-[#232F3E] text-white py-20 text-center overflow-hidden">
        {typeof campaign.banner_image_url === "string" && campaign.banner_image_url && (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={campaign.banner_image_url} alt={campaign.title as string} className="absolute inset-0 w-full h-full object-cover opacity-20" />
        )}
        <div className="relative z-10">
          <h1 className="text-4xl font-bold">{campaign.title as string}</h1>
          <p className="mt-3 text-xl text-gray-200">{campaign.headline as string}</p>
          {typeof campaign.valid_until === "string" && campaign.valid_until && (
            <div className="mt-4">
              <CountdownTimer validUntil={campaign.valid_until} />
            </div>
          )}
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 py-10">
        {products.length > 0 ? (
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
            {products.map((p) => <ProductCard key={p.id as string} product={p as { id: string; name: string; price_amount?: number; currency?: string; images?: { url: string }[] }} />)}
          </div>
        ) : (
          <p className="text-center text-gray-500 py-12">No products in this deal yet.</p>
        )}
      </div>
    </div>
  );
}
