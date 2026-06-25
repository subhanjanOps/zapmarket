import type { Metadata } from "next";
import ProductCard from "@/components/ProductCard";
import Paginator from "@/components/Paginator";
import AnimateIn from "@/components/AnimateIn";
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

  // Fetch ALL images for each product (enables card slider)
  const imagesMap: Record<string, { url: string }[]> = {};
  await Promise.all(
    products.map(async (p) => {
      try {
        const r = await fetch(`${GW}/v1/products/${p.id}/images`, { next: { revalidate: 300 } });
        if (!r.ok) return;
        const d = await r.json();
        const imgs: { url: string; position: number }[] = d.data ?? [];
        if (imgs.length > 0) {
          imgs.sort((a, b) => a.position - b.position);
          imagesMap[p.id as string] = imgs.map((img) => ({ url: publicImageUrl(img.url) }));
        }
      } catch { /* keep placeholder */ }
    })
  );

  const baseUrl = `/products${sp.category_id ? `?category_id=${sp.category_id}` : ""}`;

  const SORT_OPTIONS = [
    ["created_at", "desc", "Newest"],
    ["price_amount", "asc", "Price: Low to High"],
    ["price_amount", "desc", "Price: High to Low"],
  ] as const;

  return (
    <div className="max-w-7xl mx-auto px-4 py-8 flex gap-7">
      {/* Sidebar */}
      <aside className="hidden md:block w-52 shrink-0">
        <div
          className="rounded-2xl p-5 sticky top-24"
          style={{ background: "#fff", border: "1px solid #F0EDE8", boxShadow: "0 1px 6px rgba(26,18,8,0.04)" }}
        >
          <h3 className="font-extrabold text-sm mb-3" style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}>
            Categories
          </h3>
          <ul className="space-y-1.5 text-sm">
            <li>
              <Link
                href="/products"
                className={`block px-3 py-1.5 rounded-lg transition-all duration-150 font-semibold hover:text-[#FF2D78] ${!sp.category_id ? "text-[#FF2D78] bg-[#FF2D7808]" : "text-gray-600"}`}
              >
                All
              </Link>
            </li>
            {categories.map((c) => (
              <li key={c.id}>
                <Link
                  href={`/products?category_id=${c.id}`}
                  className={`block px-3 py-1.5 rounded-lg transition-all duration-150 hover:text-[#FF2D78] ${sp.category_id === c.id ? "text-[#FF2D78] font-semibold bg-[#FF2D7808]" : "text-gray-600"}`}
                >
                  {c.name}
                </Link>
              </li>
            ))}
          </ul>

          <div style={{ borderTop: "1px solid #F0EDE8" }} className="mt-5 pt-5">
            <h3 className="font-extrabold text-sm mb-3" style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}>
              Sort by
            </h3>
            <ul className="space-y-1.5 text-sm">
              {SORT_OPTIONS.map(([sb, so, label]) => (
                <li key={label}>
                  <Link
                    href={`/products?sort_by=${sb}&sort_order=${so}${sp.category_id ? `&category_id=${sp.category_id}` : ""}`}
                    className={`block px-3 py-1.5 rounded-lg transition-all duration-150 hover:text-[#FF2D78] ${sp.sort_by === sb && sp.sort_order === so ? "text-[#FF2D78] font-semibold bg-[#FF2D7808]" : "text-gray-600"}`}
                  >
                    {label}
                  </Link>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between mb-5">
          <p className="text-sm font-medium" style={{ color: "#6B6052" }}>
            {total} product{total !== 1 ? "s" : ""}
          </p>
          {sp.search && (
            <p className="text-sm" style={{ color: "#1A1208" }}>
              Results for <strong>&ldquo;{sp.search}&rdquo;</strong>
            </p>
          )}
        </div>

        {products.length === 0 ? (
          <div className="text-center py-20" style={{ color: "#9CA3AF" }}>
            <p className="text-lg font-semibold mb-2" style={{ color: "#1A1208" }}>No products found</p>
            <p className="text-sm">Try a different category or search term.</p>
          </div>
        ) : (
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
            {products.map((p, i) => (
              <AnimateIn key={p.id as string} animation="scale-in" delay={Math.min(i * 40, 300)}>
                <ProductCard
                  product={{
                    ...(p as { id: string; name: string; price_amount?: number; currency?: string; sku_count?: number }),
                    images: imagesMap[p.id as string] ?? undefined,
                  }}
                />
              </AnimateIn>
            ))}
          </div>
        )}

        <Paginator page={page} total={total} limit={PAGE_SIZE} baseUrl={baseUrl} />
      </div>
    </div>
  );
}
