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

export default function ProductCard({ product }: { product: Product }) {
  const img = product.images?.[0]?.url ?? "/placeholder-product.png";
  const price = product.price_amount ? (product.price_amount / 100).toFixed(2) : null;
  return (
    <Link href={`/products/${product.id}`} className="group block bg-white rounded-lg border border-gray-200 overflow-hidden hover:shadow-md transition-shadow">
      <div className="aspect-square relative bg-gray-100">
        <Image src={img} alt={product.name} fill className="object-contain p-2" sizes="(max-width: 640px) 50vw, 25vw" />
      </div>
      <div className="p-3">
        <p className="text-sm font-medium text-gray-900 line-clamp-2 group-hover:text-[#FF9900]">{product.name}</p>
        {price && (
          <p className="mt-1 text-sm font-bold text-gray-900">
            {product.currency ?? "INR"} {price}
          </p>
        )}
        {product.sku_count != null && product.sku_count > 1 && (
          <p className="text-xs text-gray-500 mt-1">{product.sku_count} variants</p>
        )}
      </div>
    </Link>
  );
}
