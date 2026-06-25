"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import { ArrowRight } from "lucide-react";
import type { PostMeta } from "@/lib/mdx";

const CATEGORY_COLORS: Record<string, string> = {
  guide:   "#00736A",
  review:  "#6B2FFF",
  news:    "#FF8C00",
  tips:    "#0086C9",
};

function categoryColor(cat?: string): string {
  if (!cat) return "#E91E8C";
  return CATEGORY_COLORS[cat.toLowerCase()] ?? "#E91E8C";
}

export default function BlogCard({ post }: { post: PostMeta }) {
  const accent = categoryColor(post.category);

  return (
    <motion.div
      whileHover={{ y: -3 }}
      transition={{ duration: 0.25, ease: [0.16, 1, 0.3, 1] }}
    >
      <Link
        href={`/blog/${post.slug}`}
        className="group block rounded-2xl overflow-hidden cursor-pointer
                   transition-shadow duration-300 hover:shadow-xl"
        style={{
          background: "#fff",
          border: "1px solid #F0EDE8",
          boxShadow: "0 1px 4px rgba(15,10,4,0.04)",
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
            style={{ background: `${accent}15`, color: accent }}
          >
            {post.category ?? "Article"}
          </span>
          <h3
            className="text-base font-bold line-clamp-2 leading-snug transition-colors duration-150 group-hover:text-[#E91E8C]"
            style={{ color: "#0F0A04", fontFamily: "var(--font-syne)" }}
          >
            {post.title}
          </h3>
          <p className="mt-2 text-sm leading-relaxed line-clamp-2" style={{ color: "#7A6856" }}>
            {post.excerpt}
          </p>
          <div className="mt-4 flex items-center justify-between">
            <p className="text-xs" style={{ color: "#9CA3AF" }}>
              {post.published_at}
            </p>
            <span
              className="flex items-center gap-1 text-xs font-semibold opacity-0 group-hover:opacity-100 transition-opacity duration-200"
              style={{ color: accent }}
            >
              Read
              <ArrowRight size={12} strokeWidth={2.5} />
            </span>
          </div>
        </div>
      </Link>
    </motion.div>
  );
}
