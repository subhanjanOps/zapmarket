"use client";
import Link from "next/link";
import { ShoppingCart } from "lucide-react";
import { useCartStore } from "@/lib/cart";
import { useEffect, useRef } from "react";

export default function CartButton() {
  const count = useCartStore((s) => s.items.reduce((n, i) => n + i.qty, 0));
  const badgeRef = useRef<HTMLSpanElement>(null);
  const prevCount = useRef(count);

  useEffect(() => {
    if (count > prevCount.current && badgeRef.current) {
      badgeRef.current.classList.remove("animate-badge-pop");
      void badgeRef.current.offsetWidth;
      badgeRef.current.classList.add("animate-badge-pop");
    }
    prevCount.current = count;
  }, [count]);

  return (
    <Link
      href="/cart"
      className="relative flex items-center gap-1.5 transition-all duration-150 hover:text-[#FF2D78] hover:-translate-y-px cursor-pointer"
      aria-label={`Cart, ${count} items`}
    >
      <span className="relative">
        <ShoppingCart size={20} />
        {count > 0 && (
          <span
            ref={badgeRef}
            className="animate-badge-pop absolute -top-2.5 -right-2.5 text-white text-[10px] font-bold rounded-full w-4.5 h-4.5 flex items-center justify-center leading-none"
            style={{ background: "#FF2D78", minWidth: "18px", minHeight: "18px", padding: "2px" }}
          >
            {count > 9 ? "9+" : count}
          </span>
        )}
      </span>
      <span className="text-sm font-medium hidden md:inline">Cart</span>
    </Link>
  );
}
