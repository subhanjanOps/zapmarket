import type { Metadata } from "next";
import Link from "next/link";
import ProductCard from "@/components/ProductCard";
import Paginator from "@/components/Paginator";
import AnimateIn from "@/components/AnimateIn";
import { publicImageUrl } from "@/lib/images";

import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button, buttonVariants } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import SortSelect from "@/components/SortSelect";
import { cn } from "@/lib/utils";

export const metadata: Metadata = { title: "All Products" };
export const revalidate = 60;

const GW =
  process.env.GATEWAY_URL ??
  process.env.NEXT_PUBLIC_GATEWAY_URL ??
  "http://localhost:8000";
const PAGE_SIZE = 20;

interface SearchParams {
  page?: string;
  category_id?: string;
  search?: string;
  sort_by?: string;
  sort_order?: string;
}

const SORT_OPTIONS = [
  { sort_by: "created_at", sort_order: "desc", label: "Newest" },
  { sort_by: "price_amount", sort_order: "asc", label: "Price: Low to High" },
  { sort_by: "price_amount", sort_order: "desc", label: "Price: High to Low" },
] as const;

function buildSortValue(sort_by?: string, sort_order?: string) {
  if (!sort_by || !sort_order) return "created_at:desc";
  return `${sort_by}:${sort_order}`;
}

function buildFilterHref(
  sp: SearchParams,
  overrides: Partial<SearchParams>
): string {
  const merged = { ...sp, ...overrides, page: "1" };
  const qs = new URLSearchParams();
  if (merged.category_id) qs.set("category_id", merged.category_id);
  if (merged.search) qs.set("search", merged.search);
  if (merged.sort_by) qs.set("sort_by", merged.sort_by);
  if (merged.sort_order) qs.set("sort_order", merged.sort_order);
  const str = qs.toString();
  return str ? `/products?${str}` : "/products";
}

function CategoryList({
  categories,
  activeCategoryId,
  sp,
}: {
  categories: { id: string; name: string }[];
  activeCategoryId?: string;
  sp: SearchParams;
}) {
  return (
    <div className="space-y-1">
      <Link href={buildFilterHref(sp, { category_id: undefined })}>
        <Button
          variant={!activeCategoryId ? "secondary" : "ghost"}
          className="w-full justify-start rounded-xl text-sm font-medium"
        >
          All
        </Button>
      </Link>
      {categories.map((c) => (
        <Link key={c.id} href={buildFilterHref(sp, { category_id: c.id })}>
          <Button
            variant={activeCategoryId === c.id ? "secondary" : "ghost"}
            className="w-full justify-start rounded-xl text-sm font-medium"
          >
            {c.name}
          </Button>
        </Link>
      ))}
    </div>
  );
}

export default async function ProductsPage({
  searchParams,
}: {
  searchParams: Promise<SearchParams>;
}) {
  const sp = await searchParams;
  const page = Math.max(1, parseInt(sp.page ?? "1", 10));
  const offset = (page - 1) * PAGE_SIZE;

  const qs = new URLSearchParams({
    limit: String(PAGE_SIZE),
    offset: String(offset),
  });
  if (sp.category_id) qs.set("category_id", sp.category_id);
  if (sp.search) qs.set("search", sp.search);
  if (sp.sort_by) qs.set("sort_by", sp.sort_by);
  if (sp.sort_order) qs.set("sort_order", sp.sort_order);

  const [productsRes, categoriesRes] = await Promise.all([
    fetch(`${GW}/v1/products?${qs}`, { next: { revalidate: 60 } })
      .then((r) => r.json())
      .catch(() => ({ data: [], total: 0 })),
    fetch(`${GW}/v1/categories`, { next: { revalidate: 3600 } })
      .then((r) => r.json())
      .catch(() => ({ data: [] })),
  ]);

  const products: Record<string, unknown>[] =
    productsRes.data ?? productsRes ?? [];
  const total: number =
    productsRes.total ?? productsRes.pagination?.total ?? products.length;
  const categories: { id: string; name: string }[] =
    categoriesRes.data ?? categoriesRes ?? [];

  // Fetch ALL images for each product (enables card slider)
  const imagesMap: Record<string, { url: string }[]> = {};
  await Promise.all(
    products.map(async (p) => {
      try {
        const r = await fetch(`${GW}/v1/products/${p.id}/images`, {
          next: { revalidate: 300 },
        });
        if (!r.ok) return;
        const d = await r.json();
        const imgs: { url: string; position: number }[] = d.data ?? [];
        if (imgs.length > 0) {
          imgs.sort((a, b) => a.position - b.position);
          imagesMap[p.id as string] = imgs.map((img) => ({
            url: publicImageUrl(img.url),
          }));
        }
      } catch {
        /* keep placeholder */
      }
    })
  );

  const baseUrl = `/products${sp.category_id ? `?category_id=${sp.category_id}` : ""}`;

  const activeCategory = categories.find((c) => c.id === sp.category_id);
  const activeSort = SORT_OPTIONS.find(
    (o) => o.sort_by === sp.sort_by && o.sort_order === sp.sort_order
  );
  const currentSortValue = buildSortValue(sp.sort_by, sp.sort_order);

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      <div className="flex gap-7">
        {/* Desktop Sidebar */}
        <aside className="hidden md:block w-56 shrink-0">
          <Card className="sticky top-24 rounded-2xl">
            <CardContent className="p-5 space-y-5">
              <div>
                <h3 className="font-extrabold text-sm mb-3 text-foreground">
                  Categories
                </h3>
                <CategoryList
                  categories={categories}
                  activeCategoryId={sp.category_id}
                  sp={sp}
                />
              </div>

              <Separator />

              <div>
                <h3 className="font-extrabold text-sm mb-3 text-foreground">
                  Sort by
                </h3>
                <div className="space-y-1">
                  {SORT_OPTIONS.map((opt) => (
                    <Link
                      key={opt.label}
                      href={buildFilterHref(sp, {
                        sort_by: opt.sort_by,
                        sort_order: opt.sort_order,
                      })}
                    >
                      <Button
                        variant={
                          activeSort?.label === opt.label
                            ? "secondary"
                            : "ghost"
                        }
                        className="w-full justify-start rounded-xl text-sm font-medium"
                      >
                        {opt.label}
                      </Button>
                    </Link>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>
        </aside>

        {/* Main content */}
        <div className="flex-1 min-w-0">
          {/* Header row */}
          <div className="flex items-center justify-between mb-5 gap-3 flex-wrap">
            <div className="flex items-center gap-3">
              <h1 className="text-xl font-extrabold text-foreground">
                Products
              </h1>
              <Badge variant="secondary" className="rounded-full px-2.5">
                {total} result{total !== 1 ? "s" : ""}
              </Badge>

              {/* Mobile filter trigger */}
              <div className="md:hidden">
                <Sheet>
                  <SheetTrigger className={cn(buttonVariants({ variant: "outline", size: "sm" }), "rounded-xl")}>
                    Filters
                  </SheetTrigger>
                  <SheetContent side="left" className="w-72 p-5">
                    <SheetHeader className="mb-4">
                      <SheetTitle>Filters</SheetTitle>
                    </SheetHeader>
                    <div className="space-y-5">
                      <div>
                        <h3 className="font-extrabold text-sm mb-3">
                          Categories
                        </h3>
                        <CategoryList
                          categories={categories}
                          activeCategoryId={sp.category_id}
                          sp={sp}
                        />
                      </div>
                      <Separator />
                      <div>
                        <h3 className="font-extrabold text-sm mb-3">
                          Sort by
                        </h3>
                        <div className="space-y-1">
                          {[
                            { sort_by: "created_at", sort_order: "desc", label: "Newest" },
                            { sort_by: "price_amount", sort_order: "asc", label: "Price: Low to High" },
                            { sort_by: "price_amount", sort_order: "desc", label: "Price: High to Low" },
                          ].map((opt) => (
                            <Link
                              key={opt.label}
                              href={buildFilterHref(sp, {
                                sort_by: opt.sort_by,
                                sort_order: opt.sort_order,
                              })}
                              className={cn(
                                buttonVariants({ variant: activeSort?.label === opt.label ? "secondary" : "ghost" }),
                                "w-full justify-start rounded-xl text-sm"
                              )}
                            >
                              {opt.label}
                            </Link>
                          ))}
                        </div>
                      </div>
                    </div>
                  </SheetContent>
                </Sheet>
              </div>
            </div>

            {/* Sort select (desktop header) */}
            <div className="hidden md:block">
              <SortSelect
                baseHref={`/products${sp.category_id ? `?category_id=${sp.category_id}` : ""}${sp.search ? `?search=${encodeURIComponent(sp.search)}` : ""}`}
                currentValue={currentSortValue}
              />
            </div>
          </div>

          {/* Active filter badges */}
          {(sp.category_id || sp.search || (sp.sort_by && sp.sort_order)) && (
            <div className="flex flex-wrap gap-2 mb-5">
              {sp.search && (
                <Badge
                  variant="outline"
                  className="rounded-full flex items-center gap-1 pr-1"
                >
                  <span>Search: {sp.search}</span>
                  <Link href={buildFilterHref(sp, { search: undefined })}>
                    <button
                      className="ml-1 rounded-full hover:bg-muted p-0.5 text-muted-foreground hover:text-foreground transition-colors"
                      aria-label="Remove search filter"
                    >
                      ×
                    </button>
                  </Link>
                </Badge>
              )}
              {activeCategory && (
                <Badge
                  variant="outline"
                  className="rounded-full flex items-center gap-1 pr-1"
                >
                  <span>{activeCategory.name}</span>
                  <Link href={buildFilterHref(sp, { category_id: undefined })}>
                    <button
                      className="ml-1 rounded-full hover:bg-muted p-0.5 text-muted-foreground hover:text-foreground transition-colors"
                      aria-label="Remove category filter"
                    >
                      ×
                    </button>
                  </Link>
                </Badge>
              )}
              {activeSort && activeSort.label !== "Newest" && (
                <Badge
                  variant="outline"
                  className="rounded-full flex items-center gap-1 pr-1"
                >
                  <span>{activeSort.label}</span>
                  <Link
                    href={buildFilterHref(sp, {
                      sort_by: undefined,
                      sort_order: undefined,
                    })}
                  >
                    <button
                      className="ml-1 rounded-full hover:bg-muted p-0.5 text-muted-foreground hover:text-foreground transition-colors"
                      aria-label="Remove sort filter"
                    >
                      ×
                    </button>
                  </Link>
                </Badge>
              )}
            </div>
          )}

          {/* Search label */}
          {sp.search && (
            <p className="text-sm mb-4 text-muted-foreground">
              Results for{" "}
              <strong className="text-foreground">
                &ldquo;{sp.search}&rdquo;
              </strong>
            </p>
          )}

          {/* Product grid */}
          {products.length === 0 ? (
            <div className="text-center py-20">
              <p className="text-lg font-semibold mb-2 text-foreground">
                No products found
              </p>
              <p className="text-sm text-muted-foreground">
                Try a different category or search term.
              </p>
            </div>
          ) : (
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
              {products.map((p, i) => (
                <AnimateIn
                  key={p.id as string}
                  animation="scale-in"
                  delay={Math.min(i * 40, 300)}
                >
                  <ProductCard
                    product={{
                      ...(p as {
                        id: string;
                        name: string;
                        price_amount?: number;
                        currency?: string;
                        sku_count?: number;
                      }),
                      images: imagesMap[p.id as string] ?? undefined,
                    }}
                  />
                </AnimateIn>
              ))}
            </div>
          )}

          <Paginator
            page={page}
            total={total}
            limit={PAGE_SIZE}
            baseUrl={baseUrl}
          />
        </div>
      </div>
    </div>
  );
}
