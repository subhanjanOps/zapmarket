import type { Metadata } from "next";
import ProductCard from "@/components/ProductCard";
import Paginator from "@/components/Paginator";
import Link from "next/link";
import { publicImageUrl } from "@/lib/images";

export const metadata: Metadata = { title: "All Products" };
export const revalidate = 60;

const GW = process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";
const PAGE_SIZE = 20;

interface SearchParams { page?: string; category_id?: string; search?: string; sort_by?: string; sort_order?: string; }

export default async function ProductsPage({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const sp = await searchParams;
  const page = Math.max(1, parseInt(sp.page ?? "1", 10));
  const offset = (page - 1) * PAGE_SIZE;

  const qs = new URLSearchParams({ limit: String(PAGE_SIZE), offset: String(offset) });
  if (sp.category_id) qs.set("category_id", sp.category_id);
  if (sp.search) qs.set("search", sp.search);
  if (sp.sort_by) qs.set("sort_by", sp.sort_by);
  if (sp.sort_order) qs.set("sort_order", sp.sort_order);

  const [productsRes, categoriesRes] = await Promise.all([
    fetch(`${GW}/v1/products?${qs}`, { next: { revalidate: 60 } }).then((r) => r.json()).catch(() => ({ data: [], total: 0 })),
    fetch(`${GW}/v1/categories`, { next: { revalidate: 3600 } }).then((r) => r.json()).catch(() => ({ data: [] })),
  ]);

  const products: Record<string, unknown>[] = productsRes.data ?? productsRes ?? [];
  const total: number = productsRes.total ?? productsRes.pagination?.total ?? products.length;
  const categories: { id: string; name: string }[] = categoriesRes.data ?? categoriesRes ?? [];

  // Fetch images for all products in parallel
  const imagesMap: Record<string, string> = {};
  await Promise.all(
    products.map(async (p) => {
      try {
        const r = await fetch(`${GW}/v1/products/${p.id}/images`, { next: { revalidate: 300 } });
        if (!r.ok) return;
        const d = await r.json();
        const imgs: { url: string; position: number }[] = d.data ?? [];
        if (imgs.length > 0) {
          imgs.sort((a, b) => a.position - b.position);
          imagesMap[p.id as string] = publicImageUrl(imgs[0].url);
        }
      } catch { /* keep placeholder */ }
    })
  );

  const baseUrl = `/products${sp.category_id ? `?category_id=${sp.category_id}` : ""}`;

  return (
    <div className="max-w-7xl mx-auto px-4 py-6 flex gap-6">
      <aside className="hidden md:block w-56 shrink-0">
        <h3 className="font-semibold mb-3">Categories</h3>
        <ul className="space-y-1 text-sm">
          <li><Link href="/products" className="text-gray-700 hover:text-[#FF9900]">All</Link></li>
          {categories.map((c) => (
            <li key={c.id}>
              <Link href={`/products?category_id=${c.id}`}
                className={`hover:text-[#FF9900] ${sp.category_id === c.id ? "text-[#FF9900] font-semibold" : "text-gray-700"}`}>
                {c.name}
              </Link>
            </li>
          ))}
        </ul>
        <div className="mt-6">
          <h3 className="font-semibold mb-3">Sort</h3>
          <ul className="space-y-1 text-sm">
            {[["created_at", "desc", "Newest"], ["price_amount", "asc", "Price: Low to High"], ["price_amount", "desc", "Price: High to Low"]].map(([sb, so, label]) => (
              <li key={label}>
                <Link href={`/products?sort_by=${sb}&sort_order=${so}${sp.category_id ? `&category_id=${sp.category_id}` : ""}`}
                  className={`hover:text-[#FF9900] ${sp.sort_by === sb && sp.sort_order === so ? "text-[#FF9900] font-semibold" : "text-gray-700"}`}>
                  {label}
                </Link>
              </li>
            ))}
          </ul>
        </div>
      </aside>

      <div className="flex-1">
        <div className="flex items-center justify-between mb-4">
          <p className="text-sm text-gray-600">{total} products</p>
          {sp.search && <p className="text-sm">Results for <strong>&ldquo;{sp.search}&rdquo;</strong></p>}
        </div>
        {products.length === 0 ? (
          <div className="text-center py-16 text-gray-500">No products found.</div>
        ) : (
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
            {products.map((p) => (
              <ProductCard
                key={p.id as string}
                product={{
                  ...(p as { id: string; name: string; price_amount?: number; currency?: string }),
                  images: imagesMap[p.id as string] ? [{ url: imagesMap[p.id as string] }] : undefined,
                }}
              />
            ))}
          </div>
        )}
        <Paginator page={page} total={total} limit={PAGE_SIZE} baseUrl={baseUrl} />
      </div>
    </div>
  );
}
