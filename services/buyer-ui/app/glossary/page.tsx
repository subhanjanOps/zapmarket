import type { Metadata } from "next";
import { getAllTerms } from "@/lib/mdx";
import GlossaryEntry from "@/components/GlossaryEntry";
import { BookOpen } from "lucide-react";

export const metadata: Metadata = { title: "Glossary" };

export default function GlossaryPage() {
  const terms = getAllTerms().sort((a, b) => a.term.localeCompare(b.term));
  const letters = [...new Set(terms.map((t) => t.term[0].toUpperCase()))];

  return (
    <div className="max-w-5xl mx-auto px-4 py-10">
      <div className="mb-8 flex items-start gap-4">
        <div className="w-10 h-10 rounded-2xl flex items-center justify-center shrink-0 mt-0.5" style={{ background: "#FF2D7812" }}>
          <BookOpen size={18} style={{ color: "#FF2D78" }} />
        </div>
        <div>
          <p className="text-xs font-bold uppercase tracking-widest mb-1" style={{ color: "#FF2D78" }}>Reference</p>
          <h1 className="text-3xl font-extrabold leading-tight" style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}>
            Glossary
          </h1>
          <p className="mt-1 text-sm" style={{ color: "#9CA3AF" }}>
            Common e-commerce and marketplace terms, explained simply.
          </p>
        </div>
      </div>

      {letters.map((letter) => (
        <div key={letter} className="mb-8">
          <div className="flex items-center gap-3 mb-3">
            <span
              className="text-lg font-extrabold w-8 h-8 rounded-lg flex items-center justify-center"
              style={{ background: "#FF2D7812", color: "#FF2D78", fontFamily: "var(--font-syne)" }}
            >
              {letter}
            </span>
            <div className="flex-1 h-px" style={{ background: "#F0EDE8" }} />
          </div>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3">
            {terms.filter((t) => t.term[0].toUpperCase() === letter).map((t) => (
              <GlossaryEntry key={t.slug} term={t} />
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}
