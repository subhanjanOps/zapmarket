"use client";
import Link from "next/link";
import { ShoppingCart } from "lucide-react";
import { useCartStore } from "@/lib/cart";

export default function CartButton() {
  const count = useCartStore((s) => s.items.reduce((n, i) => n + i.qty, 0));
  return (
    <Link href="/cart" className="relative flex items-center gap-1 hover:text-[#FF9900]">
      <ShoppingCart size={20} />
      {count > 0 && (
        <span className="absolute -top-2 -right-2 bg-[#FF9900] text-black text-xs font-bold rounded-full w-5 h-5 flex items-center justify-center">
          {count}
        </span>
      )}
      <span className="text-sm">Cart</span>
    </Link>
  );
}
