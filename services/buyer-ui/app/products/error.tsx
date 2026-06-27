"use client";

import { useEffect } from "react";
import Link from "next/link";

export default function ProductsError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error("Products page error:", error);
  }, [error]);

  return (
    <div className="max-w-7xl mx-auto px-4 py-24 flex flex-col items-center gap-6 text-center">
      <h2 className="text-2xl font-semibold text-foreground">
        Could not load products
      </h2>
      <p className="text-muted-foreground max-w-sm">
        Something went wrong while fetching products. Please try again or browse
        the home page.
      </p>
      <div className="flex gap-3">
        <button
          onClick={reset}
          className="px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors"
        >
          Try again
        </button>
        <Link
          href="/"
          className="px-4 py-2 rounded-lg border border-border text-sm font-medium hover:bg-accent transition-colors"
        >
          Go home
        </Link>
      </div>
    </div>
  );
}
