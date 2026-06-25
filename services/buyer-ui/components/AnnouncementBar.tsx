"use client";

import { useState, useEffect } from "react";
import { X, Zap } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";

const MESSAGES = [
  "Free delivery on orders above ₹499 · Use code ZAPFREE",
  "Flash Sale ends in 4 hours — Up to 60% off Electronics",
  "New arrivals every Monday — Follow us for early access",
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
              <Zap className="h-3.5 w-3.5 text-[#E91E8C] shrink-0" strokeWidth={2.5} />
              <AnimatePresence mode="wait">
                <motion.span
                  key={index}
                  initial={{ opacity: 0, y: 8 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -8 }}
                  transition={{ duration: 0.25 }}
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
