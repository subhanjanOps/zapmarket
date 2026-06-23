import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { MDXRemote } from "next-mdx-remote/rsc";
import { getPost, getAllPosts } from "@/lib/mdx";
import ProductCard from "@/components/ProductCard";

const GW = process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";
export const revalidate = 3600;

export async function generateStaticParams() {
  return getAllPosts().map((p) => ({ slug: p.slug }));
}

export async function generateMetadata({ params }: { params: Promise<{ slug: string }> }): Promise<Metadata> {
  const { slug } = await params;
  const post = getPost(slug);
  if (!post) return { title: "Post not found" };
  return { title: post.meta.title, description: post.meta.excerpt };
}

export default async function BlogPostPage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const post = getPost(slug);
  if (!post) notFound();

  const relatedProducts: Record<string, unknown>[] = [];
  if (post.meta.product_slugs?.length) {
    const results = await Promise.allSettled(
      post.meta.product_slugs.slice(0, 4).map((s) =>
        fetch(`${GW}/v1/products/slug/${s}`, { next: { revalidate: 3600 } }).then((r) => r.ok ? r.json() : null)
      )
    );
    results.forEach((r) => { if (r.status === "fulfilled" && r.value) relatedProducts.push(r.value); });
  }

  return (
    <div className="max-w-3xl mx-auto px-4 py-8">
      <p className="text-xs text-[#FF9900] font-medium uppercase tracking-wide">{post.meta.category}</p>
      <h1 className="text-3xl font-bold mt-2 mb-2">{post.meta.title}</h1>
      <p className="text-sm text-gray-500 mb-8">{post.meta.published_at}</p>
      <article className="prose prose-lg max-w-none">
        <MDXRemote source={post.content} />
      </article>
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
