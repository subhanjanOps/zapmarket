import Link from "next/link";

interface Category { id: string; name: string; slug: string; }

export default function CategoryGrid({ categories }: { categories: Category[] }) {
  if (!categories.length) return null;
  return (
    <section className="max-w-7xl mx-auto px-4 py-8">
      <h2 className="text-xl font-bold mb-4">Shop by Category</h2>
      <div className="grid grid-cols-2 sm:grid-cols-4 md:grid-cols-6 gap-3">
        {categories.slice(0, 12).map((c) => (
          <Link key={c.id} href={`/products?category_id=${c.id}`}
            className="bg-white border border-gray-200 rounded-lg p-4 text-center hover:border-[#FF9900] hover:shadow-sm transition-all">
            <p className="text-sm font-medium text-gray-800">{c.name}</p>
          </Link>
        ))}
      </div>
    </section>
  );
}
