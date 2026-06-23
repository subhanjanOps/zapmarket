import Link from "next/link";
import type { TermMeta } from "@/lib/mdx";

export default function GlossaryEntry({ term }: { term: TermMeta }) {
  return (
    <Link href={`/glossary/${term.slug}`} className="block bg-white border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow">
      <h3 className="font-semibold text-gray-900">{term.term}</h3>
      <p className="text-xs text-gray-500 mt-1">{term.category}</p>
    </Link>
  );
}
