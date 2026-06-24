import Link from "next/link";
import type { PostMeta } from "@/lib/mdx";

export default function BlogCard({ post }: { post: PostMeta }) {
  return (
    <Link href={`/blog/${post.slug}`} className="block bg-white border border-gray-200 rounded-lg p-5 hover:shadow-md transition-shadow">
      <span className="text-xs font-medium text-[#FF2D78] uppercase tracking-wide">{post.category}</span>
      <h3 className="mt-1 text-base font-semibold text-gray-900 line-clamp-2">{post.title}</h3>
      <p className="mt-2 text-sm text-gray-600 line-clamp-3">{post.excerpt}</p>
      <p className="mt-3 text-xs text-gray-400">{post.published_at}</p>
    </Link>
  );
}
