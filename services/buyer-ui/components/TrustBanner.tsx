"use client";

import { motion } from "framer-motion";
import { Truck, RefreshCw, ShieldCheck, Headphones } from "lucide-react";

const TRUST_ITEMS = [
  { icon: Truck,        title: "Free Delivery",    sub: "On orders above ₹499" },
  { icon: RefreshCw,    title: "Easy Returns",     sub: "30-day hassle-free" },
  { icon: ShieldCheck,  title: "Secure Payments",  sub: "256-bit SSL encrypted" },
  { icon: Headphones,   title: "24/7 Support",     sub: "Always here to help" },
];

const containerVariants = {
  hidden: {},
  visible: {
    transition: {
      staggerChildren: 0.08,
    },
  },
};

const itemVariants = {
  hidden:  { opacity: 0, y: 16 },
  visible: { opacity: 1, y: 0, transition: { duration: 0.35, ease: [0.25, 0.1, 0.25, 1] as const } },
};

export function TrustBanner() {
  return (
    <section className="bg-white border-y border-[#E8E8E8] py-8">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <motion.div
          className="grid grid-cols-2 lg:grid-cols-4 divide-x divide-[#E8E8E8]"
          variants={containerVariants}
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-60px" }}
        >
          {TRUST_ITEMS.map(({ icon: Icon, title, sub }) => (
            <motion.div
              key={title}
              variants={itemVariants}
              className="flex flex-col items-center text-center px-6 py-4 gap-2"
            >
              <Icon className="h-6 w-6 text-[#E91E8C]" strokeWidth={1.75} />
              <div>
                <p className="text-sm font-semibold text-[#111111] leading-snug">{title}</p>
                <p className="text-xs text-[#555555] mt-0.5">{sub}</p>
              </div>
            </motion.div>
          ))}
        </motion.div>
      </div>
    </section>
  );
}
