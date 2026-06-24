import Link from "next/link";
import Image from "next/image";

interface Product {
  id: string;
  name: string;
  slug?: string;
  price_amount?: number;
  currency?: string;
  images?: { url: string }[];
  sku_count?: number;
}

const TOP_COLORS = ["#FF2D78", "#00736A", "#FF8C00", "#6B2FFF"];

function accentFromId(id: string): string {
  let h = 0;
  for (let i = 0; i < id.length; i++) h = ((h * 31) + id.charCodeAt(i)) >>> 0;
  return TOP_COLORS[h % TOP_COLORS.length];
}

export default function ProductCard({ product }: { product: Product }) {
  const img = product.images?.[0]?.url ?? "/placeholder-product.png";
  const price = product.price_amount ? (product.price_amount / 100).toFixed(2) : null;
  const currencySymbol = product.currency === "INR" || !product.currency ? "₹" : product.currency;
  const accent = accentFromId(product.id);

  return (
    <Link
      href={`/products/${product.id}`}
      className="group block bg-white rounded-2xl overflow-hidden transition-all duration-200 hover:-translate-y-1 hover:shadow-lg"
      style={{ border: "1px solid #F0EDE8" }}
    >
      <div className="h-1.5 w-full" style={{ background: accent }} />
      <div className="aspect-square relative" style={{ background: "#FFFCF5" }}>
        <Image
          src={img}
          alt={product.name}
          fill
          className="object-contain p-3"
          sizes="(max-width: 640px) 50vw, 25vw"
        />
      </div>
      <div className="p-3">
        <p
          className="text-sm font-medium line-clamp-2"
          style={{ color: "#1A1208" }}
        >
          {product.name}
        </p>
        {price && (
          <p
            className="mt-2 text-base font-bold"
            style={{ color: "#FF2D78", fontVariantNumeric: "tabular-nums" }}
          >
            {currencySymbol} {price}
          </p>
        )}
        {product.sku_count != null && product.sku_count > 1 && (
          <p className="text-xs mt-1" style={{ color: "#9CA3AF" }}>
            {product.sku_count} variants
          </p>
        )}
      </div>
    </Link>
  );
}
