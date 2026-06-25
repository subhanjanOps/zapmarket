import Link from "next/link";
import type { PostMeta } from "@/lib/mdx";

const CATEGORY_COLORS: Record<string, string> = {
  guide:   "#00736A",
  review:  "#6B2FFF",
  news:    "#FF8C00",
  tips:    "#0086C9",
};

function categoryColor(cat?: string): string {
  if (!cat) return "#FF2D78";
  return CATEGORY_COLORS[cat.toLowerCase()] ?? "#FF2D78";
}

export default function BlogCard({ post }: { post: PostMeta }) {
  const accent = categoryColor(post.category);

  return (
    <Link
      href={`/blog/${post.slug}`}
      className="group block rounded-2xl overflow-hidden cursor-pointer
                 transition-all duration-300 hover:-translate-y-1.5 hover:shadow-xl"
      style={{
        background: "#fff",
        border: "1px solid #F0EDE8",
        boxShadow: "0 1px 4px rgba(26,18,8,0.04)",
      }}
    >
      {/* Colored header band */}
      <div
        className="h-1.5 w-full transition-all duration-300 group-hover:h-2"
        style={{ background: accent }}
      />

      <div className="p-5">
        <span
          className="inline-block text-[10px] font-bold uppercase tracking-[0.18em] px-2.5 py-1 rounded-full mb-3"
          style={{ background: `${accent}18`, color: accent }}
        >
          {post.category ?? "Article"}
        </span>
        <h3
          className="text-base font-bold line-clamp-2 leading-snug transition-colors duration-150 group-hover:text-[#FF2D78]"
          style={{ color: "#1A1208", fontFamily: "var(--font-syne)" }}
        >
          {post.title}
        </h3>
        <p className="mt-2 text-sm leading-relaxed line-clamp-2" style={{ color: "#6B6052" }}>
          {post.excerpt}
        </p>
        <div className="mt-4 flex items-center justify-between">
          <p className="text-xs" style={{ color: "#9CA3AF" }}>
            {post.published_at}
          </p>
          <span
            className="text-xs font-semibold opacity-0 group-hover:opacity-100 transition-opacity duration-200"
            style={{ color: accent }}
          >
            Read →
          </span>
        </div>
      </div>
    </Link>
  );
}
