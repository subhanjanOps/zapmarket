import Link from "next/link";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button, buttonVariants } from "@/components/ui/button";

interface PaginatorProps {
  page: number;
  total: number;
  limit: number;
  baseUrl: string;
}

export default function Paginator({ page, total, limit, baseUrl }: PaginatorProps) {
  const totalPages = Math.ceil(total / limit);
  if (totalPages <= 1) return null;
  const sep = baseUrl.includes("?") ? "&" : "?";

  return (
    <div className="flex items-center justify-center gap-3 mt-8">
      {page > 1 ? (
        <Link
          href={`${baseUrl}${sep}page=${page - 1}`}
          className={cn(buttonVariants({ variant: "outline" }), "gap-1")}
        >
          <ChevronLeft size={14} /> Prev
        </Link>
      ) : (
        <Button variant="outline" disabled className="gap-1">
          <ChevronLeft size={14} /> Prev
        </Button>
      )}

      <Button variant="secondary" disabled>
        {page} / {totalPages}
      </Button>

      {page < totalPages ? (
        <Link
          href={`${baseUrl}${sep}page=${page + 1}`}
          className={cn(buttonVariants({ variant: "outline" }), "gap-1")}
        >
          Next <ChevronRight size={14} />
        </Link>
      ) : (
        <Button variant="outline" disabled className="gap-1">
          Next <ChevronRight size={14} />
        </Button>
      )}
    </div>
  );
}
