"use client";

import { motion } from "framer-motion";

const TESTIMONIALS = [
  {
    quote:
      "Ordered a laptop for my daughter and it arrived the very next day. The seller was responsive and packaging was excellent. Will definitely shop again.",
    author: "Priya Sharma",
    city: "Mumbai",
    initials: "PS",
  },
  {
    quote:
      "Returns process was so smooth. I returned a defective phone and got my refund in 3 days. Customer support was great throughout.",
    author: "Rahul Verma",
    city: "Delhi",
    initials: "RV",
  },
  {
    quote:
      "Found exactly what I was looking for at a price 30% cheaper than other sites. ZapMarket has become my go-to shopping destination.",
    author: "Anjali Patel",
    city: "Bangalore",
    initials: "AP",
  },
];

const cardVariants = {
  hidden: { opacity: 0, y: 16 },
  visible: (delay: number) => ({
    opacity: 1,
    y: 0,
    transition: { duration: 0.35, delay, ease: [0.25, 0.1, 0.25, 1] as const },
  }),
};

export function Testimonials() {
  return (
    <section className="bg-[#F6F6F6] py-14">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Section heading */}
        <div className="text-center">
          <h2 className="text-2xl font-bold text-[#111111] tracking-tight">
            What our customers say
          </h2>
          <p className="text-sm text-[#555555] mt-2">
            Trusted by over 2 million shoppers across India
          </p>
        </div>

        {/* Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-5 mt-10">
          {TESTIMONIALS.map((t, i) => (
            <motion.div
              key={t.author}
              custom={i * 0.1}
              variants={cardVariants}
              initial="hidden"
              whileInView="visible"
              viewport={{ once: true, margin: "-50px" }}
              whileHover={{ y: -4 }}
              className="bg-white border border-[#E8E8E8] rounded-lg p-6 shadow-sm hover:shadow-md transition-shadow duration-200"
            >
              {/* Quote icon */}
              <div className="text-[#E91E8C] text-4xl font-serif leading-none select-none">
                &ldquo;
              </div>

              {/* Review text */}
              <p className="text-sm text-[#555555] leading-relaxed italic mt-2">
                {t.quote}
              </p>

              {/* Stars */}
              <div className="flex items-center gap-0.5 mt-3">
                {Array.from({ length: 5 }).map((_, si) => (
                  <span key={si} className="text-[#F59E0B] text-xs">
                    &#9733;
                  </span>
                ))}
              </div>

              {/* Author row */}
              <div className="flex items-center gap-3 mt-4">
                <div className="h-7 w-7 rounded-full bg-[#111111] flex items-center justify-center text-white text-xs font-bold shrink-0">
                  {t.initials}
                </div>
                <div>
                  <p className="text-sm font-semibold text-[#111111] leading-tight">
                    {t.author}
                  </p>
                  <p className="text-xs text-[#999999]">{t.city}</p>
                </div>
              </div>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
}
