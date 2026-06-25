import type { Metadata } from "next";
import { getAllPosts } from "@/lib/mdx";
import BlogCard from "@/components/BlogCard";

export const metadata: Metadata = { title: "Blog", description: "Buying guides, trends, and more." };

export default function BlogPage() {
  const posts = getAllPosts();
  return (
    <div className="max-w-5xl mx-auto px-4 py-10">
      <div className="mb-8">
        <p className="text-xs font-bold uppercase tracking-widest mb-2" style={{ color: "#FF2D78" }}>Journal</p>
        <h1
          className="text-3xl sm:text-4xl font-extrabold leading-tight"
          style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}
        >
          From the Blog
        </h1>
        <p className="mt-2 text-sm" style={{ color: "#9CA3AF" }}>
          Buying guides, product trends, and tips from the ZapMarket team.
        </p>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-6">
        {posts.map((p) => <BlogCard key={p.slug} post={p} />)}
      </div>
      {posts.length === 0 && (
        <p className="text-center py-16 text-sm" style={{ color: "#9CA3AF" }}>No posts yet — check back soon.</p>
      )}
    </div>
  );
}
