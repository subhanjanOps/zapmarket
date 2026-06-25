"use client";

import { motion } from "framer-motion";
import { Star } from "lucide-react";
import { SectionHeader } from "@/components/SectionHeader";

const TESTIMONIALS = [
  {
    quote:
      "Received my order in 2 days! Quality is exactly as described. Will definitely shop again.",
    author: "Priya M.",
    city: "Mumbai",
    rating: 5,
    initials: "PM",
  },
  {
    quote:
      "Best prices I've found anywhere. The seller response was instant and packaging was immaculate.",
    author: "Arjun K.",
    city: "Bengaluru",
    rating: 5,
    initials: "AK",
  },
  {
    quote:
      "Returned a product hassle-free. Customer support resolved my issue in under an hour.",
    author: "Sneha R.",
    city: "Delhi",
    rating: 5,
    initials: "SR",
  },
];

export function Testimonials() {
  return (
    <section className="bg-[#F3F0EB] py-10 sm:py-14">
      <div className="container-zap">
        <SectionHeader
          title="What our customers say"
          subtitle="Over 2 million verified reviews"
        />

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-5 mt-8">
          {TESTIMONIALS.map((t, i) => (
            <motion.div
              key={t.author}
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true, margin: "-50px" }}
              transition={{ duration: 0.4, delay: i * 0.1, ease: [0.16, 1, 0.3, 1] }}
              className="bg-white rounded-2xl p-6 shadow-[0_1px_3px_rgba(15,10,4,0.06),0_4px_12px_rgba(15,10,4,0.04)]"
            >
              {/* Stars */}
              <div className="flex items-center gap-0.5">
                {Array.from({ length: t.rating }).map((_, si) => (
                  <Star
                    key={si}
                    className="h-4 w-4 fill-[#D97706] text-[#D97706]"
                  />
                ))}
              </div>

              {/* Quote */}
              <p className="text-sm text-[#3D2E1A] leading-relaxed italic mt-3 mb-4">
                &ldquo;{t.quote}&rdquo;
              </p>

              {/* Author */}
              <div className="flex items-center gap-3">
                <div className="h-9 w-9 rounded-full bg-gradient-to-br from-[#E91E8C] to-[#FF5A35] flex items-center justify-center text-white text-xs font-bold shrink-0">
                  {t.initials}
                </div>
                <div>
                  <p className="text-xs font-semibold text-[#0F0A04]">
                    {t.author}
                  </p>
                  <p className="text-xs text-[#8B7355]">{t.city}</p>
                </div>
              </div>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
}
