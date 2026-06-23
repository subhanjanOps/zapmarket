import type { Metadata } from "next";
import { cookies } from "next/headers";
import Link from "next/link";

export const metadata: Metadata = { title: "My Orders" };
const GW = process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";

const STATUS_COLOR: Record<string, string> = {
  PENDING: "bg-yellow-100 text-yellow-700",
  CONFIRMED: "bg-blue-100 text-blue-700",
  SHIPPED: "bg-purple-100 text-purple-700",
  DELIVERED: "bg-green-100 text-green-700",
  CANCELLED: "bg-red-100 text-red-700",
};

export default async function OrdersPage() {
  const jar = await cookies();
  const token = jar.get("buyer_token")?.value;

  let orders: Record<string, unknown>[] = [];
  if (token) {
    try {
      const r = await fetch(`${GW}/v1/orders`, {
        headers: { Authorization: `Bearer ${token}` },
        cache: "no-store",
      });
      if (r.ok) {
        const d = await r.json();
        orders = d.data ?? d ?? [];
      }
    } catch { /* show empty state */ }
  }

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <h1 className="text-2xl font-bold mb-6">My Orders</h1>
      {orders.length === 0 ? (
        <div className="text-center py-16 text-gray-500">
          <p className="mb-4">You haven&apos;t placed any orders yet.</p>
          <Link href="/products" className="text-[#FF9900] hover:underline">Start shopping →</Link>
        </div>
      ) : (
        <div className="space-y-4">
          {orders.map((o) => (
            <Link key={o.id as string} href={`/account/orders/${o.id}`}
              className="block bg-white border border-gray-200 rounded-lg p-5 hover:shadow-md transition-shadow">
              <div className="flex justify-between items-center">
                <div>
                  <p className="text-sm text-gray-500">Order ID</p>
                  <p className="font-mono text-xs text-gray-700">{o.id as string}</p>
                </div>
                <span className={`text-xs font-semibold px-2 py-1 rounded-full ${STATUS_COLOR[o.status as string] ?? "bg-gray-100 text-gray-700"}`}>
                  {o.status as string}
                </span>
              </div>
              <p className="mt-2 text-sm text-gray-600">
                Total: INR {((o.total_amount as number ?? 0) / 100).toFixed(2)}
              </p>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
