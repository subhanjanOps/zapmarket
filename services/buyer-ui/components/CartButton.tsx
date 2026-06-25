"use client";
import Link from "next/link";
import { ShoppingCart } from "lucide-react";
import { useCartStore } from "@/lib/cart";
import { useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";

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
    <Link href="/cart" aria-label={`Cart, ${count} items`}>
      <Button
        variant="ghost"
        size="icon"
        className="relative transition-all duration-150 hover:text-[#FF2D78] hover:-translate-y-px"

      >
        <ShoppingCart size={20} />
        {count > 0 && (
          <span
            ref={badgeRef}
            className="animate-badge-pop absolute -top-1.5 -right-1.5 bg-primary text-primary-foreground text-[10px] font-bold rounded-full flex items-center justify-center leading-none"
            style={{ minWidth: "18px", minHeight: "18px", padding: "2px" }}
          >
            {count > 9 ? "9+" : count}
          </span>
        )}
      </Button>
    </Link>
  );
}
