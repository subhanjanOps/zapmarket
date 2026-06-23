import type { Metadata } from "next";
import { getAllPosts } from "@/lib/mdx";
import BlogCard from "@/components/BlogCard";

export const metadata: Metadata = { title: "Blog", description: "Buying guides, trends, and more." };

export default function BlogPage() {
  const posts = getAllPosts();
  return (
    <div className="max-w-5xl mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold mb-8">From the Blog</h1>
      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-6">
        {posts.map((p) => <BlogCard key={p.slug} post={p} />)}
      </div>
    </div>
  );
}
