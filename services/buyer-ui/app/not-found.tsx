import Link from "next/link";
import { Zap, Search } from "lucide-react";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export default function NotFound() {
  return (
    <div className="min-h-[70vh] flex items-center justify-center px-4 py-16" style={{ background: "#FFFCF5" }}>
      <div className="w-full max-w-md text-center space-y-6 animate-fade-up">
        {/* Large accent number */}
        <div className="flex items-center justify-center gap-2">
          <Zap size={40} fill="#FF2D78" stroke="#FF2D78" className="animate-float" />
          <span
            className="text-8xl font-extrabold leading-none"
            style={{ fontFamily: "var(--font-syne)", color: "#F0EDE8" }}
          >
            404
          </span>
        </div>

        <div>
          <h1 className="text-2xl font-extrabold mb-2" style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}>
            Page not found
          </h1>
          <p className="text-sm" style={{ color: "#9CA3AF" }}>
            The page you&apos;re looking for doesn&apos;t exist or has been moved.
          </p>
        </div>

        {/* Search bar */}
        <form action="/products" method="GET" className="flex items-center overflow-hidden rounded-full" style={{ border: "2px solid #F0EDE8", background: "#fff" }}>
          <input
            type="text"
            name="search"
            placeholder="Search for products…"
            className="flex-1 px-4 py-2.5 text-sm outline-none bg-transparent"
            style={{ color: "#1A1208" }}
          />
          <button
            type="submit"
            className="mr-1.5 p-2 rounded-full text-white transition-all hover:scale-105 cursor-pointer"
            style={{ background: "#FF2D78" }}
            aria-label="Search"
          >
            <Search size={15} />
          </button>
        </form>

        <div className="flex flex-col sm:flex-row items-center justify-center gap-3">
          <Link href="/" className={cn(buttonVariants({ size: "lg" }), "bg-[#FF2D78] hover:bg-[#e0245f] text-white")}>
            Go home
          </Link>
          <Link href="/products" className={buttonVariants({ variant: "outline", size: "lg" })}>
            Browse products
          </Link>
        </div>
      </div>
    </div>
  );
}
