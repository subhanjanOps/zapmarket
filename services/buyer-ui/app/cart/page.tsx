"use client";

import Image from "next/image";
import Link from "next/link";
import { useCartStore } from "@/lib/cart";
import { Minus, Plus, Trash2, ShoppingBag } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button, buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Separator } from "@/components/ui/separator";
import { Badge } from "@/components/ui/badge";

export default function CartPage() {
  const { items, removeItem, updateQty, total } = useCartStore();

  if (items.length === 0) {
    return (
      <div className="max-w-3xl mx-auto px-4 py-24 flex justify-center">
        <Card className="w-full max-w-sm text-center shadow-sm">
          <CardContent className="pt-10 pb-10 flex flex-col items-center gap-4">
            <div className="w-16 h-16 rounded-full bg-pink-50 flex items-center justify-center">
              <ShoppingBag size={28} className="text-pink-500" />
            </div>
            <div className="space-y-1">
              <h2 className="text-xl font-bold" style={{ fontFamily: "var(--font-syne)" }}>
                Your cart is empty
              </h2>
              <p className="text-sm text-muted-foreground">
                Looks like you haven&apos;t added anything yet.
              </p>
            </div>
            <Link href="/products" className={cn(buttonVariants(), "mt-2 bg-[#FF2D78] hover:bg-[#e02068] text-white font-bold px-8")}>
              Browse products
            </Link>
          </CardContent>
        </Card>
      </div>
    );
  }

  const subtotal = total();

  return (
    <div className="max-w-5xl mx-auto px-4 py-10">
      <h1
        className="text-2xl font-bold mb-7"
        style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}
      >
        Shopping Cart
      </h1>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Cart Items — lg:col-span-2 */}
        <div className="lg:col-span-2 space-y-3">
          {items.map((item) => (
            <Card key={item.skuId} className="shadow-sm">
              <CardContent className="p-4 flex gap-4 items-center">
                {/* Image */}
                <div className="w-12 h-12 relative shrink-0 rounded-lg overflow-hidden bg-[#FFFCF5] border border-[#F0EDE8]">
                  <Image
                    src={item.image || "/placeholder-product.png"}
                    alt={item.name}
                    fill
                    className="object-contain p-1"
                    sizes="48px"
                  />
                </div>

                {/* Name + Price + Stepper */}
                <div className="flex-1 min-w-0 space-y-1.5">
                  <p className="text-sm font-medium truncate" style={{ color: "#1A1208" }}>
                    {item.name}
                  </p>
                  <Badge
                    variant="secondary"
                    className="text-[#FF2D78] bg-pink-50 border-0 font-bold tabular-nums text-xs px-2 py-0.5"
                  >
                    {item.currency} {(item.price / 100).toFixed(2)}
                  </Badge>

                  {/* Qty stepper */}
                  <div className="flex items-center gap-1.5 pt-0.5">
                    <Button
                      variant="outline"
                      size="icon"
                      className="h-7 w-7"
                      aria-label="Decrease quantity"
                      onClick={() => updateQty(item.skuId, item.qty - 1)}
                    >
                      <Minus size={12} />
                    </Button>
                    <span className="text-sm w-6 text-center font-bold tabular-nums" style={{ color: "#1A1208" }}>
                      {item.qty}
                    </span>
                    <Button
                      variant="outline"
                      size="icon"
                      className="h-7 w-7"
                      aria-label="Increase quantity"
                      onClick={() => updateQty(item.skuId, item.qty + 1)}
                    >
                      <Plus size={12} />
                    </Button>
                  </div>
                </div>

                {/* Line total + Remove */}
                <div className="flex flex-col items-end gap-2 shrink-0">
                  <span
                    className="text-sm font-bold tabular-nums"
                    style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}
                  >
                    {item.currency} {((item.price * item.qty) / 100).toFixed(2)}
                  </span>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 text-red-400 hover:text-red-600 hover:bg-red-50"
                    aria-label="Remove item"
                    onClick={() => removeItem(item.skuId)}
                  >
                    <Trash2 size={13} />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>

        {/* Order Summary — lg:col-span-1 */}
        <div className="lg:col-span-1">
          <Card className="sticky top-20 shadow-sm">
            <CardHeader className="pb-3">
              <CardTitle className="text-base" style={{ fontFamily: "var(--font-syne)" }}>
                Order Summary
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">Subtotal</span>
                <span className="font-semibold tabular-nums">
                  ₹ {(subtotal / 100).toFixed(2)}
                </span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">Shipping</span>
                <span className="font-semibold text-green-600">Free</span>
              </div>

              <Separator />

              <div className="flex justify-between font-bold text-base">
                <span>Total</span>
                <span className="text-primary tabular-nums" style={{ fontFamily: "var(--font-syne)" }}>
                  ₹ {(subtotal / 100).toFixed(2)}
                </span>
              </div>

              <Link
                href="/checkout"
                className={cn(buttonVariants({ size: "lg" }), "w-full bg-[#FF2D78] hover:bg-[#e02068] text-white font-bold mt-1 justify-center")}
              >
                Checkout
              </Link>

              <Link
                href="/products"
                className={cn(buttonVariants({ variant: "ghost" }), "w-full text-muted-foreground text-sm justify-center")}
              >
                Continue shopping
              </Link>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
