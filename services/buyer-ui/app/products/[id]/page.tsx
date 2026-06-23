import type { Metadata } from "next";
import Image from "next/image";
import ProductInteractions from "@/components/ProductInteractions";
import { publicImageUrl } from "@/lib/images";

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

export default async function ProductDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;

  const [productRes, skusRes, imagesRes] = await Promise.all([
    fetch(`${GW}/v1/products/${id}`, { next: { revalidate: 300 } }).then((r) => r.ok ? r.json() : null).catch(() => null),
    fetch(`${GW}/v1/skus?product_id=${id}`, { next: { revalidate: 300 } }).then((r) => r.ok ? r.json() : { data: [] }).catch(() => ({ data: [] })),
    fetch(`${GW}/v1/products/${id}/images`, { next: { revalidate: 300 } }).then((r) => r.ok ? r.json() : { data: [] }).catch(() => ({ data: [] })),
  ]);

  if (!productRes) {
    return (
      <div className="text-center py-20 text-gray-500">
        Product not found. <a href="/products" className="text-[#FF9900]">Back to shopping</a>
      </div>
    );
  }

  const skus = skusRes.data ?? skusRes ?? [];
  const rawImages: { url: string; position: number }[] = imagesRes.data ?? [];
  rawImages.sort((a, b) => a.position - b.position);
  const firstImage = publicImageUrl(rawImages[0]?.url);

  return (
    <div className="max-w-5xl mx-auto px-4 py-8">
      <div className="flex flex-col md:flex-row gap-8">
        <div className="md:w-1/2">
          <div className="aspect-square relative bg-gray-100 rounded-lg overflow-hidden border border-gray-200">
            <Image src={firstImage} alt={productRes.name} fill className="object-contain p-4" sizes="(max-width: 768px) 100vw, 50vw" />
          </div>
        </div>
        <div className="md:w-1/2">
          <h1 className="text-2xl font-bold text-gray-900">{productRes.name}</h1>
          {productRes.description && <p className="mt-3 text-gray-600">{productRes.description}</p>}
          {skus.length > 0 && (
            <div className="mt-4">
              <ProductInteractions skus={skus} productName={productRes.name} productImage={firstImage} />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
