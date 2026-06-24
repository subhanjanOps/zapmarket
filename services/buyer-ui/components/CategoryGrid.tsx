import Link from "next/link";

interface Category { id: string; name: string; slug: string; }

const ACCENTS = ["#FF2D78", "#00736A", "#FF8C00", "#6B2FFF", "#0086C9", "#E04A00"];

export default function CategoryGrid({ categories }: { categories: Category[] }) {
  if (!categories.length) return null;
  return (
    <section className="max-w-7xl mx-auto px-4 pt-10 pb-2">
      <p className="text-xs font-semibold uppercase tracking-widest mb-3" style={{ color: "#9CA3AF" }}>
        Browse categories
      </p>
      <div
        className="flex gap-3 overflow-x-auto pb-2"
        style={{ scrollbarWidth: "none" }}
      >
        {categories.slice(0, 12).map((c, i) => {
          const color = ACCENTS[i % ACCENTS.length];
          return (
            <Link
              key={c.id}
              href={`/products?category_id=${c.id}`}
              className="shrink-0 flex items-center gap-2 bg-white rounded-full px-4 py-2 text-sm font-semibold transition-all hover:-translate-y-0.5 hover:shadow-md"
              style={{ border: `2px solid ${color}`, color: "#1A1208" }}
            >
              <span
                className="w-2 h-2 rounded-full shrink-0"
                style={{ background: color }}
              />
              {c.name}
            </Link>
          );
        })}
      </div>
    </section>
  );
}
