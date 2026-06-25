"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import {
  Laptop,
  Shirt,
  Home,
  Dumbbell,
  Sparkles,
  BookOpen,
  Gamepad2,
  Apple,
  Car,
  Baby,
  Watch,
  Headphones,
} from "lucide-react";
import { SectionHeader } from "@/components/SectionHeader";
import { MotionWrapper, MotionChild } from "@/components/MotionWrapper";

const ICON_MAP: Record<string, React.ComponentType<{ className?: string }>> = {
  Electronics:        Laptop,
  Fashion:            Shirt,
  "Home & Kitchen":   Home,
  "Sports & Fitness": Dumbbell,
  "Sports":           Dumbbell,
  "Beauty & Health":  Sparkles,
  "Beauty":           Sparkles,
  Books:              BookOpen,
  "Toys & Games":     Gamepad2,
  "Toys":             Gamepad2,
  Grocery:            Apple,
  Automotive:         Car,
  "Baby & Kids":      Baby,
  Watches:            Watch,
  Audio:              Headphones,
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

interface Category {
  id: string;
  name: string;
  slug?: string;
  product_count?: number;
}

export function FeaturedCategories({ categories }: { categories: Category[] }) {
  const display = categories.slice(0, 8);

  if (display.length === 0) return null;

  return (
    <section className="container-zap py-8 sm:py-10">
      <SectionHeader
        title="Shop by Category"
        subtitle="Find exactly what you're looking for"
        href="/products"
        viewAllLabel="All categories"
        className="mb-6 sm:mb-8"
      />
      <MotionWrapper stagger className="grid grid-cols-4 sm:grid-cols-4 md:grid-cols-8 gap-3 sm:gap-4">
        {display.map((cat, i) => {
          const color = COLORS[i % COLORS.length];
          const Icon = ICON_MAP[cat.name] ?? Sparkles;
          return (
            <MotionChild key={cat.id}>
              <Link
                href={`/products?category=${encodeURIComponent(cat.name)}`}
                className="group flex flex-col items-center gap-2.5 sm:gap-3 p-3 sm:p-4 rounded-2xl bg-white border border-[#EDE9E3] hover:border-transparent hover:shadow-[0_8px_24px_rgba(15,10,4,0.1)] transition-all duration-300 text-center"
              >
                <motion.div
                  className="h-12 w-12 sm:h-14 sm:w-14 rounded-2xl flex items-center justify-center"
                  style={{
                    backgroundColor: color.bg,
                    boxShadow: `0 0 0 1px ${color.ring}`,
                  }}
                  whileHover={{ scale: 1.1, rotate: -4 }}
                  transition={{ duration: 0.22, ease: [0.16, 1, 0.3, 1] }}
                >
                  <Icon className="h-5 w-5 sm:h-6 sm:w-6" style={{ color: color.icon }} />
                </motion.div>
                <span className="text-xs font-semibold text-[#3D2E1A] group-hover:text-[#E91E8C] transition-colors leading-tight">
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
