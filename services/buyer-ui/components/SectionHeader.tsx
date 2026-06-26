import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { cn } from "@/lib/utils";

interface SectionHeaderProps {
  title: string;
  subtitle?: string;
  href?: string;
  viewAllLabel?: string;
  centered?: boolean;
  className?: string;
}

export function SectionHeader({
  title,
  subtitle,
  href,
  viewAllLabel = "View all",
  centered = false,
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
      <div>
        <h2 className="text-xl font-bold text-[#111111] tracking-tight">
          {title}
        </h2>
        {subtitle && (
          <p className="mt-1 text-sm text-[#555555]">{subtitle}</p>
        )}
      </div>

      {href && (
        <Link
          href={href}
          className="flex items-center gap-1 text-sm font-medium text-[#E91E8C] hover:text-[#C2187A] transition-colors shrink-0"
        >
          {viewAllLabel}
          <ArrowRight className="h-4 w-4" />
        </Link>
      )}
    </div>
  );
}
