import type { Metadata } from "next";
import { cookies } from "next/headers";
import Link from "next/link";
import { Package, ShoppingBag, ChevronRight } from "lucide-react";
import { apiFetch, friendlyOrderId } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { buttonVariants } from "@/components/ui/button";

export const metadata: Metadata = { title: "My Orders" };

type BadgeVariant = "secondary" | "default" | "destructive" | "outline";

const STATUS_BADGE: Record<string, { variant: BadgeVariant; className?: string }> = {
  PENDING:   { variant: "secondary" },
  CONFIRMED: { variant: "secondary", className: "bg-blue-100 text-blue-700 hover:bg-blue-100" },
  SHIPPED:   { variant: "secondary", className: "bg-purple-100 text-purple-700 hover:bg-purple-100" },
  DELIVERED: { variant: "default",   className: "bg-green-600 text-white hover:bg-green-600" },
  CANCELLED: { variant: "destructive" },
};

export default async function OrdersPage() {
  const jar = await cookies();
  const token = jar.get("buyer_token")?.value;

  let orders: Record<string, unknown>[] = [];
  if (token) {
    const res = await apiFetch<{ data?: Record<string, unknown>[] } | Record<string, unknown>[]>(
      "/v1/orders",
      { headers: { Authorization: `Bearer ${token}` }, cache: "no-store" }
    );
    if (res) {
      orders = Array.isArray(res) ? res : ((res as Record<string, unknown>).data as Record<string, unknown>[]) ?? [];
    }
  }

  return (
    <div className="max-w-3xl mx-auto px-4 py-10">
      {/* Header */}
      <div className="flex items-center gap-3 mb-8">
        <div className="w-10 h-10 rounded-2xl flex items-center justify-center bg-primary/10">
          <Package size={20} className="text-primary" />
        </div>
        <div>
          <h1 className="text-2xl font-extrabold leading-tight" style={{ fontFamily: "var(--font-syne)" }}>
            My Orders
          </h1>
          <p className="text-xs text-muted-foreground">
            {orders.length} order{orders.length !== 1 ? "s" : ""}
          </p>
        </div>
      </div>

      {orders.length === 0 ? (
        <Card className="border-dashed">
          <CardContent className="flex flex-col items-center py-20 text-center">
            <div className="w-16 h-16 rounded-full bg-muted flex items-center justify-center mb-4">
              <ShoppingBag size={28} className="text-muted-foreground" />
            </div>
            <p className="text-base font-semibold mb-1">No orders yet</p>
            <p className="text-sm text-muted-foreground mb-6">
              Looks like you haven&apos;t placed any orders.
            </p>
            <Link href="/products" className={buttonVariants()}>
              Start shopping
            </Link>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {orders.map((o) => {
            const badgeConfig = STATUS_BADGE[o.status as string] ?? { variant: "outline" as BadgeVariant };
            return (
              <Link key={o.id as string} href={`/account/orders/${o.id}`} className="block group">
                <Card className="transition-shadow duration-200 hover:shadow-md">
                  <CardHeader className="pb-0 pt-4 px-5">
                    <CardTitle className="text-xs font-bold uppercase tracking-wider text-muted-foreground">
                      Order
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="flex items-center justify-between px-5 pb-4 pt-1">
                    <div className="space-y-0.5">
                      <p className="text-sm font-bold text-primary font-mono">
                        {friendlyOrderId(o.id as string)}
                      </p>
                      <p className="text-base font-extrabold tabular-nums">
                        ₹{((o.total_amount as number ?? 0) / 100).toFixed(2)}
                      </p>
                    </div>
                    <div className="flex items-center gap-3">
                      <Badge
                        variant={badgeConfig.variant}
                        className={badgeConfig.className}
                      >
                        {o.status as string}
                      </Badge>
                      <ChevronRight
                        size={16}
                        className="text-muted-foreground transition-transform duration-150 group-hover:translate-x-0.5"
                      />
                    </div>
                  </CardContent>
                </Card>
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}
