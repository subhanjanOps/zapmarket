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
    <div
      className={cn(
        "flex items-end justify-between gap-4",
        centered && "flex-col items-center text-center",
        className
      )}
    >
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
          style={{ fontSize: "clamp(1.375rem, 2.5vw, 1.875rem)" }}
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
