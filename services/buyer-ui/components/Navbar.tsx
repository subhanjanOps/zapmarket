"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Search, Zap, Package, LogOut, Menu } from "lucide-react";
import CartButton from "./CartButton";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Sheet,
  SheetContent,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Button, buttonVariants } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { cn } from "@/lib/utils";

export interface NavUser {
  name: string | null;
  email: string | null;
}

function getInitials(user: NavUser): string {
  const src = user.name ?? user.email ?? "";
  return src
    .split(/[\s@]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((s) => s[0].toUpperCase())
    .join("");
}

export default function Navbar({ user }: { user?: NavUser | null }) {
  const [scrolled, setScrolled] = useState(false);
  const [query, setQuery] = useState("");
  const router = useRouter();

  useEffect(() => {
    const handler = () => setScrolled(window.scrollY > 12);
    window.addEventListener("scroll", handler, { passive: true });
    return () => window.removeEventListener("scroll", handler);
  }, []);

  function handleSearch(e: React.FormEvent) {
    e.preventDefault();
    const q = query.trim();
    if (q) router.push(`/products?search=${encodeURIComponent(q)}`);
  }

  async function handleLogout() {
    await fetch("/api/auth/logout", { method: "POST" });
    router.push("/");
    router.refresh();
  }

  const displayName = user?.name ?? user?.email ?? null;
  const initials = user ? getInitials(user) : "";

  return (
    <header
      className={cn(
        "sticky top-0 z-50 transition-all duration-300",
        scrolled
          ? "bg-[rgba(255,252,245,0.88)] backdrop-blur-md border-b border-[rgba(240,237,232,0.8)] shadow-[0_2px_20px_rgba(26,18,8,0.06)]"
          : "bg-[#FFFCF5] border-b-[3px] border-[#FF2D78]"
      )}
    >
      <div className="max-w-7xl mx-auto px-4 py-3 flex items-center gap-4">
        {/* Logo */}
        <Link
          href="/"
          className="shrink-0 flex items-center gap-1 text-2xl font-extrabold tracking-tight leading-none group"
          style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}
        >
          <Zap
            size={22}
            fill="#FF2D78"
            stroke="#FF2D78"
            className="transition-transform duration-200 group-hover:scale-110 group-hover:rotate-12"
          />
          <span style={{ color: "#FF2D78" }}>Zap</span>Market
        </Link>

        {/* Search */}
        <form onSubmit={handleSearch} className="flex-1 flex items-center max-w-2xl">
          <div className="relative w-full flex items-center rounded-full overflow-hidden border-2 border-[#F0EDE8] bg-white shadow-[0_1px_4px_rgba(26,18,8,0.04)] focus-within:border-[#FF2D78] transition-colors duration-200">
            <Input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Find anything…"
              className="flex-1 border-0 rounded-full bg-transparent px-4 py-2.5 text-sm text-[#1A1208] placeholder:text-[#9e9287] focus-visible:ring-0 shadow-none"
            />
            <Button
              type="submit"
              size="icon"
              className="shrink-0 mr-1.5 rounded-full bg-[#00736A] hover:bg-[#005d54] text-white h-8 w-8"
              aria-label="Search"
            >
              <Search size={15} />
            </Button>
          </div>
        </form>

        {/* Desktop nav */}
        <nav className="hidden sm:flex items-center gap-3 shrink-0">
          <Link
            href="/account/orders"
            className={cn(
              buttonVariants({ variant: "ghost", size: "sm" }),
              "gap-1.5 text-[#1A1208] hover:text-[#FF2D78] hover:bg-transparent"
            )}
          >
            <Package size={16} />
            <span className="hidden md:inline">Orders</span>
          </Link>

          {displayName ? (
            <DropdownMenu>
              <DropdownMenuTrigger
                className="rounded-full p-0 h-9 w-9 inline-flex items-center justify-center hover:ring-2 hover:ring-[#FF2D78] hover:ring-offset-1 transition-all outline-none cursor-pointer"
                aria-label="Account menu"
              >
                <Avatar className="h-9 w-9">
                  <AvatarFallback className="bg-[#FF2D78] text-white text-xs font-semibold">
                    {initials}
                  </AvatarFallback>
                </Avatar>
              </DropdownMenuTrigger>
              <DropdownMenuContent
                align="end"
                className="w-44 rounded-2xl border border-[#F0EDE8] shadow-[0_8px_32px_rgba(26,18,8,0.1)] bg-white"
              >
                <div className="px-3 py-2 text-xs text-[#9e9287] truncate font-medium">
                  {displayName}
                </div>
                <DropdownMenuSeparator className="bg-[#F0EDE8]" />
                <DropdownMenuItem
                  className="gap-2.5 cursor-pointer text-[#1A1208]"
                  onClick={() => router.push("/account/orders")}
                >
                  <Package size={14} />
                  My Orders
                </DropdownMenuItem>
                <DropdownMenuSeparator className="bg-[#F0EDE8]" />
                <DropdownMenuItem
                  onClick={handleLogout}
                  className="gap-2.5 cursor-pointer text-[#e0245f]"
                >
                  <LogOut size={14} />
                  Sign Out
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          ) : (
            <Link
              href="/login"
              className={cn(
                buttonVariants({ variant: "ghost", size: "sm" }),
                "text-[#1A1208] hover:text-[#FF2D78] hover:bg-transparent"
              )}
            >
              Sign in
            </Link>
          )}

          <CartButton />
        </nav>

        {/* Mobile */}
        <div className="flex sm:hidden items-center gap-2 shrink-0 ml-auto">
          <CartButton />
          <Sheet>
            <SheetTrigger
              className={cn(
                buttonVariants({ variant: "ghost", size: "icon" }),
                "text-[#1A1208] hover:text-[#FF2D78] hover:bg-transparent"
              )}
              aria-label="Open menu"
            >
              <Menu size={22} />
            </SheetTrigger>
            <SheetContent side="right" className="w-72 bg-[#FFFCF5] border-l border-[#F0EDE8] p-0">
              <div className="flex flex-col h-full">
                <div className="flex items-center gap-2 px-5 py-5 border-b border-[#F0EDE8]">
                  <Zap size={18} fill="#FF2D78" stroke="#FF2D78" />
                  <span className="text-lg font-extrabold tracking-tight" style={{ fontFamily: "var(--font-syne)", color: "#1A1208" }}>
                    <span style={{ color: "#FF2D78" }}>Zap</span>Market
                  </span>
                </div>

                {user && displayName && (
                  <div className="flex items-center gap-3 px-5 py-4 border-b border-[#F0EDE8]">
                    <Avatar className="h-10 w-10">
                      <AvatarFallback className="bg-[#FF2D78] text-white text-sm font-semibold">
                        {initials}
                      </AvatarFallback>
                    </Avatar>
                    <div className="min-w-0">
                      <p className="text-sm font-semibold text-[#1A1208] truncate">{user.name ?? "Account"}</p>
                      {user.email && <p className="text-xs text-[#9e9287] truncate">{user.email}</p>}
                    </div>
                  </div>
                )}

                <nav className="flex flex-col gap-1 px-3 py-4 flex-1">
                  <Link
                    href="/account/orders"
                    className={cn(
                      buttonVariants({ variant: "ghost" }),
                      "justify-start gap-3 text-[#1A1208] hover:text-[#FF2D78] hover:bg-[#fff5f8] rounded-xl h-11"
                    )}
                  >
                    <Package size={18} />
                    My Orders
                  </Link>

                  {!user && (
                    <Link
                      href="/login"
                      className={cn(
                        buttonVariants(),
                        "mt-2 bg-[#FF2D78] hover:bg-[#e0245f] text-white rounded-xl h-11 justify-center"
                      )}
                    >
                      Sign in
                    </Link>
                  )}
                </nav>

                {user && (
                  <div className="px-3 py-4 border-t border-[#F0EDE8]">
                    <Button
                      variant="ghost"
                      onClick={handleLogout}
                      className="w-full justify-start gap-3 text-[#e0245f] hover:text-[#e0245f] hover:bg-red-50 rounded-xl h-11"
                    >
                      <LogOut size={18} />
                      Sign Out
                    </Button>
                  </div>
                )}
              </div>
            </SheetContent>
          </Sheet>
        </div>
      </div>
    </header>
  );
}
