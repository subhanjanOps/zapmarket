import Link from "next/link";
import type { TermMeta } from "@/lib/mdx";

export default function GlossaryEntry({ term }: { term: TermMeta }) {
  const firstLetter = term.term?.charAt(0).toUpperCase() ?? "#";

  return (
    <Link
      href={`/glossary/${term.slug}`}
      className="
        group block
        bg-white rounded-2xl border border-[#EDE9E3]
        p-4 gap-3 flex items-start
        transition-all duration-200
        hover:shadow-[0_8px_24px_rgba(15,10,4,0.1)]
        hover:border-transparent
        hover:-translate-y-0.5
      "
    >
      <span
        className="
          shrink-0 w-10 h-10 flex items-center justify-center
          rounded-xl bg-[#FDF0F7]
          font-display text-2xl font-bold text-[#E91E8C]
          leading-none select-none
        "
        aria-hidden="true"
      >
        {firstLetter}
      </span>

      <div className="min-w-0 flex flex-col gap-0.5">
        <span className="font-semibold text-sm text-[#0F0A04] leading-snug truncate group-hover:text-[#E91E8C] transition-colors duration-200">
          {term.term}
        </span>
        {term.category && (
          <span className="text-xs text-[#7A6856] truncate">
            {term.category}
          </span>
        )}
      </div>
    </Link>
  );
}
