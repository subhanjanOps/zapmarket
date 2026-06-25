"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { motion } from "framer-motion";
import {
  Zap,
  Search,
  Heart,
  User,
  Menu,
  X,
  ChevronDown,
  Package,
  LogOut,
  Settings,
  LayoutGrid,
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
        .map(n => n[0])
        .join("")
        .slice(0, 2)
        .toUpperCase()
    : "?";

  return (
    <>
      <SearchModal open={searchOpen} onClose={() => setSearchOpen(false)} />

      <motion.header
        className={cn(
          "sticky top-0 z-40 w-full transition-all duration-300",
          scrolled
            ? "bg-white/95 backdrop-blur-md shadow-[0_1px_0_rgba(15,10,4,0.06),0_4px_24px_rgba(15,10,4,0.04)]"
            : "bg-white border-b border-[#EDE9E3]"
        )}
        initial={{ y: -4, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        transition={{ duration: 0.4, ease: [0.16, 1, 0.3, 1] }}
      >
        <div className="container-zap">
          {/* Primary row */}
          <div className="flex items-center gap-3 sm:gap-4 h-16">
            {/* Logo */}
            <Link
              href="/"
              className="flex items-center gap-2 shrink-0 group"
              aria-label="ZapMarket home"
            >
              <div className="h-8 w-8 bg-[#E91E8C] rounded-xl flex items-center justify-center shadow-[0_2px_8px_rgba(233,30,140,0.3)] group-hover:shadow-[0_4px_16px_rgba(233,30,140,0.4)] transition-shadow">
                <Zap className="h-4 w-4 text-white" strokeWidth={2.5} />
              </div>
              <span className="font-display font-bold text-lg text-[#0F0A04] tracking-tight hidden sm:block">
                Zap<span className="text-[#E91E8C]">Market</span>
              </span>
            </Link>

            {/* Categories pill — desktop */}
            <button className="hidden lg:flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-sm font-semibold text-[#3D2E1A] hover:bg-[#F9F8F5] transition-colors shrink-0">
              <LayoutGrid className="h-4 w-4 text-[#E91E8C]" />
              Categories
              <ChevronDown className="h-3.5 w-3.5 text-[#B8A898]" />
            </button>

            {/* Search — desktop */}
            <button
              onClick={() => setSearchOpen(true)}
              className="hidden sm:flex flex-1 items-center gap-3 h-10 px-4 bg-[#F9F8F5] hover:bg-[#F3F0EB] border-2 border-transparent hover:border-[#EDE9E3] rounded-xl text-sm text-[#B8A898] transition-all max-w-lg"
              aria-label="Open search"
            >
              <Search className="h-4 w-4 text-[#B8A898] shrink-0" />
              <span className="flex-1 text-left">Search products, brands...</span>
              <kbd className="hidden lg:inline-flex items-center gap-0.5 px-1.5 py-0.5 bg-[#EDE9E3] rounded text-xs text-[#7A6856] font-mono">
                ⌘K
              </kbd>
            </button>

            {/* Right actions */}
            <div className="flex items-center gap-0.5 ml-auto sm:ml-0">
              {/* Mobile search */}
              <button
                onClick={() => setSearchOpen(true)}
                className="sm:hidden p-2.5 rounded-xl hover:bg-[#F9F8F5] transition-colors"
                aria-label="Search"
              >
                <Search className="h-5 w-5 text-[#3D2E1A]" />
              </button>

              {/* Wishlist */}
              <Link
                href="/account/wishlist"
                className="hidden md:flex p-2.5 rounded-xl hover:bg-[#F9F8F5] transition-colors"
                aria-label="Wishlist"
              >
                <Heart className="h-5 w-5 text-[#3D2E1A]" />
              </Link>

              {/* Cart */}
              <CartButton />

              {/* Profile */}
              {user ? (
                <DropdownMenu>
                  <DropdownMenuTrigger>
                    <button className="flex items-center gap-2 pl-1.5 pr-2.5 py-1.5 rounded-xl hover:bg-[#F9F8F5] transition-colors ml-1">
                      <div className="h-7 w-7 bg-gradient-to-br from-[#E91E8C] to-[#FF5A35] rounded-lg flex items-center justify-center text-white text-xs font-bold shadow-sm">
                        {initials}
                      </div>
                      <ChevronDown className="h-3.5 w-3.5 text-[#B8A898] hidden sm:block" />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent
                    align="end"
                    className="w-52 mt-1 rounded-2xl border-[#EDE9E3] shadow-[0_8px_24px_rgba(15,10,4,0.12)] p-1.5"
                  >
                    <div className="px-3 py-2 mb-1">
                      <p className="font-semibold text-sm text-[#0F0A04] truncate">{user.name}</p>
                      <p className="text-xs text-[#7A6856] truncate">{user.email}</p>
                    </div>
                    <DropdownMenuSeparator className="bg-[#EDE9E3]" />
                    <DropdownMenuItem>
                      <Link
                        href="/account/orders"
                        className="flex items-center gap-2 rounded-xl px-3 py-2 text-sm text-[#3D2E1A] cursor-pointer"
                      >
                        <Package className="h-4 w-4 text-[#7A6856]" />
                        My Orders
                      </Link>
                    </DropdownMenuItem>
                    <DropdownMenuItem>
                      <Link
                        href="/account/settings"
                        className="flex items-center gap-2 rounded-xl px-3 py-2 text-sm text-[#3D2E1A] cursor-pointer"
                      >
                        <Settings className="h-4 w-4 text-[#7A6856]" />
                        Settings
                      </Link>
                    </DropdownMenuItem>
                    <DropdownMenuSeparator className="bg-[#EDE9E3]" />
                    <DropdownMenuItem
                      onClick={handleLogout}
                      className="flex items-center gap-2 rounded-xl px-3 py-2 text-sm text-red-600 cursor-pointer focus:bg-red-50"
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
                    className="hidden sm:block px-4 py-2 text-sm font-semibold text-[#3D2E1A] hover:text-[#E91E8C] transition-colors"
                  >
                    Sign in
                  </Link>
                  <Link
                    href="/register"
                    className="px-4 py-2 bg-[#E91E8C] hover:bg-[#B5166E] text-white text-sm font-semibold rounded-xl transition-colors shadow-[0_2px_8px_rgba(233,30,140,0.25)]"
                  >
                    Join free
                  </Link>
                </div>
              )}

              {/* Mobile hamburger */}
              <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
                <SheetTrigger>
                  <button
                    className="lg:hidden p-2.5 rounded-xl hover:bg-[#F9F8F5] transition-colors ml-0.5"
                    aria-label="Open menu"
                  >
                    <Menu className="h-5 w-5 text-[#3D2E1A]" />
                  </button>
                </SheetTrigger>
                <SheetContent side="left" className="w-72 p-0 border-r border-[#EDE9E3]">
                  <div className="flex flex-col h-full">
                    <div className="flex items-center justify-between px-5 py-4 border-b border-[#EDE9E3]">
                      <Link
                        href="/"
                        onClick={() => setMobileOpen(false)}
                        className="flex items-center gap-2"
                      >
                        <div className="h-7 w-7 bg-[#E91E8C] rounded-lg flex items-center justify-center">
                          <Zap className="h-4 w-4 text-white" strokeWidth={2.5} />
                        </div>
                        <span className="font-display font-bold text-[#0F0A04]">ZapMarket</span>
                      </Link>
                      <button
                        onClick={() => setMobileOpen(false)}
                        className="p-1.5 rounded-lg hover:bg-[#F9F8F5]"
                      >
                        <X className="h-4 w-4 text-[#7A6856]" />
                      </button>
                    </div>
                    <div className="flex-1 overflow-auto py-4 px-4">
                      <p className="text-xs font-bold tracking-widest text-[#B8A898] uppercase px-2 mb-2">
                        Browse
                      </p>
                      <nav className="space-y-0.5">
                        {CATEGORIES.map(cat => (
                          <Link
                            key={cat}
                            href={`/products?category=${encodeURIComponent(cat)}`}
                            onClick={() => setMobileOpen(false)}
                            className="flex items-center px-3 py-2.5 rounded-xl text-sm font-medium text-[#3D2E1A] hover:bg-[#F9F8F5] hover:text-[#E91E8C] transition-colors"
                          >
                            {cat}
                          </Link>
                        ))}
                      </nav>
                      <div className="border-t border-[#EDE9E3] mt-4 pt-4 space-y-0.5">
                        <Link
                          href="/account/orders"
                          onClick={() => setMobileOpen(false)}
                          className="flex items-center gap-2.5 px-3 py-2.5 rounded-xl text-sm font-medium text-[#3D2E1A] hover:bg-[#F9F8F5]"
                        >
                          <Package className="h-4 w-4 text-[#7A6856]" />
                          My Orders
                        </Link>
                        {!user && (
                          <Link
                            href="/login"
                            onClick={() => setMobileOpen(false)}
                            className="flex items-center gap-2.5 px-3 py-2.5 rounded-xl text-sm font-medium text-[#E91E8C] hover:bg-[#FDE8F4]"
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

          {/* Category strip — desktop */}
          <div className="hidden lg:flex items-center gap-0.5 h-10 border-t border-[#EDE9E3] overflow-x-auto scrollbar-none -mx-12 px-12">
            {CATEGORIES.map(cat => (
              <Link
                key={cat}
                href={`/products?category=${encodeURIComponent(cat)}`}
                className="shrink-0 px-3.5 py-1.5 text-xs font-semibold text-[#7A6856] hover:text-[#E91E8C] hover:bg-[#FDE8F4] rounded-lg transition-colors whitespace-nowrap"
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
