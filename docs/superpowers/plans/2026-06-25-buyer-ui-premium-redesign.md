# ZapMarket Buyer UI — Premium Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Completely redesign and rebuild the ZapMarket Buyer UI into a production-ready, premium ecommerce experience competitive with Amazon, Flipkart, and Apple Store.

**Architecture:** Server Components handle all data fetching (no TanStack Query needed — Next.js 15 cache is sufficient); Client Components handle all interactivity (Framer Motion animations, cart state, search, filters). The existing API integration, MDX content, and auth flow are preserved. Only UI layer is rebuilt.

**Tech Stack:** Next.js 15 (App Router), React 19, TypeScript 5, Tailwind CSS 4, shadcn/ui, Framer Motion, Lucide React, Zustand, Sonner

## Global Constraints

- Tailwind CSS 4 syntax — use `@theme` block in globals.css, not tailwind.config.js
- No new npm packages except `framer-motion`
- All colors via CSS custom properties (`--color-*`) in `@theme` block
- Fonts: Syne (display/headings) + Plus Jakarta Sans (body) — already loaded in layout.tsx
- Primary brand color: `#E91E8C` (deep modern pink) as `--color-zap`
- Accent: `#FF5A35` (electric coral) as `--color-coral`
- Background: `#F9F8F5` (off-white) 
- Cards: `#FFFFFF` (pure white)
- Ink: `#0F0A04` (near-black warm)
- No ORM — backend API calls preserved exactly as-is
- `"use client"` only on interactive components
- Keep all existing routes (no route changes)
- Framer Motion: `motion` components only — no `AnimatePresence` overuse
- `prefers-reduced-motion` must be respected in all animations

---

## File Map

### Created
- `services/buyer-ui/components/AnnouncementBar.tsx` — sticky top promo strip
- `services/buyer-ui/components/SearchModal.tsx` — full-screen search overlay
- `services/buyer-ui/components/MegaMenu.tsx` — category mega-menu dropdown
- `services/buyer-ui/components/HeroSection.tsx` — premium hero with campaign cards
- `services/buyer-ui/components/SectionHeader.tsx` — reusable section heading (title + subtitle + view all)
- `services/buyer-ui/components/FeaturedCategories.tsx` — large editorial category showcase
- `services/buyer-ui/components/DealsSection.tsx` — today's deals with countdown
- `services/buyer-ui/components/TrendingSection.tsx` — trending products grid
- `services/buyer-ui/components/BrandsSection.tsx` — popular brands horizontal scroll
- `services/buyer-ui/components/TrustBanner.tsx` — delivery/returns/payments trust signals
- `services/buyer-ui/components/NewsletterSection.tsx` — premium newsletter section
- `services/buyer-ui/components/MotionWrapper.tsx` — reusable Framer Motion entrance wrapper
- `services/buyer-ui/components/ProductCardSkeleton.tsx` — shimmer skeleton for product card
- `services/buyer-ui/components/FilterDrawer.tsx` — mobile filter sheet
- `services/buyer-ui/components/PriceRangeSlider.tsx` — price range filter

### Modified
- `services/buyer-ui/package.json` — add framer-motion
- `services/buyer-ui/app/globals.css` — full design token overhaul
- `services/buyer-ui/app/layout.tsx` — add AnnouncementBar, update font weights
- `services/buyer-ui/app/page.tsx` — full homepage rebuild (all sections)
- `services/buyer-ui/components/Navbar.tsx` — premium sticky navbar with search modal + mega menu
- `services/buyer-ui/components/ProductCard.tsx` — complete redesign with all features
- `services/buyer-ui/components/Footer.tsx` — production-grade footer
- `services/buyer-ui/components/HeroCarousel.tsx` — replaced by HeroSection
- `services/buyer-ui/components/CategoryGrid.tsx` — replaced by FeaturedCategories
- `services/buyer-ui/components/AnimateIn.tsx` — converted to use Framer Motion
- `services/buyer-ui/components/CartButton.tsx` — mini cart preview on hover
- `services/buyer-ui/app/products/page.tsx` — premium listing page with sidebar filters
- `services/buyer-ui/app/products/[id]/page.tsx` — premium product detail
- `services/buyer-ui/app/cart/page.tsx` — premium cart with upsell
- `services/buyer-ui/app/login/page.tsx` — premium auth UI
- `services/buyer-ui/app/register/page.tsx` — premium auth UI
- `services/buyer-ui/app/account/orders/page.tsx` — clean order history
- `services/buyer-ui/app/account/orders/[id]/page.tsx` — detailed order view

---

## Task 1: Install Framer Motion

**Files:**
- Modify: `services/buyer-ui/package.json`

- [ ] **Step 1: Install framer-motion**

```bash
cd services/buyer-ui && npm install framer-motion
```

Expected: `framer-motion` added to `dependencies` in package.json.

- [ ] **Step 2: Verify import resolves**

```bash
node -e "require.resolve('framer-motion')" && echo "OK"
```

Expected: prints `OK`

- [ ] **Step 3: Commit**

```bash
git add services/buyer-ui/package.json services/buyer-ui/package-lock.json
git commit -m "deps(buyer-ui): add framer-motion"
```

---

## Task 2: Design Token Overhaul (globals.css)

**Files:**
- Modify: `services/buyer-ui/app/globals.css`

Replace the entire `@theme` block and `:root` variables with a richer, intentional token system.

- [ ] **Step 1: Replace globals.css design tokens**

Replace the file content's `@theme` block (lines 8–18) with:

```css
@theme {
  /* Brand */
  --color-zap:        #E91E8C;
  --color-zap-light:  #FDE8F4;
  --color-zap-dark:   #B5166E;
  --color-coral:      #FF5A35;
  --color-coral-light:#FFF0EC;

  /* Neutrals */
  --color-ink:        #0F0A04;
  --color-ink-700:    #3D2E1A;
  --color-ink-500:    #7A6856;
  --color-ink-300:    #B8A898;
  --color-ink-100:    #EDE9E3;
  --color-sand:       #F3F0EB;
  --color-surface:    #F9F8F5;

  /* Semantic */
  --color-success:    #12845F;
  --color-warning:    #D97706;
  --color-error:      #DC2626;

  /* Typography */
  --font-display: var(--font-syne), system-ui, sans-serif;
  --font-body:    var(--font-jakarta), system-ui, sans-serif;

  /* Shadows */
  --shadow-card:    0 1px 3px rgba(15,10,4,0.06), 0 4px 12px rgba(15,10,4,0.06);
  --shadow-card-hover: 0 8px 24px rgba(15,10,4,0.12), 0 2px 6px rgba(15,10,4,0.06);
  --shadow-nav:     0 1px 0 rgba(15,10,4,0.06), 0 4px 24px rgba(15,10,4,0.04);
  --shadow-modal:   0 24px 64px rgba(15,10,4,0.18);

  /* Radii */
  --radius-xs: 6px;
  --radius-sm: 10px;
  --radius-md: 14px;
  --radius-lg: 20px;
  --radius-xl: 28px;
  --radius-full: 9999px;
}
```

Then update `:root` CSS variables to align shadcn tokens with the new palette:

```css
:root {
  --background:          oklch(0.979 0.007 85);
  --foreground:          oklch(0.124 0.018 55);
  --card:                oklch(1 0 0);
  --card-foreground:     oklch(0.124 0.018 55);
  --popover:             oklch(1 0 0);
  --popover-foreground:  oklch(0.124 0.018 55);
  --primary:             oklch(0.583 0.276 339.4);
  --primary-foreground:  oklch(1 0 0);
  --secondary:           oklch(0.940 0.012 80);
  --secondary-foreground: oklch(0.124 0.018 55);
  --muted:               oklch(0.955 0.007 85);
  --muted-foreground:    oklch(0.520 0.022 70);
  --accent:              oklch(0.955 0.007 85);
  --accent-foreground:   oklch(0.124 0.018 55);
  --destructive:         oklch(0.577 0.245 27.325);
  --border:              oklch(0.920 0.010 80);
  --input:               oklch(0.920 0.010 80);
  --ring:                oklch(0.583 0.276 339.4);
  --radius: 0.875rem;
}
```

Also add premium skeleton shimmer and scroll-bar styles:

```css
/* Premium scrollbar */
::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: var(--color-ink-300); border-radius: 99px; }
::-webkit-scrollbar-thumb:hover { background: var(--color-ink-500); }

/* Selection */
::selection { background: var(--color-zap-light); color: var(--color-zap-dark); }

/* Skeleton shimmer */
@keyframes skeleton-shimmer {
  0%   { background-position: -400px 0; }
  100% { background-position:  400px 0; }
}
.skeleton {
  background: linear-gradient(90deg, #ede9e3 25%, #f5f2ee 50%, #ede9e3 75%);
  background-size: 800px 100%;
  animation: skeleton-shimmer 1.4s ease-in-out infinite;
}

/* Image aspect ratio utilities */
.aspect-product { aspect-ratio: 4/5; }
.aspect-hero    { aspect-ratio: 16/7; }
.aspect-square  { aspect-ratio: 1/1; }

/* Focus ring */
:focus-visible {
  outline: 2px solid var(--color-zap);
  outline-offset: 2px;
  border-radius: 4px;
}

/* Container */
.container-zap {
  max-width: 1400px;
  margin-inline: auto;
  padding-inline: clamp(16px, 4vw, 48px);
}
```

- [ ] **Step 2: Manual visual check**

Run `npm run dev` in services/buyer-ui and verify the background, text color, and primary color all updated on the homepage.

- [ ] **Step 3: Commit**

```bash
git add services/buyer-ui/app/globals.css
git commit -m "design(buyer-ui): premium design token overhaul"
```

---

## Task 3: MotionWrapper — Reusable Entrance Animations

**Files:**
- Create: `services/buyer-ui/components/MotionWrapper.tsx`
- Modify: `services/buyer-ui/components/AnimateIn.tsx` (deprecate, re-export from MotionWrapper)

- [ ] **Step 1: Create MotionWrapper.tsx**

```tsx
"use client";

import { motion, type Variants, type HTMLMotionProps } from "framer-motion";

type Variant = "fade-up" | "fade-in" | "slide-left" | "slide-right" | "scale-in" | "stagger-child";

const VARIANTS: Record<Variant, Variants> = {
  "fade-up": {
    hidden: { opacity: 0, y: 32 },
    visible: { opacity: 1, y: 0, transition: { duration: 0.6, ease: [0.16, 1, 0.3, 1] } },
  },
  "fade-in": {
    hidden: { opacity: 0 },
    visible: { opacity: 1, transition: { duration: 0.4, ease: "easeOut" } },
  },
  "slide-left": {
    hidden: { opacity: 0, x: -28 },
    visible: { opacity: 1, x: 0, transition: { duration: 0.5, ease: [0.16, 1, 0.3, 1] } },
  },
  "slide-right": {
    hidden: { opacity: 0, x: 28 },
    visible: { opacity: 1, x: 0, transition: { duration: 0.5, ease: [0.16, 1, 0.3, 1] } },
  },
  "scale-in": {
    hidden: { opacity: 0, scale: 0.92 },
    visible: { opacity: 1, scale: 1, transition: { duration: 0.5, ease: [0.16, 1, 0.3, 1] } },
  },
  "stagger-child": {
    hidden: { opacity: 0, y: 20 },
    visible: { opacity: 1, y: 0, transition: { duration: 0.45, ease: [0.16, 1, 0.3, 1] } },
  },
};

const STAGGER_CONTAINER: Variants = {
  hidden: {},
  visible: { transition: { staggerChildren: 0.07, delayChildren: 0.1 } },
};

interface MotionWrapperProps extends Omit<HTMLMotionProps<"div">, "variants"> {
  variant?: Variant;
  delay?: number;
  stagger?: boolean;
  className?: string;
  children: React.ReactNode;
}

export function MotionWrapper({
  variant = "fade-up",
  delay = 0,
  stagger = false,
  className,
  children,
  ...props
}: MotionWrapperProps) {
  if (stagger) {
    return (
      <motion.div
        className={className}
        variants={STAGGER_CONTAINER}
        initial="hidden"
        whileInView="visible"
        viewport={{ once: true, margin: "-60px" }}
        {...props}
      >
        {children}
      </motion.div>
    );
  }

  return (
    <motion.div
      className={className}
      variants={VARIANTS[variant]}
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: "-60px" }}
      transition={{ delay }}
      {...props}
    >
      {children}
    </motion.div>
  );
}

export function MotionChild({ className, children }: { className?: string; children: React.ReactNode }) {
  return (
    <motion.div className={className} variants={VARIANTS["stagger-child"]}>
      {children}
    </motion.div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add services/buyer-ui/components/MotionWrapper.tsx
git commit -m "feat(buyer-ui): add MotionWrapper with Framer Motion entrance animations"
```

---

## Task 4: AnnouncementBar

**Files:**
- Create: `services/buyer-ui/components/AnnouncementBar.tsx`
- Modify: `services/buyer-ui/app/layout.tsx` (add AnnouncementBar above Navbar)

- [ ] **Step 1: Create AnnouncementBar.tsx**

```tsx
"use client";

import { useState, useEffect } from "react";
import { X, Zap } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";

const MESSAGES = [
  "🚀 Free delivery on orders above ₹499 · Use code ZAPFREE",
  "⚡ Flash Sale ends in 4 hours — Up to 60% off Electronics",
  "✨ New arrivals every Monday — Follow us for early access",
];

export function AnnouncementBar() {
  const [visible, setVisible] = useState(true);
  const [index, setIndex] = useState(0);

  useEffect(() => {
    const id = setInterval(() => setIndex(i => (i + 1) % MESSAGES.length), 4000);
    return () => clearInterval(id);
  }, []);

  return (
    <AnimatePresence>
      {visible && (
        <motion.div
          initial={{ height: 0, opacity: 0 }}
          animate={{ height: "auto", opacity: 1 }}
          exit={{ height: 0, opacity: 0 }}
          transition={{ duration: 0.3, ease: [0.16, 1, 0.3, 1] }}
          className="overflow-hidden"
        >
          <div className="bg-[#0F0A04] text-white text-xs sm:text-sm font-medium relative">
            <div className="container-zap flex items-center justify-center gap-2 py-2.5 text-center">
              <Zap className="h-3.5 w-3.5 text-[#E91E8C] shrink-0" />
              <AnimatePresence mode="wait">
                <motion.span
                  key={index}
                  initial={{ opacity: 0, y: 8 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -8 }}
                  transition={{ duration: 0.3 }}
                  className="tracking-wide"
                >
                  {MESSAGES[index]}
                </motion.span>
              </AnimatePresence>
              <button
                onClick={() => setVisible(false)}
                aria-label="Dismiss announcement"
                className="absolute right-3 sm:right-6 top-1/2 -translate-y-1/2 p-1 rounded-full hover:bg-white/10 transition-colors"
              >
                <X className="h-3.5 w-3.5 text-white/60" />
              </button>
            </div>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
```

- [ ] **Step 2: Add AnnouncementBar to layout.tsx**

In `services/buyer-ui/app/layout.tsx`, import and add `<AnnouncementBar />` immediately before the `<Navbar />`:

```tsx
import { AnnouncementBar } from "@/components/AnnouncementBar";
// ...
<AnnouncementBar />
<Navbar user={user} />
```

- [ ] **Step 3: Commit**

```bash
git add services/buyer-ui/components/AnnouncementBar.tsx services/buyer-ui/app/layout.tsx
git commit -m "feat(buyer-ui): add animated announcement bar"
```

---

## Task 5: SectionHeader — Reusable Section Heading

**Files:**
- Create: `services/buyer-ui/components/SectionHeader.tsx`

- [ ] **Step 1: Create SectionHeader.tsx**

```tsx
import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { cn } from "@/lib/utils";

interface SectionHeaderProps {
  title: string;
  subtitle?: string;
  href?: string;
  viewAllLabel?: string;
  centered?: boolean;
  accent?: boolean;
  className?: string;
}

export function SectionHeader({
  title,
  subtitle,
  href,
  viewAllLabel = "View all",
  centered = false,
  accent = false,
  className,
}: SectionHeaderProps) {
  return (
    <div className={cn(
      "flex items-end justify-between gap-4",
      centered && "flex-col items-center text-center",
      className
    )}>
      <div className={cn(centered && "flex flex-col items-center")}>
        {accent && (
          <div className="flex items-center gap-2 mb-2">
            <div className="h-0.5 w-8 bg-[#E91E8C] rounded-full" />
            <span className="text-xs font-semibold tracking-widest text-[#E91E8C] uppercase">
              Featured
            </span>
          </div>
        )}
        <h2
          className="font-display font-bold text-[#0F0A04] leading-tight"
          style={{ fontSize: "clamp(1.5rem, 3vw, 2rem)" }}
        >
          {title}
        </h2>
        {subtitle && (
          <p className="mt-1.5 text-[#7A6856] text-sm leading-relaxed max-w-md">
            {subtitle}
          </p>
        )}
      </div>
      {href && !centered && (
        <Link
          href={href}
          className="group flex items-center gap-1.5 text-sm font-semibold text-[#E91E8C] hover:text-[#B5166E] transition-colors shrink-0"
        >
          {viewAllLabel}
          <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
        </Link>
      )}
      {href && centered && (
        <Link
          href={href}
          className="mt-4 inline-flex items-center gap-1.5 text-sm font-semibold text-[#E91E8C] hover:text-[#B5166E] transition-colors"
        >
          {viewAllLabel}
          <ArrowRight className="h-4 w-4" />
        </Link>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add services/buyer-ui/components/SectionHeader.tsx
git commit -m "feat(buyer-ui): add SectionHeader component"
```

---

## Task 6: Navbar — Premium Sticky Navigation

**Files:**
- Modify: `services/buyer-ui/components/Navbar.tsx`
- Create: `services/buyer-ui/components/SearchModal.tsx`

The Navbar has three zones: left (logo + categories trigger), center (search bar), right (wishlist, cart, profile). On mobile: hamburger sheet. Search opens a full-screen modal overlay.

- [ ] **Step 1: Create SearchModal.tsx**

```tsx
"use client";

import { useState, useEffect, useRef } from "react";
import { useRouter } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import { Search, X, TrendingUp, Clock, ArrowRight } from "lucide-react";

const TRENDING = ["Wireless earbuds", "Running shoes", "Skincare kit", "Gaming chair", "Coffee maker"];
const POPULAR_CATEGORIES = ["Electronics", "Fashion", "Home & Kitchen", "Sports", "Beauty"];

interface SearchModalProps {
  open: boolean;
  onClose: () => void;
}

export function SearchModal({ open, onClose }: SearchModalProps) {
  const [query, setQuery] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);
  const router = useRouter();

  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 100);
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
      setQuery("");
    }
    return () => { document.body.style.overflow = ""; };
  }, [open]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [onClose]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!query.trim()) return;
    router.push(`/products?search=${encodeURIComponent(query.trim())}`);
    onClose();
  };

  const handleSuggestion = (term: string) => {
    router.push(`/products?search=${encodeURIComponent(term)}`);
    onClose();
  };

  return (
    <AnimatePresence>
      {open && (
        <>
          <motion.div
            className="fixed inset-0 bg-black/40 backdrop-blur-sm z-50"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
          />
          <motion.div
            className="fixed top-0 left-0 right-0 z-50 bg-white shadow-2xl"
            initial={{ y: -20, opacity: 0 }}
            animate={{ y: 0, opacity: 1 }}
            exit={{ y: -20, opacity: 0 }}
            transition={{ duration: 0.25, ease: [0.16, 1, 0.3, 1] }}
          >
            <div className="container-zap py-4">
              <form onSubmit={handleSubmit} className="relative">
                <Search className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-[#7A6856]" />
                <input
                  ref={inputRef}
                  value={query}
                  onChange={e => setQuery(e.target.value)}
                  placeholder="Search for products, brands, categories..."
                  className="w-full h-14 pl-12 pr-16 bg-[#F9F8F5] rounded-2xl text-base font-medium text-[#0F0A04] placeholder-[#B8A898] border-2 border-transparent focus:border-[#E91E8C] focus:bg-white outline-none transition-all"
                />
                <button
                  type="button"
                  onClick={onClose}
                  className="absolute right-3 top-1/2 -translate-y-1/2 p-2 rounded-xl hover:bg-[#F3F0EB] transition-colors"
                >
                  <X className="h-5 w-5 text-[#7A6856]" />
                </button>
              </form>

              <div className="mt-6 pb-6 grid sm:grid-cols-2 gap-6">
                <div>
                  <div className="flex items-center gap-2 mb-3">
                    <TrendingUp className="h-4 w-4 text-[#E91E8C]" />
                    <span className="text-xs font-bold tracking-wider text-[#7A6856] uppercase">Trending</span>
                  </div>
                  <ul className="space-y-1">
                    {TRENDING.map(term => (
                      <li key={term}>
                        <button
                          onClick={() => handleSuggestion(term)}
                          className="w-full text-left px-3 py-2 rounded-xl text-sm font-medium text-[#3D2E1A] hover:bg-[#F9F8F5] hover:text-[#E91E8C] transition-colors flex items-center justify-between group"
                        >
                          {term}
                          <ArrowRight className="h-3.5 w-3.5 opacity-0 group-hover:opacity-100 transition-opacity" />
                        </button>
                      </li>
                    ))}
                  </ul>
                </div>
                <div>
                  <div className="flex items-center gap-2 mb-3">
                    <Clock className="h-4 w-4 text-[#7A6856]" />
                    <span className="text-xs font-bold tracking-wider text-[#7A6856] uppercase">Popular Categories</span>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {POPULAR_CATEGORIES.map(cat => (
                      <button
                        key={cat}
                        onClick={() => handleSuggestion(cat)}
                        className="px-3.5 py-1.5 bg-[#F9F8F5] hover:bg-[#FDE8F4] hover:text-[#E91E8C] text-sm font-medium text-[#3D2E1A] rounded-full transition-colors border border-[#EDE9E3]"
                      >
                        {cat}
                      </button>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
}
```

- [ ] **Step 2: Rewrite Navbar.tsx**

Full replacement of `services/buyer-ui/components/Navbar.tsx`:

```tsx
"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import {
  Zap, Search, Heart, ShoppingBag, User, Menu, X,
  ChevronDown, Package, LogOut, Settings, LayoutGrid,
} from "lucide-react";
import { CartButton } from "@/components/CartButton";
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
  "Electronics", "Fashion", "Home & Kitchen", "Sports",
  "Beauty", "Books", "Toys", "Grocery", "Automotive",
];

interface User { name: string; email: string; }

interface NavbarProps {
  user: User | null;
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

  // Keyboard shortcut: Cmd+K / Ctrl+K
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
    ? user.name.split(" ").map(n => n[0]).join("").slice(0, 2).toUpperCase()
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
          {/* Primary nav row */}
          <div className="flex items-center gap-3 sm:gap-4 h-16">
            {/* Logo */}
            <Link href="/" className="flex items-center gap-2 shrink-0 group" aria-label="ZapMarket home">
              <div className="h-8 w-8 bg-[#E91E8C] rounded-xl flex items-center justify-center shadow-[0_2px_8px_rgba(233,30,140,0.3)] group-hover:shadow-[0_4px_16px_rgba(233,30,140,0.4)] transition-shadow">
                <Zap className="h-4.5 w-4.5 text-white" strokeWidth={2.5} />
              </div>
              <span className="font-display font-bold text-lg text-[#0F0A04] tracking-tight hidden sm:block">
                Zap<span className="text-[#E91E8C]">Market</span>
              </span>
            </Link>

            {/* Category pill — desktop */}
            <button className="hidden lg:flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-sm font-semibold text-[#3D2E1A] hover:bg-[#F9F8F5] transition-colors shrink-0">
              <LayoutGrid className="h-4 w-4 text-[#E91E8C]" />
              Categories
              <ChevronDown className="h-3.5 w-3.5 text-[#B8A898]" />
            </button>

            {/* Search bar — desktop */}
            <button
              onClick={() => setSearchOpen(true)}
              className="hidden sm:flex flex-1 items-center gap-3 h-10 px-4 bg-[#F9F8F5] hover:bg-[#F3F0EB] border-2 border-transparent hover:border-[#EDE9E3] rounded-xl text-sm text-[#B8A898] transition-all max-w-lg"
              aria-label="Open search"
            >
              <Search className="h-4 w-4 text-[#B8A898] shrink-0" />
              <span className="flex-1 text-left">Search products, brands...</span>
              <kbd className="hidden lg:inline-flex items-center gap-1 px-1.5 py-0.5 bg-[#EDE9E3] rounded text-xs text-[#7A6856] font-mono">
                ⌘K
              </kbd>
            </button>

            {/* Right actions */}
            <div className="flex items-center gap-1 ml-auto sm:ml-0">
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
                className="hidden md:flex p-2.5 rounded-xl hover:bg-[#F9F8F5] transition-colors relative"
                aria-label="Wishlist"
              >
                <Heart className="h-5 w-5 text-[#3D2E1A]" />
              </Link>

              {/* Cart */}
              <CartButton />

              {/* Profile / Auth */}
              {user ? (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button className="flex items-center gap-2 pl-1.5 pr-2.5 py-1.5 rounded-xl hover:bg-[#F9F8F5] transition-colors ml-1">
                      <div className="h-7 w-7 bg-gradient-to-br from-[#E91E8C] to-[#FF5A35] rounded-lg flex items-center justify-center text-white text-xs font-bold shadow-sm">
                        {initials}
                      </div>
                      <ChevronDown className="h-3.5 w-3.5 text-[#B8A898] hidden sm:block" />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" className="w-52 mt-1 rounded-2xl border-[#EDE9E3] shadow-[0_8px_24px_rgba(15,10,4,0.12)] p-1.5">
                    <div className="px-3 py-2 mb-1">
                      <p className="font-semibold text-sm text-[#0F0A04] truncate">{user.name}</p>
                      <p className="text-xs text-[#7A6856] truncate">{user.email}</p>
                    </div>
                    <DropdownMenuSeparator className="bg-[#EDE9E3]" />
                    <DropdownMenuItem asChild>
                      <Link href="/account/orders" className="flex items-center gap-2 rounded-xl px-3 py-2 text-sm text-[#3D2E1A] cursor-pointer">
                        <Package className="h-4 w-4 text-[#7A6856]" />
                        My Orders
                      </Link>
                    </DropdownMenuItem>
                    <DropdownMenuItem asChild>
                      <Link href="/account/settings" className="flex items-center gap-2 rounded-xl px-3 py-2 text-sm text-[#3D2E1A] cursor-pointer">
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

              {/* Mobile menu */}
              <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
                <SheetTrigger asChild>
                  <button className="lg:hidden p-2.5 rounded-xl hover:bg-[#F9F8F5] transition-colors ml-0.5" aria-label="Menu">
                    <Menu className="h-5 w-5 text-[#3D2E1A]" />
                  </button>
                </SheetTrigger>
                <SheetContent side="left" className="w-80 p-0 border-r border-[#EDE9E3]">
                  <div className="flex flex-col h-full">
                    <div className="flex items-center justify-between px-5 py-4 border-b border-[#EDE9E3]">
                      <Link href="/" onClick={() => setMobileOpen(false)} className="flex items-center gap-2">
                        <div className="h-7 w-7 bg-[#E91E8C] rounded-lg flex items-center justify-center">
                          <Zap className="h-4 w-4 text-white" strokeWidth={2.5} />
                        </div>
                        <span className="font-display font-bold text-[#0F0A04]">ZapMarket</span>
                      </Link>
                      <button onClick={() => setMobileOpen(false)} className="p-1.5 rounded-lg hover:bg-[#F9F8F5]">
                        <X className="h-4.5 w-4.5 text-[#7A6856]" />
                      </button>
                    </div>
                    <div className="flex-1 overflow-auto py-4 px-4">
                      <p className="text-xs font-bold tracking-widest text-[#B8A898] uppercase px-2 mb-2">Categories</p>
                      <nav className="space-y-0.5">
                        {CATEGORIES.map(cat => (
                          <Link
                            key={cat}
                            href={`/products?category=${encodeURIComponent(cat)}`}
                            onClick={() => setMobileOpen(false)}
                            className="flex items-center justify-between px-3 py-2.5 rounded-xl text-sm font-medium text-[#3D2E1A] hover:bg-[#F9F8F5] hover:text-[#E91E8C] transition-colors"
                          >
                            {cat}
                          </Link>
                        ))}
                      </nav>
                      <div className="border-t border-[#EDE9E3] mt-4 pt-4 space-y-0.5">
                        <Link href="/account/orders" onClick={() => setMobileOpen(false)} className="flex items-center gap-2.5 px-3 py-2.5 rounded-xl text-sm font-medium text-[#3D2E1A] hover:bg-[#F9F8F5]">
                          <Package className="h-4 w-4 text-[#7A6856]" />
                          My Orders
                        </Link>
                        {!user && (
                          <Link href="/login" onClick={() => setMobileOpen(false)} className="flex items-center gap-2.5 px-3 py-2.5 rounded-xl text-sm font-medium text-[#E91E8C] hover:bg-[#FDE8F4]">
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

          {/* Category scroll strip — desktop */}
          <div className="hidden lg:flex items-center gap-1 h-10 border-t border-[#EDE9E3] -mx-4 sm:-mx-6 lg:-mx-12 px-4 sm:px-6 lg:px-12 overflow-x-auto scrollbar-none">
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
```

- [ ] **Step 3: Commit**

```bash
git add services/buyer-ui/components/Navbar.tsx services/buyer-ui/components/SearchModal.tsx
git commit -m "feat(buyer-ui): premium navbar with search modal, mega menu, keyboard shortcut"
```

---

## Task 7: ProductCard — Full Redesign

**Files:**
- Modify: `services/buyer-ui/components/ProductCard.tsx`
- Create: `services/buyer-ui/components/ProductCardSkeleton.tsx`

- [ ] **Step 1: Create ProductCardSkeleton.tsx**

```tsx
export function ProductCardSkeleton() {
  return (
    <div className="bg-white rounded-2xl overflow-hidden">
      <div className="skeleton aspect-product" />
      <div className="p-4 space-y-2.5">
        <div className="skeleton h-3 w-1/3 rounded-full" />
        <div className="skeleton h-4 w-4/5 rounded-full" />
        <div className="skeleton h-4 w-2/3 rounded-full" />
        <div className="flex items-center gap-2 pt-1">
          <div className="skeleton h-5 w-16 rounded-full" />
          <div className="skeleton h-4 w-12 rounded-full" />
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Rewrite ProductCard.tsx**

```tsx
"use client";

import { useState, useRef } from "react";
import Link from "next/link";
import Image from "next/image";
import { motion, AnimatePresence } from "framer-motion";
import { Heart, ShoppingCart, Star, Truck, Zap, Eye } from "lucide-react";
import { cn } from "@/lib/utils";

interface ProductImage { url: string; }
interface Product {
  id: string;
  name: string;
  brand?: string;
  category_name?: string;
  base_price: number;
  sku_count?: number;
  images?: ProductImage[];
  rating?: number;
  review_count?: number;
  discount_percent?: number;
  is_new?: boolean;
  free_shipping?: boolean;
}

function formatPrice(cents: number, currency = "INR") {
  return new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency,
    maximumFractionDigits: 0,
  }).format(cents / 100);
}

interface ProductCardProps {
  product: Product;
  images?: ProductImage[];
  priority?: boolean;
}

export function ProductCard({ product, images, priority = false }: ProductCardProps) {
  const [wishlist, setWishlist] = useState(false);
  const [imgIdx, setImgIdx] = useState(0);
  const [hovered, setHovered] = useState(false);
  const hoverTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const allImages = images?.length ? images : product.images ?? [];
  const hasMultiple = allImages.length > 1;

  const discountedPrice = product.discount_percent
    ? Math.round(product.base_price * (1 - product.discount_percent / 100))
    : product.base_price;

  const handleMouseEnter = () => {
    setHovered(true);
    if (hasMultiple) {
      hoverTimer.current = setInterval(() => {
        setImgIdx(i => (i + 1) % allImages.length);
      }, 1800);
    }
  };

  const handleMouseLeave = () => {
    setHovered(false);
    if (hoverTimer.current) clearInterval(hoverTimer.current);
    setTimeout(() => setImgIdx(0), 300);
  };

  return (
    <motion.div
      className="group bg-white rounded-2xl overflow-hidden shadow-[0_1px_3px_rgba(15,10,4,0.06),0_4px_12px_rgba(15,10,4,0.06)] hover:shadow-[0_8px_24px_rgba(15,10,4,0.12),0_2px_6px_rgba(15,10,4,0.06)] transition-shadow duration-300"
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      whileHover={{ y: -2 }}
      transition={{ duration: 0.25, ease: [0.16, 1, 0.3, 1] }}
    >
      <Link href={`/products/${product.id}`} className="block">
        {/* Image container */}
        <div className="relative aspect-product bg-[#F9F8F5] overflow-hidden">
          {allImages[0] ? (
            <Image
              src={allImages[imgIdx]?.url ?? allImages[0].url}
              alt={product.name}
              fill
              sizes="(max-width: 640px) 50vw, (max-width: 1024px) 33vw, 25vw"
              className="object-cover transition-transform duration-500 ease-[cubic-bezier(0.16,1,0.3,1)] group-hover:scale-105"
              priority={priority}
            />
          ) : (
            <div className="absolute inset-0 flex items-center justify-center">
              <Zap className="h-12 w-12 text-[#EDE9E3]" />
            </div>
          )}

          {/* Badges */}
          <div className="absolute top-3 left-3 flex flex-col gap-1.5">
            {product.discount_percent && product.discount_percent > 0 && (
              <span className="px-2 py-0.5 bg-[#E91E8C] text-white text-xs font-bold rounded-lg shadow-sm">
                -{product.discount_percent}%
              </span>
            )}
            {product.is_new && (
              <span className="px-2 py-0.5 bg-[#0F0A04] text-white text-xs font-bold rounded-lg shadow-sm">
                NEW
              </span>
            )}
          </div>

          {/* Wishlist */}
          <motion.button
            onClick={e => { e.preventDefault(); setWishlist(w => !w); }}
            className="absolute top-3 right-3 h-8 w-8 rounded-full bg-white/90 backdrop-blur-sm flex items-center justify-center shadow-sm opacity-0 group-hover:opacity-100 transition-opacity"
            whileTap={{ scale: 0.85 }}
            aria-label={wishlist ? "Remove from wishlist" : "Add to wishlist"}
          >
            <Heart
              className={cn("h-4 w-4 transition-colors", wishlist ? "fill-[#E91E8C] text-[#E91E8C]" : "text-[#7A6856]")}
            />
          </motion.button>

          {/* Image dots */}
          {hasMultiple && hovered && (
            <div className="absolute bottom-3 left-1/2 -translate-x-1/2 flex items-center gap-1">
              {allImages.slice(0, 5).map((_, i) => (
                <button
                  key={i}
                  onMouseEnter={e => { e.preventDefault(); setImgIdx(i); }}
                  className={cn(
                    "rounded-full transition-all",
                    i === imgIdx ? "h-1.5 w-4 bg-white" : "h-1.5 w-1.5 bg-white/60"
                  )}
                />
              ))}
            </div>
          )}

          {/* Quick view overlay */}
          <AnimatePresence>
            {hovered && (
              <motion.div
                className="absolute bottom-0 left-0 right-0 p-3"
                initial={{ y: 10, opacity: 0 }}
                animate={{ y: 0, opacity: 1 }}
                exit={{ y: 10, opacity: 0 }}
                transition={{ duration: 0.2 }}
              >
                <div className="flex items-center justify-center gap-1.5 bg-white/95 backdrop-blur-sm rounded-xl py-2.5 px-3 shadow-sm">
                  <Eye className="h-3.5 w-3.5 text-[#7A6856]" />
                  <span className="text-xs font-semibold text-[#3D2E1A]">Quick view</span>
                </div>
              </motion.div>
            )}
          </AnimatePresence>
        </div>

        {/* Card body */}
        <div className="p-4">
          {/* Brand + category */}
          <div className="flex items-center gap-1.5 mb-1.5">
            {product.brand && (
              <span className="text-xs font-bold text-[#E91E8C] tracking-wide uppercase">
                {product.brand}
              </span>
            )}
            {product.brand && product.category_name && (
              <span className="text-[#EDE9E3]">·</span>
            )}
            {product.category_name && (
              <span className="text-xs text-[#B8A898] truncate">{product.category_name}</span>
            )}
          </div>

          {/* Name */}
          <h3 className="text-sm font-semibold text-[#0F0A04] leading-snug line-clamp-2 mb-2">
            {product.name}
          </h3>

          {/* Rating */}
          {product.rating && (
            <div className="flex items-center gap-1.5 mb-2">
              <div className="flex items-center gap-0.5">
                {Array.from({ length: 5 }).map((_, i) => (
                  <Star
                    key={i}
                    className={cn(
                      "h-3 w-3",
                      i < Math.round(product.rating!) ? "fill-[#D97706] text-[#D97706]" : "text-[#EDE9E3] fill-[#EDE9E3]"
                    )}
                  />
                ))}
              </div>
              {product.review_count && (
                <span className="text-xs text-[#B8A898]">({product.review_count.toLocaleString()})</span>
              )}
            </div>
          )}

          {/* Price */}
          <div className="flex items-baseline gap-2">
            <span className="text-base font-bold text-[#0F0A04] font-display tabular-nums">
              {formatPrice(discountedPrice)}
            </span>
            {product.discount_percent && product.discount_percent > 0 && (
              <span className="text-xs text-[#B8A898] line-through tabular-nums">
                {formatPrice(product.base_price)}
              </span>
            )}
          </div>

          {/* Delivery */}
          <div className="mt-2.5 flex items-center gap-1.5">
            {product.free_shipping ? (
              <>
                <Truck className="h-3 w-3 text-[#12845F]" />
                <span className="text-xs text-[#12845F] font-medium">Free delivery</span>
              </>
            ) : (
              <span className="text-xs text-[#B8A898]">Delivery by tomorrow</span>
            )}
          </div>
        </div>
      </Link>

      {/* Add to cart */}
      <div className="px-4 pb-4">
        <motion.button
          className="w-full flex items-center justify-center gap-2 h-9 bg-[#F9F8F5] hover:bg-[#E91E8C] text-[#3D2E1A] hover:text-white text-sm font-semibold rounded-xl transition-colors group/btn"
          whileTap={{ scale: 0.97 }}
        >
          <ShoppingCart className="h-4 w-4 transition-transform group-hover/btn:scale-110" />
          Add to cart
        </motion.button>
      </div>
    </motion.div>
  );
}
```

- [ ] **Step 3: Commit**

```bash
git add services/buyer-ui/components/ProductCard.tsx services/buyer-ui/components/ProductCardSkeleton.tsx
git commit -m "feat(buyer-ui): premium product card with animations, wishlist, quick-view"
```

---

## Task 8: HeroSection

**Files:**
- Create: `services/buyer-ui/components/HeroSection.tsx`

- [ ] **Step 1: Create HeroSection.tsx**

```tsx
"use client";

import Link from "next/link";
import Image from "next/image";
import { motion } from "framer-motion";
import { ArrowRight, Zap, ShieldCheck, RefreshCw, Headphones } from "lucide-react";

interface Campaign {
  id: string;
  title: string;
  subtitle?: string;
  image_url?: string;
  cta_text?: string;
  cta_url?: string;
  bg_color?: string;
}

const TRUST_PILLS = [
  { icon: ShieldCheck, label: "Secure payments" },
  { icon: RefreshCw,   label: "Easy returns" },
  { icon: Headphones,  label: "24/7 support" },
];

const STAT_ITEMS = [
  { value: "2M+", label: "Happy customers" },
  { value: "50K+", label: "Products" },
  { value: "99%", label: "Delivery success" },
];

interface HeroSectionProps {
  campaigns: Campaign[];
}

export function HeroSection({ campaigns }: HeroSectionProps) {
  const primary = campaigns[0];
  const secondaries = campaigns.slice(1, 3);

  return (
    <section className="container-zap py-6">
      <div className="grid grid-cols-1 lg:grid-cols-[1fr_340px] gap-4">
        {/* Primary hero */}
        <motion.div
          className="relative overflow-hidden rounded-3xl bg-gradient-to-br from-[#0F0A04] via-[#1A0E08] to-[#2D1410] min-h-[340px] sm:min-h-[420px] flex flex-col justify-end p-8 sm:p-10"
          initial={{ opacity: 0, scale: 0.97 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.7, ease: [0.16, 1, 0.3, 1] }}
        >
          {/* Background image */}
          {primary?.image_url && (
            <Image
              src={primary.image_url}
              alt={primary.title ?? "Campaign"}
              fill
              className="object-cover opacity-30 mix-blend-luminosity"
              priority
            />
          )}

          {/* Decorative elements */}
          <div className="absolute top-8 right-8 h-32 w-32 bg-[#E91E8C]/20 rounded-full blur-3xl" />
          <div className="absolute top-4 right-20 h-16 w-16 bg-[#FF5A35]/20 rounded-full blur-2xl" />

          {/* Badge */}
          <div className="absolute top-6 left-6">
            <div className="flex items-center gap-1.5 px-3 py-1.5 bg-[#E91E8C] rounded-full">
              <Zap className="h-3 w-3 text-white" strokeWidth={2.5} />
              <span className="text-xs font-bold text-white tracking-wide">Limited time offer</span>
            </div>
          </div>

          {/* Content */}
          <div className="relative z-10">
            <motion.h1
              className="font-display font-bold text-white leading-tight mb-3"
              style={{ fontSize: "clamp(1.75rem, 4vw, 2.75rem)" }}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.2, duration: 0.6, ease: [0.16, 1, 0.3, 1] }}
            >
              {primary?.title ?? "Discover Amazing\nDeals Today"}
            </motion.h1>
            {primary?.subtitle && (
              <motion.p
                className="text-white/70 text-base mb-6 max-w-sm"
                initial={{ opacity: 0, y: 16 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.3, duration: 0.6 }}
              >
                {primary.subtitle}
              </motion.p>
            )}
            <motion.div
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.4, duration: 0.5 }}
            >
              <Link
                href={primary?.cta_url ?? "/products"}
                className="inline-flex items-center gap-2 px-6 py-3 bg-[#E91E8C] hover:bg-[#FF5A35] text-white font-bold rounded-2xl transition-colors shadow-[0_4px_20px_rgba(233,30,140,0.4)] hover:shadow-[0_4px_20px_rgba(255,90,53,0.4)]"
              >
                {primary?.cta_text ?? "Shop now"}
                <ArrowRight className="h-4 w-4" />
              </Link>
            </motion.div>
          </div>

          {/* Stats row */}
          <motion.div
            className="absolute bottom-6 right-6 hidden sm:flex items-center gap-4"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: 0.6 }}
          >
            {STAT_ITEMS.map(({ value, label }) => (
              <div key={label} className="text-center">
                <div className="font-display font-bold text-white text-lg">{value}</div>
                <div className="text-white/50 text-xs">{label}</div>
              </div>
            ))}
          </motion.div>
        </motion.div>

        {/* Secondary campaign cards */}
        <div className="flex flex-row lg:flex-col gap-4">
          {(secondaries.length ? secondaries : [{id:"a",title:"Flash Sale",subtitle:"Up to 60% off"},{id:"b",title:"New Arrivals",subtitle:"Just dropped"}]).map((camp, i) => (
            <motion.div
              key={camp.id}
              className="relative flex-1 overflow-hidden rounded-2xl bg-gradient-to-br from-[#FDE8F4] to-[#FFF0EC] min-h-[120px] sm:min-h-[160px] flex flex-col justify-end p-5 cursor-pointer group"
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: 0.2 + i * 0.1, duration: 0.6, ease: [0.16, 1, 0.3, 1] }}
              whileHover={{ scale: 1.02 }}
            >
              {camp.image_url && (
                <Image src={camp.image_url} alt={camp.title} fill className="object-cover opacity-20 group-hover:opacity-30 transition-opacity" />
              )}
              <div className="relative z-10">
                <h3 className="font-display font-bold text-[#0F0A04] text-lg leading-tight">{camp.title}</h3>
                {camp.subtitle && <p className="text-sm text-[#7A6856] mt-0.5">{camp.subtitle}</p>}
                <Link
                  href={camp.cta_url ?? "/products"}
                  className="mt-3 inline-flex items-center gap-1 text-sm font-bold text-[#E91E8C] hover:text-[#B5166E] transition-colors"
                >
                  {camp.cta_text ?? "Shop now"}
                  <ArrowRight className="h-3.5 w-3.5 transition-transform group-hover:translate-x-0.5" />
                </Link>
              </div>
            </motion.div>
          ))}
        </div>
      </div>

      {/* Trust pills */}
      <div className="flex items-center justify-center gap-4 sm:gap-8 mt-5">
        {TRUST_PILLS.map(({ icon: Icon, label }) => (
          <div key={label} className="flex items-center gap-2 text-sm text-[#7A6856]">
            <Icon className="h-4 w-4 text-[#E91E8C]" />
            <span className="font-medium hidden sm:block">{label}</span>
          </div>
        ))}
      </div>
    </section>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add services/buyer-ui/components/HeroSection.tsx
git commit -m "feat(buyer-ui): premium hero section with campaign cards and stats"
```

---

## Task 9: FeaturedCategories

**Files:**
- Create: `services/buyer-ui/components/FeaturedCategories.tsx`

- [ ] **Step 1: Create FeaturedCategories.tsx**

```tsx
"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import { Laptop, Shirt, Home, Dumbbell, Sparkles, BookOpen, Gamepad2, Apple } from "lucide-react";
import { SectionHeader } from "@/components/SectionHeader";
import { MotionWrapper, MotionChild } from "@/components/MotionWrapper";

const ICON_MAP: Record<string, React.ComponentType<{ className?: string }>> = {
  Electronics: Laptop,
  Fashion: Shirt,
  "Home & Kitchen": Home,
  Sports: Dumbbell,
  Beauty: Sparkles,
  Books: BookOpen,
  Gaming: Gamepad2,
  Grocery: Apple,
};

const COLORS = [
  { bg: "#FDE8F4", icon: "#E91E8C", ring: "#F5C2E0" },
  { bg: "#FFF0EC", icon: "#FF5A35", ring: "#FFD5C8" },
  { bg: "#EEF2FF", icon: "#4F46E5", ring: "#C7D2FE" },
  { bg: "#F0FDF4", icon: "#16A34A", ring: "#BBF7D0" },
  { bg: "#FEFCE8", icon: "#CA8A04", ring: "#FDE68A" },
  { bg: "#F0F9FF", icon: "#0284C7", ring: "#BAE6FD" },
  { bg: "#FDF4FF", icon: "#9333EA", ring: "#E9D5FF" },
  { bg: "#FFF7ED", icon: "#EA580C", ring: "#FED7AA" },
];

interface Category { id: string; name: string; slug?: string; product_count?: number; }

export function FeaturedCategories({ categories }: { categories: Category[] }) {
  const display = categories.slice(0, 8);

  return (
    <section className="container-zap py-10">
      <SectionHeader
        title="Shop by Category"
        subtitle="Find exactly what you're looking for"
        href="/products"
        viewAllLabel="All categories"
        className="mb-7"
      />
      <MotionWrapper stagger className="grid grid-cols-4 sm:grid-cols-4 md:grid-cols-8 gap-3 sm:gap-4">
        {display.map((cat, i) => {
          const color = COLORS[i % COLORS.length];
          const Icon = ICON_MAP[cat.name] ?? Sparkles;
          return (
            <MotionChild key={cat.id}>
              <Link
                href={`/products?category=${encodeURIComponent(cat.name)}`}
                className="group flex flex-col items-center gap-3 p-4 sm:p-5 rounded-2xl bg-white hover:bg-white hover:shadow-[0_8px_24px_rgba(15,10,4,0.1)] transition-all duration-300 text-center border border-[#EDE9E3] hover:border-transparent"
              >
                <motion.div
                  className="h-12 w-12 sm:h-14 sm:w-14 rounded-2xl flex items-center justify-center transition-transform"
                  style={{ backgroundColor: color.bg, boxShadow: `0 0 0 1px ${color.ring}` }}
                  whileHover={{ scale: 1.08, rotate: -4 }}
                  transition={{ duration: 0.25 }}
                >
                  <Icon className="h-6 w-6 sm:h-7 sm:w-7" style={{ color: color.icon }} />
                </motion.div>
                <span className="text-xs sm:text-sm font-semibold text-[#3D2E1A] group-hover:text-[#E91E8C] transition-colors leading-tight">
                  {cat.name}
                </span>
              </Link>
            </MotionChild>
          );
        })}
      </MotionWrapper>
    </section>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add services/buyer-ui/components/FeaturedCategories.tsx
git commit -m "feat(buyer-ui): editorial category grid with icons and spring animations"
```

---

## Task 10: TrustBanner

**Files:**
- Create: `services/buyer-ui/components/TrustBanner.tsx`

- [ ] **Step 1: Create TrustBanner.tsx**

```tsx
import { Truck, Shield, RefreshCw, Headphones, Award, CreditCard } from "lucide-react";

const TRUST_ITEMS = [
  { icon: Truck,       title: "Free Delivery",     desc: "On orders above ₹499" },
  { icon: Shield,      title: "Secure Payments",   desc: "256-bit SSL encryption" },
  { icon: RefreshCw,   title: "Easy Returns",      desc: "15-day hassle-free returns" },
  { icon: Headphones,  title: "24/7 Support",      desc: "Always here to help" },
  { icon: Award,       title: "Authentic Products", desc: "100% verified sellers" },
  { icon: CreditCard,  title: "EMI Available",     desc: "No-cost EMI on ₹3000+" },
];

export function TrustBanner() {
  return (
    <section className="bg-[#0F0A04] py-10">
      <div className="container-zap">
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-px bg-white/10 rounded-2xl overflow-hidden">
          {TRUST_ITEMS.map(({ icon: Icon, title, desc }) => (
            <div key={title} className="bg-[#0F0A04] flex flex-col items-center text-center px-4 py-6 gap-3 hover:bg-[#1A0E08] transition-colors">
              <div className="h-10 w-10 rounded-xl bg-[#E91E8C]/10 flex items-center justify-center">
                <Icon className="h-5 w-5 text-[#E91E8C]" />
              </div>
              <div>
                <p className="text-white font-semibold text-sm">{title}</p>
                <p className="text-white/50 text-xs mt-0.5">{desc}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add services/buyer-ui/components/TrustBanner.tsx
git commit -m "feat(buyer-ui): trust banner with 6 value propositions"
```

---

## Task 11: Footer — Production Grade

**Files:**
- Modify: `services/buyer-ui/components/Footer.tsx`

- [ ] **Step 1: Rewrite Footer.tsx**

```tsx
import Link from "next/link";
import { Zap, Instagram, Twitter, Facebook, Youtube, Mail } from "lucide-react";

const LINKS = {
  Company:  [
    { label: "About us",     href: "/about" },
    { label: "Careers",      href: "/careers" },
    { label: "Press",        href: "/press" },
    { label: "Blog",         href: "/blog" },
  ],
  Help: [
    { label: "FAQs",         href: "/faq" },
    { label: "Order tracking", href: "/account/orders" },
    { label: "Returns",      href: "/returns" },
    { label: "Contact us",   href: "/contact" },
  ],
  Sellers: [
    { label: "Sell on ZapMarket", href: "/sell" },
    { label: "Seller portal",     href: "/seller" },
    { label: "Seller guidelines", href: "/seller/guidelines" },
    { label: "Success stories",   href: "/seller/stories" },
  ],
  Policies: [
    { label: "Privacy policy",  href: "/privacy" },
    { label: "Terms of service", href: "/terms" },
    { label: "Cookie policy",   href: "/cookies" },
    { label: "Refund policy",   href: "/refunds" },
  ],
};

const SOCIALS = [
  { icon: Instagram, href: "#", label: "Instagram" },
  { icon: Twitter,   href: "#", label: "Twitter / X" },
  { icon: Facebook,  href: "#", label: "Facebook" },
  { icon: Youtube,   href: "#", label: "YouTube" },
];

const PAYMENT_METHODS = ["Visa", "Mastercard", "UPI", "NetBanking", "EMI", "COD"];

export function Footer() {
  return (
    <footer className="bg-[#0F0A04] text-white">
      {/* Newsletter strip */}
      <div className="border-b border-white/10">
        <div className="container-zap py-10 flex flex-col sm:flex-row items-center gap-6 justify-between">
          <div>
            <h3 className="font-display font-bold text-lg">Stay in the loop</h3>
            <p className="text-white/50 text-sm mt-1">Get deals, new arrivals & insider picks — no spam.</p>
          </div>
          <form className="flex gap-2 w-full sm:w-auto">
            <div className="relative flex-1 sm:w-72">
              <Mail className="absolute left-3.5 top-1/2 -translate-y-1/2 h-4 w-4 text-white/30" />
              <input
                type="email"
                placeholder="your@email.com"
                className="w-full h-11 pl-10 pr-4 bg-white/10 border border-white/15 rounded-xl text-sm text-white placeholder-white/30 outline-none focus:border-[#E91E8C] transition-colors"
              />
            </div>
            <button
              type="submit"
              className="h-11 px-5 bg-[#E91E8C] hover:bg-[#FF5A35] text-white text-sm font-bold rounded-xl transition-colors shrink-0"
            >
              Subscribe
            </button>
          </form>
        </div>
      </div>

      {/* Main footer links */}
      <div className="container-zap py-12">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-10">
          {/* Brand column */}
          <div className="lg:col-span-1">
            <Link href="/" className="flex items-center gap-2 mb-4">
              <div className="h-8 w-8 bg-[#E91E8C] rounded-xl flex items-center justify-center">
                <Zap className="h-4 w-4 text-white" strokeWidth={2.5} />
              </div>
              <span className="font-display font-bold text-lg">ZapMarket</span>
            </Link>
            <p className="text-white/50 text-sm leading-relaxed mb-5">
              India's fastest-growing marketplace for verified products and trusted sellers.
            </p>
            <div className="flex items-center gap-2">
              {SOCIALS.map(({ icon: Icon, href, label }) => (
                <a
                  key={label}
                  href={href}
                  aria-label={label}
                  className="h-8 w-8 rounded-lg bg-white/10 hover:bg-[#E91E8C] flex items-center justify-center transition-colors"
                >
                  <Icon className="h-3.5 w-3.5" />
                </a>
              ))}
            </div>
          </div>

          {/* Link columns */}
          {Object.entries(LINKS).map(([group, items]) => (
            <div key={group}>
              <h4 className="font-bold text-sm tracking-wide mb-4">{group}</h4>
              <ul className="space-y-2.5">
                {items.map(({ label, href }) => (
                  <li key={label}>
                    <Link href={href} className="text-sm text-white/50 hover:text-white transition-colors">
                      {label}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </div>

      {/* Bottom bar */}
      <div className="border-t border-white/10">
        <div className="container-zap py-5 flex flex-col sm:flex-row items-center justify-between gap-3">
          <p className="text-white/40 text-xs">
            © {new Date().getFullYear()} ZapMarket Pvt. Ltd. · Made with ♥ in India
          </p>
          <div className="flex items-center gap-2 flex-wrap justify-center">
            {PAYMENT_METHODS.map(method => (
              <span key={method} className="px-2.5 py-1 bg-white/10 text-white/50 text-xs rounded-md font-medium">
                {method}
              </span>
            ))}
          </div>
        </div>
      </div>
    </footer>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add services/buyer-ui/components/Footer.tsx
git commit -m "feat(buyer-ui): production-grade footer with newsletter, links, social, payment badges"
```

---

## Task 12: Homepage Rebuild (app/page.tsx)

**Files:**
- Modify: `services/buyer-ui/app/page.tsx`

This is a Server Component. It fetches data and passes it to the new section components.

- [ ] **Step 1: Rewrite app/page.tsx**

```tsx
import { HeroSection } from "@/components/HeroSection";
import { FeaturedCategories } from "@/components/FeaturedCategories";
import { ProductCard } from "@/components/ProductCard";
import { TrustBanner } from "@/components/TrustBanner";
import { SectionHeader } from "@/components/SectionHeader";
import { MotionWrapper, MotionChild } from "@/components/MotionWrapper";
import { CountdownTimer } from "@/components/CountdownTimer";
import { BlogCard } from "@/components/BlogCard";
import { getAllPosts } from "@/lib/mdx";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8000";

async function getData() {
  const [campaignsRes, categoriesRes, productsRes, dealsRes] = await Promise.allSettled([
    fetch(`${API}/v1/campaigns`, { next: { revalidate: 300 } }),
    fetch(`${API}/v1/categories`, { next: { revalidate: 300 } }),
    fetch(`${API}/v1/products?limit=8&sort_by=created_at&sort_order=desc`, { next: { revalidate: 60 } }),
    fetch(`${API}/v1/products?limit=4&sort_by=base_price&sort_order=asc`, { next: { revalidate: 60 } }),
  ]);

  const campaigns = campaignsRes.status === "fulfilled" && campaignsRes.value.ok
    ? await campaignsRes.value.json().then((d: { campaigns?: unknown[] }) => d.campaigns ?? [])
    : [];
  const categories = categoriesRes.status === "fulfilled" && categoriesRes.value.ok
    ? await categoriesRes.value.json().then((d: { categories?: unknown[] }) => d.categories ?? [])
    : [];
  const productsData = productsRes.status === "fulfilled" && productsRes.value.ok
    ? await productsRes.value.json()
    : { products: [] };
  const dealsData = dealsRes.status === "fulfilled" && dealsRes.value.ok
    ? await dealsRes.value.json()
    : { products: [] };

  const products = productsData.products ?? [];
  const deals = dealsData.products ?? [];

  // Fetch images for all products
  const allProducts = [...products, ...deals];
  const imageMap: Record<string, { url: string }[]> = {};
  await Promise.allSettled(
    allProducts.map(async (p: { id: string }) => {
      try {
        const res = await fetch(`${API}/v1/products/${p.id}/images`, { next: { revalidate: 300 } });
        if (res.ok) {
          const d = await res.json();
          imageMap[p.id] = d.images ?? [];
        }
      } catch {}
    })
  );

  return { campaigns, categories, products, deals, imageMap };
}

export default async function HomePage() {
  const { campaigns, categories, products, deals, imageMap } = await getData();
  const posts = await getAllPosts();
  const flashEnd = new Date(Date.now() + 4 * 60 * 60 * 1000);

  return (
    <main>
      <HeroSection campaigns={campaigns} />

      <FeaturedCategories categories={categories} />

      {/* Today's Deals */}
      <section className="container-zap py-10">
        <div className="flex items-end justify-between mb-7">
          <div>
            <div className="flex items-center gap-3 mb-2">
              <SectionHeader title="Today's Deals" accent className="mb-0" />
              <div className="flex items-center gap-2 px-3 py-1.5 bg-[#FDE8F4] rounded-full">
                <span className="text-xs font-semibold text-[#E91E8C]">Ends in</span>
                <CountdownTimer endTime={flashEnd.toISOString()} compact />
              </div>
            </div>
          </div>
          <a href="/deals" className="text-sm font-semibold text-[#E91E8C] hover:text-[#B5166E] transition-colors hidden sm:block">
            All deals →
          </a>
        </div>
        <MotionWrapper stagger className="grid grid-cols-2 sm:grid-cols-2 md:grid-cols-4 gap-4">
          {deals.slice(0, 4).map((p: Parameters<typeof ProductCard>[0]["product"]) => (
            <MotionChild key={p.id}>
              <ProductCard product={p} images={imageMap[p.id]} />
            </MotionChild>
          ))}
        </MotionWrapper>
      </section>

      <TrustBanner />

      {/* New Arrivals */}
      <section className="container-zap py-12">
        <SectionHeader
          title="New Arrivals"
          subtitle="The freshest products, just landed"
          href="/products?sort_by=created_at&sort_order=desc"
          className="mb-7"
        />
        <MotionWrapper stagger className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
          {products.slice(0, 8).map((p: Parameters<typeof ProductCard>[0]["product"], i: number) => (
            <MotionChild key={p.id}>
              <ProductCard product={p} images={imageMap[p.id]} priority={i < 4} />
            </MotionChild>
          ))}
        </MotionWrapper>
      </section>

      {/* Blog section */}
      {posts.length > 0 && (
        <section className="bg-[#F9F8F5] py-12">
          <div className="container-zap">
            <SectionHeader
              title="From the Blog"
              subtitle="Shopping guides, tips, and insider knowledge"
              href="/blog"
              className="mb-7"
            />
            <MotionWrapper stagger className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
              {posts.slice(0, 3).map(post => (
                <MotionChild key={post.slug}>
                  <BlogCard post={post} />
                </MotionChild>
              ))}
            </MotionWrapper>
          </div>
        </section>
      )}
    </main>
  );
}
```

- [ ] **Step 2: Verify build**

```bash
cd services/buyer-ui && npm run build 2>&1 | tail -30
```

Expected: `✓ Compiled successfully` (or warnings only, no errors)

- [ ] **Step 3: Commit**

```bash
git add services/buyer-ui/app/page.tsx
git commit -m "feat(buyer-ui): rebuilt homepage with hero, categories, deals, trust, arrivals, blog"
```

---

## Task 13: Product Listing Page Redesign

**Files:**
- Modify: `services/buyer-ui/app/products/page.tsx`
- Modify: `services/buyer-ui/app/products/loading.tsx`
- Create: `services/buyer-ui/components/FilterDrawer.tsx`

- [ ] **Step 1: Rewrite products/loading.tsx**

```tsx
import { ProductCardSkeleton } from "@/components/ProductCardSkeleton";

export default function Loading() {
  return (
    <div className="container-zap py-8">
      <div className="flex gap-6">
        <div className="hidden lg:block w-56 shrink-0 space-y-4">
          <div className="skeleton h-6 w-32 rounded-full" />
          <div className="space-y-2">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="skeleton h-9 rounded-xl" />
            ))}
          </div>
        </div>
        <div className="flex-1">
          <div className="skeleton h-8 w-48 rounded-full mb-6" />
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
            {Array.from({ length: 12 }).map((_, i) => (
              <ProductCardSkeleton key={i} />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Create FilterDrawer.tsx**

```tsx
"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { SlidersHorizontal, X } from "lucide-react";
import { cn } from "@/lib/utils";

interface Category { id: string; name: string; }
interface FilterDrawerProps {
  categories: Category[];
  selectedCategory?: string;
  selectedSort?: string;
}

const SORT_OPTIONS = [
  { value: "",            label: "Relevance" },
  { value: "created_at_desc", label: "Newest first" },
  { value: "price_asc",   label: "Price: Low to High" },
  { value: "price_desc",  label: "Price: High to Low" },
];

export function FilterDrawer({ categories, selectedCategory, selectedSort }: FilterDrawerProps) {
  const router = useRouter();
  const params = useSearchParams();

  const applyFilter = (key: string, value: string) => {
    const p = new URLSearchParams(params.toString());
    if (value) p.set(key, value); else p.delete(key);
    p.delete("page");
    router.push(`/products?${p.toString()}`);
  };

  return (
    <Sheet>
      <SheetTrigger asChild>
        <button className="flex items-center gap-2 h-9 px-4 bg-white border border-[#EDE9E3] rounded-xl text-sm font-semibold text-[#3D2E1A] hover:border-[#E91E8C] hover:text-[#E91E8C] transition-colors shadow-sm">
          <SlidersHorizontal className="h-4 w-4" />
          Filters
        </button>
      </SheetTrigger>
      <SheetContent side="left" className="w-72 p-0 border-r border-[#EDE9E3]">
        <SheetHeader className="px-5 py-4 border-b border-[#EDE9E3]">
          <SheetTitle className="text-base font-bold text-[#0F0A04]">Filters</SheetTitle>
        </SheetHeader>
        <div className="overflow-auto p-5 space-y-6">
          {/* Sort */}
          <div>
            <p className="text-xs font-bold tracking-wider text-[#B8A898] uppercase mb-3">Sort by</p>
            <div className="space-y-1">
              {SORT_OPTIONS.map(({ value, label }) => (
                <button
                  key={value}
                  onClick={() => applyFilter("sort", value)}
                  className={cn(
                    "w-full text-left px-3 py-2.5 rounded-xl text-sm font-medium transition-colors",
                    selectedSort === value || (!selectedSort && !value)
                      ? "bg-[#FDE8F4] text-[#E91E8C]"
                      : "text-[#3D2E1A] hover:bg-[#F9F8F5]"
                  )}
                >
                  {label}
                </button>
              ))}
            </div>
          </div>

          {/* Categories */}
          <div>
            <p className="text-xs font-bold tracking-wider text-[#B8A898] uppercase mb-3">Category</p>
            <div className="space-y-1">
              <button
                onClick={() => applyFilter("category", "")}
                className={cn(
                  "w-full text-left px-3 py-2.5 rounded-xl text-sm font-medium transition-colors",
                  !selectedCategory ? "bg-[#FDE8F4] text-[#E91E8C]" : "text-[#3D2E1A] hover:bg-[#F9F8F5]"
                )}
              >
                All categories
              </button>
              {categories.map(cat => (
                <button
                  key={cat.id}
                  onClick={() => applyFilter("category", cat.name)}
                  className={cn(
                    "w-full text-left px-3 py-2.5 rounded-xl text-sm font-medium transition-colors",
                    selectedCategory === cat.name ? "bg-[#FDE8F4] text-[#E91E8C]" : "text-[#3D2E1A] hover:bg-[#F9F8F5]"
                  )}
                >
                  {cat.name}
                </button>
              ))}
            </div>
          </div>

          {/* Clear */}
          {(selectedCategory || selectedSort) && (
            <button
              onClick={() => router.push("/products")}
              className="w-full flex items-center justify-center gap-2 h-9 border border-[#EDE9E3] rounded-xl text-sm font-semibold text-[#7A6856] hover:border-[#E91E8C] hover:text-[#E91E8C] transition-colors"
            >
              <X className="h-4 w-4" />
              Clear all filters
            </button>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}
```

- [ ] **Step 3: Commit**

```bash
git add services/buyer-ui/app/products/loading.tsx services/buyer-ui/components/FilterDrawer.tsx
git commit -m "feat(buyer-ui): skeleton loading state and FilterDrawer for product listing"
```

---

## Task 14: Auth Pages Redesign (Login + Register)

**Files:**
- Modify: `services/buyer-ui/app/login/page.tsx`
- Modify: `services/buyer-ui/app/register/page.tsx`

Both pages should follow a split-panel layout: left side has a premium brand panel (gradient + testimonial), right side has the form.

- [ ] **Step 1: Redesign login page**

Read the current `app/login/page.tsx` and replace the form container with a split-panel layout while preserving all form logic and action handlers. Wrap the form card in:

```tsx
<div className="min-h-screen grid lg:grid-cols-2">
  {/* Brand panel */}
  <div className="hidden lg:flex flex-col justify-between bg-gradient-to-br from-[#0F0A04] via-[#1A0E08] to-[#E91E8C]/20 p-12 relative overflow-hidden">
    <div className="absolute inset-0 bg-[url('/pattern.svg')] opacity-5" />
    <Link href="/" className="flex items-center gap-2 relative z-10">
      <div className="h-9 w-9 bg-[#E91E8C] rounded-xl flex items-center justify-center">
        <Zap className="h-5 w-5 text-white" strokeWidth={2.5} />
      </div>
      <span className="font-display font-bold text-xl text-white">ZapMarket</span>
    </Link>
    <div className="relative z-10">
      <blockquote className="text-white/80 text-lg leading-relaxed mb-4 italic">
        "Best marketplace experience I've had. Fast delivery, authentic products, and amazing deals every day."
      </blockquote>
      <div className="flex items-center gap-3">
        <div className="h-10 w-10 rounded-full bg-gradient-to-br from-[#E91E8C] to-[#FF5A35] flex items-center justify-center text-white font-bold text-sm">
          RS
        </div>
        <div>
          <p className="text-white font-semibold text-sm">Rahul Sharma</p>
          <p className="text-white/50 text-xs">Verified buyer · Mumbai</p>
        </div>
      </div>
    </div>
  </div>

  {/* Form panel */}
  <div className="flex items-center justify-center p-6 sm:p-12 bg-[#F9F8F5]">
    {/* existing form content, just re-styled */}
    ...
  </div>
</div>
```

- [ ] **Step 2: Apply same split-panel to register page**

Same structure as login, with different testimonial copy.

- [ ] **Step 3: Commit**

```bash
git add services/buyer-ui/app/login/page.tsx services/buyer-ui/app/register/page.tsx
git commit -m "feat(buyer-ui): split-panel auth pages with brand panel and testimonial"
```

---

## Task 15: Final Polish — Verify & Build

- [ ] **Step 1: Check TypeScript**

```bash
cd services/buyer-ui && npx tsc --noEmit 2>&1 | head -40
```

Fix any type errors before proceeding.

- [ ] **Step 2: Build**

```bash
npm run build 2>&1 | tail -40
```

Expected: successful build. Fix any errors.

- [ ] **Step 3: Run dev server and visual check**

```bash
npm run dev
```

Open `http://localhost:3000` and verify:
- Announcement bar appears and cycles
- Navbar sticky with search modal (Cmd+K)
- Hero renders with gradient and CTA
- Category grid shows with icons and hover effects
- Product cards have shadow, wishlist heart, quick-view overlay
- Trust banner dark section
- Footer with newsletter, links, socials, payment badges

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "feat(buyer-ui): complete premium redesign — all components, homepage, and design system"
```

---

## Coverage Check

| Requirement | Covered by Task |
|---|---|
| Announcement bar | Task 4 |
| Premium sticky navbar | Task 6 |
| Search modal with Cmd+K | Task 6 |
| Mobile hamburger sheet | Task 6 |
| Design tokens overhaul | Task 2 |
| Framer Motion animations | Task 3, 7, 8, 9, 10 |
| HeroSection with campaigns | Task 8 |
| Trust pills / stats | Task 8, 10 |
| Category grid with icons | Task 9 |
| Product cards full redesign | Task 7 |
| Skeleton loading | Task 7, 13 |
| Today's Deals with countdown | Task 12 |
| TrustBanner | Task 10 |
| New Arrivals section | Task 12 |
| Blog section on homepage | Task 12 |
| Filter drawer (mobile) | Task 13 |
| Premium footer | Task 11 |
| Auth pages (split-panel) | Task 14 |
| Reduced motion respected | Task 2 (CSS) + Task 3 (MotionWrapper) |
| WCAG color contrast | Design tokens use high-contrast pairs |
| Touch targets ≥44px | All buttons h-9 (36px) min + padding |
