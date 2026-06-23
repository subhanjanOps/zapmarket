import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { MDXRemote } from "next-mdx-remote/rsc";
import { getTerm, getAllTerms } from "@/lib/mdx";
import Link from "next/link";
import ProductCard from "@/components/ProductCard";

const GW = process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";
export const revalidate = 3600;

export async function generateStaticParams() {
  return getAllTerms().map((t) => ({ term: t.slug }));
}

export async function generateMetadata({ params }: { params: Promise<{ term: string }> }): Promise<Metadata> {
  const { term } = await params;
  const entry = getTerm(term);
  return entry ? { title: `${entry.meta.term} — Glossary` } : { title: "Term not found" };
}

export default async function GlossaryTermPage({ params }: { params: Promise<{ term: string }> }) {
  const { term: slug } = await params;
  const entry = getTerm(slug);
  if (!entry) notFound();

  const relatedProducts: Record<string, unknown>[] = [];
  if (entry.meta.product_slugs?.length) {
    const results = await Promise.allSettled(
      entry.meta.product_slugs.slice(0, 4).map((s) =>
        fetch(`${GW}/v1/products/slug/${s}`, { next: { revalidate: 3600 } }).then((r) => r.ok ? r.json() : null)
      )
    );
    results.forEach((r) => { if (r.status === "fulfilled" && r.value) relatedProducts.push(r.value); });
  }

  return (
    <div className="max-w-3xl mx-auto px-4 py-8">
      <Link href="/glossary" className="text-sm text-[#FF9900] hover:underline">← Back to Glossary</Link>
      <h1 className="text-3xl font-bold mt-3">{entry.meta.term}</h1>
      <p className="text-xs text-gray-500 mb-6">{entry.meta.category}</p>
      <article className="prose prose-lg max-w-none">
        <MDXRemote source={entry.content} />
      </article>
      {entry.meta.related_terms?.length > 0 && (
        <div className="mt-8">
          <h3 className="font-semibold mb-3">Related Terms</h3>
          <div className="flex flex-wrap gap-2">
            {entry.meta.related_terms.map((t) => (
              <Link key={t} href={`/glossary/${t}`} className="px-3 py-1 border border-gray-300 rounded text-sm hover:border-[#FF9900] hover:text-[#FF9900]">{t}</Link>
            ))}
          </div>
        </div>
      )}
      {relatedProducts.length > 0 && (
        <section className="mt-12">
          <h2 className="text-xl font-bold mb-4">Related Products</h2>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
            {relatedProducts.map((p) => <ProductCard key={p.id as string} product={p as { id: string; name: string; price_amount?: number; currency?: string; images?: { url: string }[] }} />)}
          </div>
        </section>
      )}
    </div>
  );
}
