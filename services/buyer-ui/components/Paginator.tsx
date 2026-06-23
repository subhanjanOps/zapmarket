import Link from "next/link";

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
    <div className="flex items-center justify-center gap-2 mt-8">
      {page > 1 && (
        <Link href={`${baseUrl}${sep}page=${page - 1}`} className="px-3 py-1 border rounded text-sm hover:bg-gray-100">← Prev</Link>
      )}
      <span className="text-sm text-gray-600">Page {page} of {totalPages}</span>
      {page < totalPages && (
        <Link href={`${baseUrl}${sep}page=${page + 1}`} className="px-3 py-1 border rounded text-sm hover:bg-gray-100">Next →</Link>
      )}
    </div>
  );
}
