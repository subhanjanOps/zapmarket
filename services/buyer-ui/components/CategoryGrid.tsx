import Link from "next/link";
import AnimateIn from "./AnimateIn";

interface Category { id: string; name: string; slug: string; }

const ACCENTS = ["#FF2D78", "#00736A", "#FF8C00", "#6B2FFF", "#0086C9", "#E04A00"];

export default function CategoryGrid({ categories }: { categories: Category[] }) {
  if (!categories.length) return null;
  return (
    <section className="max-w-7xl mx-auto px-4 pt-8 pb-2">
      <p
        className="text-[11px] font-bold uppercase tracking-widest mb-3"
        style={{ color: "#9CA3AF" }}
      >
        Browse categories
      </p>
      <div className="flex gap-2.5 overflow-x-auto pb-2" style={{ scrollbarWidth: "none" }}>
        {categories.slice(0, 12).map((c, i) => {
          const color = ACCENTS[i % ACCENTS.length];
          return (
            <AnimateIn
              key={c.id}
              animation="scale-in"
              delay={i * 40}
              className="shrink-0"
            >
              <Link
                href={`/products?category_id=${c.id}`}
                className="flex items-center gap-2 bg-white rounded-full px-4 py-2.5 text-sm font-semibold
                           transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md
                           whitespace-nowrap cursor-pointer"
                style={{
                  border: `2px solid ${color}`,
                  color: "#1A1208",
                  boxShadow: "0 1px 3px rgba(26,18,8,0.04)",
                }}
              >
                <span className="w-2.5 h-2.5 rounded-full shrink-0" style={{ background: color }} />
                {c.name}
              </Link>
            </AnimateIn>
          );
        })}
      </div>
    </section>
  );
}
