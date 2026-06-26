"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { motion } from "framer-motion";
import {
  Search,
  Heart,
  User,
  Menu,
  X,
  Package,
  LogOut,
  Settings,
} from "lucide-react";
import CartButton from "@/components/CartButton";
import { SearchModal } from "@/components/SearchModal";
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

const CATEGORIES = [
  "Electronics",
  "Fashion",
  "Home & Kitchen",
  "Sports & Fitness",
  "Beauty & Health",
  "Books",
  "Toys & Games",
  "Grocery",
  "Automotive",
  "Deals",
];

interface NavUser {
  name: string;
  email: string;
}

interface NavbarProps {
  user: NavUser | null;
}

export default function Navbar({ user }: NavbarProps) {
  const [scrolled, setScrolled] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const [activeCategory, setActiveCategory] = useState<string | null>(null);
  const router = useRouter();

  useEffect(() => {
    const handler = () => setScrolled(window.scrollY > 12);
    window.addEventListener("scroll", handler, { passive: true });
    return () => window.removeEventListener("scroll", handler);
  }, []);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        setSearchOpen(true);
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, []);

  const handleLogout = async () => {
    await fetch("/api/auth/logout", { method: "POST" });
    router.push("/login");
    router.refresh();
  };

  const initials = user?.name
    ? user.name
        .split(" ")
        .map((n) => n[0])
        .join("")
        .slice(0, 2)
        .toUpperCase()
    : "?";

  return (
    <>
      <SearchModal open={searchOpen} onClose={() => setSearchOpen(false)} />

      <motion.header
        className={cn(
          "sticky top-0 z-40 w-full bg-white transition-all duration-200",
          scrolled
            ? "border-b border-transparent shadow-[0_1px_8px_rgba(0,0,0,0.06)]"
            : "border-b border-[#E8E8E8]"
        )}
        initial={{ y: -4, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        transition={{ duration: 0.3, ease: [0.25, 0.1, 0.25, 1] }}
      >
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          {/* Primary row */}
          <div className="flex items-center gap-4 h-14">
            {/* Logo */}
            <Link
              href="/"
              className="shrink-0"
              aria-label="ZapMarket home"
            >
              <span className="font-bold text-xl tracking-tight text-[#111111]">
                Zap<span className="text-[#E91E8C]">Market</span>
              </span>
            </Link>

            {/* Search bar — desktop */}
            <button
              onClick={() => setSearchOpen(true)}
              className={cn(
                "hidden sm:flex flex-1 items-center gap-2 h-9 px-3 bg-[#F6F6F6] border border-[#E8E8E8] rounded-md text-sm text-[#999999] transition-colors duration-200 max-w-xl",
                "hover:border-[#D0D0D0] focus:border-[#111111] focus:outline-none"
              )}
              aria-label="Open search"
            >
              <Search className="h-4 w-4 text-[#999999] shrink-0" />
              <span className="flex-1 text-left">Search products, brands...</span>
              <kbd className="hidden lg:inline-flex items-center gap-0.5 px-1.5 py-0.5 bg-white border border-[#E8E8E8] rounded text-xs text-[#999999] font-mono">
                ⌘K
              </kbd>
            </button>

            {/* Right actions */}
            <div className="flex items-center gap-1 ml-auto sm:ml-0">
              {/* Mobile search */}
              <button
                onClick={() => setSearchOpen(true)}
                className="sm:hidden p-2 hover:text-[#111111] text-[#555555] transition-colors"
                aria-label="Search"
              >
                <Search className="h-5 w-5" />
              </button>

              {/* Wishlist */}
              <Link
                href="/account/wishlist"
                className="hidden md:flex p-2 text-[#555555] hover:text-[#111111] transition-colors"
                aria-label="Wishlist"
              >
                <Heart className="h-5 w-5" strokeWidth={1.75} />
              </Link>

              {/* Cart */}
              <CartButton />

              {/* Profile */}
              {user ? (
                <DropdownMenu>
                  <DropdownMenuTrigger
                    render={
                      <button
                        className="flex items-center justify-center h-7 w-7 rounded-full bg-[#F0F0F0] text-[#111111] text-xs font-semibold ml-1 hover:bg-[#E8E8E8] transition-colors"
                        aria-label="Account menu"
                      />
                    }
                  >
                    {initials}
                  </DropdownMenuTrigger>
                  <DropdownMenuContent
                    align="end"
                    className="w-52 mt-2 rounded-md border border-[#E8E8E8] shadow-xl p-1"
                  >
                    <div className="px-3 py-2 mb-1">
                      <p className="font-semibold text-sm text-[#111111] truncate">
                        {user.name}
                      </p>
                      <p className="text-xs text-[#999999] truncate">{user.email}</p>
                    </div>
                    <DropdownMenuSeparator className="bg-[#E8E8E8]" />
                    <DropdownMenuItem
                      render={<Link href="/account/orders" />}
                      className="flex items-center gap-2 px-3 py-2 text-sm text-[#555555] hover:text-[#111111] cursor-pointer rounded-md"
                    >
                      <Package className="h-4 w-4" />
                      My Orders
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      render={<Link href="/account/settings" />}
                      className="flex items-center gap-2 px-3 py-2 text-sm text-[#555555] hover:text-[#111111] cursor-pointer rounded-md"
                    >
                      <Settings className="h-4 w-4" />
                      Settings
                    </DropdownMenuItem>
                    <DropdownMenuSeparator className="bg-[#E8E8E8]" />
                    <DropdownMenuItem
                      onClick={handleLogout}
                      className="flex items-center gap-2 px-3 py-2 text-sm text-[#DC2626] cursor-pointer rounded-md focus:bg-red-50"
                    >
                      <LogOut className="h-4 w-4" />
                      Sign out
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              ) : (
                <div className="flex items-center gap-2 ml-1">
                  <Link
                    href="/login"
                    className="hidden sm:block text-sm text-[#555555] hover:text-[#111111] transition-colors"
                  >
                    Sign in
                  </Link>
                  <Link
                    href="/register"
                    className="h-8 px-4 inline-flex items-center bg-[#E91E8C] hover:bg-[#C2187A] text-white text-sm font-medium rounded-md transition-colors"
                  >
                    Register
                  </Link>
                </div>
              )}

              {/* Mobile hamburger */}
              <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
                <SheetTrigger
                  render={
                    <button
                      className="lg:hidden p-2 ml-0.5 text-[#555555] hover:text-[#111111] transition-colors"
                      aria-label="Open menu"
                    />
                  }
                >
                  <Menu className="h-5 w-5" />
                </SheetTrigger>
                <SheetContent side="left" className="w-72 p-0 bg-white border-r border-[#E8E8E8]">
                  <div className="flex flex-col h-full">
                    {/* Sheet header */}
                    <div className="flex items-center justify-between px-5 py-4 border-b border-[#E8E8E8]">
                      <Link
                        href="/"
                        onClick={() => setMobileOpen(false)}
                      >
                        <span className="font-bold text-lg tracking-tight text-[#111111]">
                          Zap<span className="text-[#E91E8C]">Market</span>
                        </span>
                      </Link>
                      <button
                        onClick={() => setMobileOpen(false)}
                        className="p-1.5 text-[#999999] hover:text-[#111111] transition-colors"
                      >
                        <X className="h-4 w-4" />
                      </button>
                    </div>

                    {/* Sheet body */}
                    <div className="flex-1 overflow-auto py-4 px-4">
                      <p className="text-[11px] font-semibold tracking-[0.06em] uppercase text-[#999999] px-3 mb-2">
                        Browse
                      </p>
                      <nav className="space-y-0.5">
                        {CATEGORIES.map((cat) => (
                          <Link
                            key={cat}
                            href={`/products?category=${encodeURIComponent(cat)}`}
                            onClick={() => setMobileOpen(false)}
                            className="flex items-center px-3 py-2.5 rounded-md text-sm font-medium text-[#555555] hover:text-[#111111] hover:bg-[#F6F6F6] transition-colors"
                          >
                            {cat}
                          </Link>
                        ))}
                      </nav>

                      <div className="border-t border-[#E8E8E8] mt-4 pt-4 space-y-0.5">
                        <Link
                          href="/account/orders"
                          onClick={() => setMobileOpen(false)}
                          className="flex items-center gap-2.5 px-3 py-2.5 rounded-md text-sm font-medium text-[#555555] hover:text-[#111111] hover:bg-[#F6F6F6] transition-colors"
                        >
                          <Package className="h-4 w-4" />
                          My Orders
                        </Link>
                        {!user && (
                          <Link
                            href="/login"
                            onClick={() => setMobileOpen(false)}
                            className="flex items-center gap-2.5 px-3 py-2.5 rounded-md text-sm font-medium text-[#555555] hover:text-[#111111] hover:bg-[#F6F6F6] transition-colors"
                          >
                            <User className="h-4 w-4" />
                            Sign in
                          </Link>
                        )}
                      </div>
                    </div>
                  </div>
                </SheetContent>
              </Sheet>
            </div>
          </div>

          {/* Category strip — desktop only */}
          <div className="hidden lg:flex items-center h-9 border-t border-[#E8E8E8] overflow-x-auto scrollbar-none gap-0.5">
            {CATEGORIES.map((cat) => (
              <Link
                key={cat}
                href={`/products?category=${encodeURIComponent(cat)}`}
                onClick={() => setActiveCategory(cat)}
                className={cn(
                  "shrink-0 px-3.5 h-full inline-flex items-center text-xs font-medium whitespace-nowrap transition-colors duration-150",
                  activeCategory === cat
                    ? "border-b-2 border-[#E91E8C] text-[#111111]"
                    : "text-[#555555] hover:text-[#111111] border-b-2 border-transparent"
                )}
              >
                {cat}
              </Link>
            ))}
          </div>
        </div>
      </motion.header>
    </>
  );
}
