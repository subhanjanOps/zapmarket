import type { Metadata } from "next";
import { cookies } from "next/headers";
import { notFound } from "next/navigation";

export const metadata: Metadata = { title: "Order Detail" };
const GW = process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";

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
    try {
      const r = await fetch(`${GW}/v1/orders/${id}`, {
        headers: { Authorization: `Bearer ${token}` },
        cache: "no-store",
      });
      if (r.ok) order = await r.json();
    } catch { /* fall through to notFound */ }
  }

  if (!order) notFound();

  const items: Record<string, unknown>[] = (order.items as Record<string, unknown>[]) ?? [];

  return (
    <div className="max-w-3xl mx-auto px-4 py-8">
      {sp.new === "1" && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-4 mb-6 text-green-700 font-medium">
          ✓ Order placed successfully!
        </div>
      )}
      <h1 className="text-2xl font-bold mb-2">Order Detail</h1>
      <p className="font-mono text-xs text-gray-500 mb-6">{id}</p>

      <div className="bg-white border border-gray-200 rounded-lg p-6 mb-6">
        <div className="flex justify-between items-center mb-4">
          <span className="font-semibold">Status</span>
          <span className="text-sm font-medium">{order.status as string}</span>
        </div>
        <div className="space-y-3">
          {items.map((item, i) => (
            <div key={i} className="flex justify-between text-sm">
              <span>SKU: {item.sku_id as string} ×{item.quantity as number}</span>
              <span>INR {(((item.unit_price as number) * (item.quantity as number)) / 100).toFixed(2)}</span>
            </div>
          ))}
        </div>
        <div className="border-t mt-4 pt-4 flex justify-between font-bold">
          <span>Total</span>
          <span>INR {((order.total_amount as number ?? 0) / 100).toFixed(2)}</span>
        </div>
      </div>

      {order.status === "PENDING" && (
        <form action={`/api/proxy/v1/orders/${id}/cancel`} method="POST">
          <button type="submit" className="text-sm text-red-500 hover:underline border border-red-300 px-4 py-2 rounded">
            Cancel Order
          </button>
        </form>
      )}
    </div>
  );
}
