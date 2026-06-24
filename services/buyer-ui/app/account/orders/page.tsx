import type { Metadata } from "next";
import { cookies } from "next/headers";
import Link from "next/link";

export const metadata: Metadata = { title: "My Orders" };
const GW = process.env.GATEWAY_URL ?? process.env.GATEWAY_URL ?? process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";

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
      <h1 className="text-2xl font-extrabold mb-6" style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}>My Orders</h1>
      {orders.length === 0 ? (
        <div className="text-center py-16" style={{ color: "#9CA3AF" }}>
          <p className="mb-4">You haven&apos;t placed any orders yet.</p>
          <Link href="/products" className="font-semibold hover:underline" style={{ color: "#FF2D78" }}>Start shopping →</Link>
        </div>
      ) : (
        <div className="space-y-3">
          {orders.map((o) => (
            <Link key={o.id as string} href={`/account/orders/${o.id}`}
              className="block rounded-2xl p-5 transition-shadow hover:shadow-md"
              style={{ background: "#fff", border: "1px solid #F0EDE8" }}>
              <div className="flex justify-between items-center">
                <div>
                  <p className="text-xs font-semibold uppercase tracking-wide" style={{ color: "#9CA3AF" }}>Order ID</p>
                  <p className="font-mono text-xs mt-0.5" style={{ color: "#1A1208" }}>{o.id as string}</p>
                </div>
                <span className={`text-xs font-semibold px-2.5 py-1 rounded-full ${STATUS_COLOR[o.status as string] ?? "bg-gray-100 text-gray-700"}`}>
                  {o.status as string}
                </span>
              </div>
              <p className="mt-2 text-sm font-semibold" style={{ color: "#FF2D78", fontVariantNumeric: "tabular-nums" }}>
                INR {((o.total_amount as number ?? 0) / 100).toFixed(2)}
              </p>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
