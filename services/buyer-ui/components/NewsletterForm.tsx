"use client";
import { ArrowRight } from "lucide-react";

export default function NewsletterForm() {
  return (
    <form
      onSubmit={(e) => e.preventDefault()}
      className="flex w-full sm:w-auto max-w-sm"
    >
      <input
        type="email"
        placeholder="your@email.com"
        className="flex-1 px-4 py-2.5 text-sm rounded-l-full outline-none"
        style={{ background: "#2A2012", color: "#F0EDE8", border: "1px solid #3A3020", borderRight: "none" }}
      />
      <button
        type="submit"
        className="flex items-center gap-1.5 px-5 py-2.5 text-sm font-bold rounded-r-full text-white cursor-pointer
                   transition-all duration-200 hover:brightness-110 active:scale-95"
        style={{ background: "#FF2D78", fontFamily: "var(--font-syne)" }}
      >
        Subscribe <ArrowRight size={14} />
      </button>
    </form>
  );
}
