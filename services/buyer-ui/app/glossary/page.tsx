import type { Metadata } from "next";
import { getAllTerms } from "@/lib/mdx";
import GlossaryEntry from "@/components/GlossaryEntry";

export const metadata: Metadata = { title: "Glossary" };

export default function GlossaryPage() {
  const terms = getAllTerms().sort((a, b) => a.term.localeCompare(b.term));
  const letters = [...new Set(terms.map((t) => t.term[0].toUpperCase()))];

  return (
    <div className="max-w-5xl mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold mb-6">Glossary</h1>
      {letters.map((letter) => (
        <div key={letter} className="mb-8">
          <h2 className="text-lg font-bold text-[#FF2D78] mb-3">{letter}</h2>
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
