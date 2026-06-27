import type { Metadata } from "next";
import { cookies } from "next/headers";
import { notFound } from "next/navigation";
import Link from "next/link";
import { Package, CheckCircle2 } from "lucide-react";
import { apiFetch, friendlyOrderId } from "@/lib/api";
import CancelOrderButton from "./CancelOrderButton";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";

export const metadata: Metadata = { title: "Order Detail" };

const STATUS_STEPS = ["PENDING", "CONFIRMED", "SHIPPED", "DELIVERED"];

const STATUS_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  PENDING:   "secondary",
  CONFIRMED: "default",
  SHIPPED:   "default",
  DELIVERED: "default",
  CANCELLED: "destructive",
};

const STATUS_CLASS: Record<string, string> = {
  PENDING:   "bg-amber-50 text-amber-700 border-amber-200",
  CONFIRMED: "bg-blue-50 text-blue-700 border-blue-200",
  SHIPPED:   "bg-violet-50 text-violet-700 border-violet-200",
  DELIVERED: "bg-emerald-50 text-emerald-700 border-emerald-200",
  CANCELLED: "bg-rose-50 text-rose-600 border-rose-200",
};

export default async function OrderDetailPage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>;
  searchParams: Promise<{ new?: string }>;
}) {
  const { id } = await params;
  const sp = await searchParams;
  const jar = await cookies();
  const token = jar.get("buyer_token")?.value;

  let order: Record<string, unknown> | null = null;
  if (token) {
    const env = await apiFetch<{ data?: Record<string, unknown> } & Record<string, unknown>>(
      `/v1/orders/${id}`,
      { headers: { Authorization: `Bearer ${token}` }, cache: "no-store" }
    );
    order = env?.data ?? (env as Record<string, unknown> | null);
  }

  if (!order) notFound();

  const items: Record<string, unknown>[] = (order.items as Record<string, unknown>[]) ?? [];
  const status = order.status as string;
  const isCancelled = status === "CANCELLED";
  const stepIndex = STATUS_STEPS.indexOf(status);
  const badgeClass = STATUS_CLASS[status] ?? "bg-stone-100 text-stone-600 border-stone-200";

  return (
    <div className="max-w-2xl mx-auto px-4 py-10 space-y-5">
      {/* Breadcrumb */}
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink render={<Link href="/account/orders" />}>
              My Orders
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage>{friendlyOrderId(id)}</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>

      {/* Success banner */}
      {sp.new === "1" && (
        <Card className="border-emerald-300 bg-emerald-50 animate-slide-up-sm">
          <CardContent className="flex items-center gap-3 py-4 px-5">
            <CheckCircle2 size={20} className="text-emerald-600 shrink-0" />
            <p className="text-sm font-semibold text-emerald-800">
              Order placed successfully!
            </p>
          </CardContent>
        </Card>
      )}

      {/* Header card — order ID + status badge + progress tracker */}
      <Card className="overflow-hidden">
        {/* Accent bar */}
        <div
          className="h-1.5"
          style={{
            background: isCancelled
              ? "#e0245f"
              : "linear-gradient(90deg, #E91E8C, #00736A)",
          }}
        />
        <CardHeader className="flex flex-row items-start justify-between gap-4 pb-2">
          <div className="space-y-0.5">
            <p className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
              Order
            </p>
            <CardTitle className="text-xl">{friendlyOrderId(id)}</CardTitle>
          </div>
          <Badge
            variant="outline"
            className={`text-xs font-bold shrink-0 mt-1 ${badgeClass}`}
          >
            {status}
          </Badge>
        </CardHeader>

        {/* Progress tracker */}
        {!isCancelled && stepIndex >= 0 && (
          <CardContent className="pt-2 pb-6">
            <div className="flex items-center">
              {STATUS_STEPS.map((step, i) => (
                <div
                  key={step}
                  className="flex items-center"
                  style={{ flex: i < STATUS_STEPS.length - 1 ? "1" : "none" }}
                >
                  <div className="flex flex-col items-center">
                    <div
                      className="w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold transition-all"
                      style={{
                        background: i <= stepIndex ? "#E91E8C" : "#EDE9E3",
                        color: i <= stepIndex ? "#fff" : "#B8A898",
                      }}
                    >
                      {i <= stepIndex ? <CheckCircle2 size={14} /> : i + 1}
                    </div>
                    <p
                      className="text-[9px] font-semibold mt-1 uppercase tracking-wider"
                      style={{ color: i <= stepIndex ? "#E91E8C" : "#B8A898" }}
                    >
                      {step}
                    </p>
                  </div>
                  {i < STATUS_STEPS.length - 1 && (
                    <div
                      className="flex-1 h-0.5 mx-1 mb-4 rounded-full transition-all"
                      style={{
                        background: i < stepIndex ? "#E91E8C" : "#EDE9E3",
                      }}
                    />
                  )}
                </div>
              ))}
            </div>
          </CardContent>
        )}
      </Card>

      {/* Items card */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="flex items-center gap-2 text-sm font-bold">
            <Package size={16} className="text-[#E91E8C]" />
            Items
          </CardTitle>
        </CardHeader>
        <CardContent className="px-6 pb-0">
          {items.map((item, i) => {
            const skuCode = (item.sku_code as string) ?? null;
            const productName = (item.product_name as string) ?? null;
            const qty = item.quantity as number;
            const unitPrice = item.unit_price as number;
            return (
              <div key={i}>
                {i > 0 && <Separator className="my-3" />}
                <div className="flex justify-between items-start gap-4 py-1">
                  <div className="min-w-0">
                    <p className="text-sm font-semibold leading-snug text-foreground">
                      {productName ?? "Product"}
                    </p>
                    {skuCode && (
                      <p className="text-xs mt-0.5 font-mono text-muted-foreground">
                        {skuCode}
                      </p>
                    )}
                    <p className="text-xs mt-0.5 text-muted-foreground">
                      Qty: {qty}
                    </p>
                  </div>
                  <p
                    className="text-sm font-bold shrink-0 tabular-nums"
                    style={{ color: "#1A1208" }}
                  >
                    ₹{((unitPrice * qty) / 100).toFixed(2)}
                  </p>
                </div>
              </div>
            );
          })}
        </CardContent>

        {/* Total row */}
        <div className="mx-6">
          <Separator className="mt-3" />
        </div>
        <CardContent className="flex justify-between items-center py-4 bg-amber-50/40 rounded-b-lg">
          <p className="font-bold text-foreground">Total</p>
          <p
            className="text-xl font-extrabold tabular-nums"
            style={{ color: "#E91E8C", fontFamily: "var(--font-syne)" }}
          >
            ₹{((order.total_amount as number ?? 0) / 100).toFixed(2)}
          </p>
        </CardContent>
      </Card>

      {/* Cancel button */}
      {status === "PENDING" && (
        <div className="flex justify-end">
          <CancelOrderButton orderId={id} />
        </div>
      )}
    </div>
  );
}
