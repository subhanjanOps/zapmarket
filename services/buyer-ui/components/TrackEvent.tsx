"use client";
import { useEffect } from "react";

const ANALYTICS_URL =
  process.env.NEXT_PUBLIC_ANALYTICS_URL ?? "http://localhost:8095";

export function trackEvent(
  eventType: "view" | "cart_add" | "purchase" | "search",
  opts: { productId?: string; categoryId?: string; sessionId?: string; metadata?: Record<string, unknown> }
) {
  const sessionId = opts.sessionId ?? (typeof window !== "undefined" ? sessionStorage.getItem("zap_sid") ?? "" : "");
  const userId = typeof window !== "undefined" ? localStorage.getItem("zap_user_id") ?? undefined : undefined;
  fetch(`${ANALYTICS_URL}/v1/events`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      user_id: userId,
      session_id: sessionId,
      event_type: eventType,
      product_id: opts.productId,
      category_id: opts.categoryId,
      metadata: opts.metadata ?? {},
    }),
    keepalive: true,
  }).catch(() => {/* best-effort */});
}

interface TrackEventProps {
  eventType: "view" | "cart_add" | "purchase" | "search";
  productId?: string;
  categoryId?: string;
}

// TrackEvent fires a single analytics event on mount (fire-and-forget).
export default function TrackEvent({ eventType, productId, categoryId }: TrackEventProps) {
  useEffect(() => {
    // Ensure a session ID exists.
    if (!sessionStorage.getItem("zap_sid")) {
      sessionStorage.setItem("zap_sid", crypto.randomUUID());
    }
    trackEvent(eventType, { productId, categoryId });
  }, [eventType, productId, categoryId]);
  return null;
}
