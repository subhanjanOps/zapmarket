"use client";
import { useEffect } from "react";
import { AlertTriangle, RefreshCw } from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <div className="min-h-[60vh] flex items-center justify-center px-4 py-16" style={{ background: "#FFFCF5" }}>
      <div className="w-full max-w-sm text-center space-y-5 animate-fade-up">
        <div
          className="w-16 h-16 rounded-2xl flex items-center justify-center mx-auto"
          style={{ background: "#FFF0F3", border: "1px solid #FFD0DC" }}
        >
          <AlertTriangle size={28} style={{ color: "#e0245f" }} />
        </div>
        <div>
          <h2 className="text-xl font-extrabold mb-2" style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}>
            Something went wrong
          </h2>
          <p className="text-sm" style={{ color: "#9CA3AF" }}>
            An unexpected error occurred. Please try again.
          </p>
        </div>
        <div className="flex flex-col sm:flex-row items-center justify-center gap-3">
          <Button size="lg" onClick={reset}>
            <RefreshCw size={14} /> Try again
          </Button>
          <a href="/" className={cn(buttonVariants({ variant: "outline", size: "lg" }))}>
            Go home
          </a>
        </div>
      </div>
    </div>
  );
}
