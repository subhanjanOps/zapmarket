"use client";

import { useState } from "react";
import { X } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import Link from "next/link";

export function AnnouncementBar() {
  const [visible, setVisible] = useState(true);

  return (
    <AnimatePresence>
      {visible && (
        <motion.div
          initial={{ height: "auto", opacity: 1 }}
          exit={{ height: 0, opacity: 0 }}
          transition={{ duration: 0.3, ease: [0.16, 1, 0.3, 1] }}
          className="overflow-hidden"
        >
          <div className="bg-[#0F0A04] py-2.5 relative">
            <div className="container-zap flex items-center justify-center">
              <p className="text-xs text-white/80 font-medium tracking-wide text-center">
                🎉 Free delivery on orders above ₹499 ·{" "}
                <Link
                  href="/deals"
                  className="text-[#E91E8C] font-bold hover:underline"
                >
                  Shop deals →
                </Link>
              </p>
            </div>
            <button
              onClick={() => setVisible(false)}
              aria-label="Dismiss announcement"
              className="absolute right-3 sm:right-6 top-1/2 -translate-y-1/2 p-1 rounded-full hover:bg-white/10 transition-colors text-white/50 hover:text-white"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
