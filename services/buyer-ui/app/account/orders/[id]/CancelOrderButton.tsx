"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { XCircle, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";

export default function CancelOrderButton({ orderId }: { orderId: string }) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const router = useRouter();

  async function handleCancel() {
    if (!window.confirm("Cancel this order? This cannot be undone.")) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/proxy/v1/orders/${orderId}/cancel`, { method: "POST" });
      if (!res.ok) {
        const d = await res.json().catch(() => ({}));
        setError((d as Record<string, string>).error ?? "Cancel failed. Please try again.");
        return;
      }
      router.refresh();
    } catch {
      setError("Network error. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-2">
      <Button
        variant="outline"
        size="sm"
        onClick={handleCancel}
        disabled={loading}
        className="flex items-center gap-2 border-rose-200 bg-rose-50 text-rose-600 hover:bg-rose-100 hover:text-rose-700"
      >
        {loading ? <Loader2 size={14} className="animate-spin" /> : <XCircle size={14} />}
        {loading ? "Cancelling…" : "Cancel Order"}
      </Button>
      {error && (
        <p className="text-xs rounded-lg px-3 py-2" style={{ background: "#FFF0F3", color: "#e0245f" }} role="alert">
          {error}
        </p>
      )}
    </div>
  );
}
