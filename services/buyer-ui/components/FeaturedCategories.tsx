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

const ICON_MAP: Record<string, React.ComponentType<{ className?: string }>> = {
  Electronics:        Laptop,
  Fashion:            Shirt,
  "Home & Kitchen":   Home,
  "Sports & Fitness": Dumbbell,
  Sports:             Dumbbell,
  "Beauty & Health":  Sparkles,
  Beauty:             Sparkles,
  Books:              BookOpen,
  "Toys & Games":     Gamepad2,
  Toys:               Gamepad2,
  Grocery:            Apple,
  Automotive:         Car,
  "Baby & Kids":      Baby,
  Watches:            Watch,
  Audio:              Headphones,
};

const staggerParent = {
  hidden: {},
  visible: {
    transition: { staggerChildren: 0.07 },
  },
};

const staggerChild = {
  hidden: { opacity: 0, y: 16 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { duration: 0.35, ease: [0.25, 0.1, 0.25, 1] as const },
  },
};

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
    <section className="bg-[#F6F6F6] py-10">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="mb-6">
          <h2 className="text-xl font-bold text-[#111111] tracking-tight">
            Shop by Category
          </h2>
          <p className="text-sm text-[#555555] mt-1">
            Find exactly what you&apos;re looking for
          </p>
        </div>

        <motion.div
          className="grid grid-cols-4 md:grid-cols-8 gap-3 sm:gap-4"
          variants={staggerParent}
          initial="hidden"
          animate="visible"
        >
          {display.map((cat) => {
            const Icon = ICON_MAP[cat.name] ?? Sparkles;
            return (
              <motion.div key={cat.id} variants={staggerChild}>
                <Link
                  href={`/products?category_id=${encodeURIComponent(cat.id)}`}
                  className="group flex flex-col items-center gap-2 bg-white border border-[#E8E8E8] rounded-lg p-4 hover:border-[#111111] hover:shadow-sm transition-all duration-200"
                >
                  <motion.span whileHover={{ y: -2 }} className="block">
                    <Icon className="h-6 w-6 text-[#555555] group-hover:text-[#E91E8C] transition-colors duration-200" />
                  </motion.span>
                  <span className="text-xs font-medium text-[#555555] group-hover:text-[#111111] text-center leading-tight transition-colors duration-200">
                    {cat.name}
                  </span>
                </Link>
              </motion.div>
            );
          })}
        </motion.div>
      </div>
    </section>
  );
}
