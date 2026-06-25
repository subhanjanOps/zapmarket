"use client";
import { useRouter } from "next/navigation";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const SORT_OPTIONS = [
  { sort_by: "created_at", sort_order: "desc", label: "Newest" },
  { sort_by: "price_amount", sort_order: "asc", label: "Price: Low to High" },
  { sort_by: "price_amount", sort_order: "desc", label: "Price: High to Low" },
] as const;

interface Props {
  baseHref: string; // current URL without sort params
  currentValue: string;
}

export default function SortSelect({ baseHref, currentValue }: Props) {
  const router = useRouter();

  function handleChange(val: string | null) {
    if (!val) return;
    const [sort_by, sort_order] = val.split(":");
    const url = new URL(baseHref, "http://x");
    url.searchParams.set("sort_by", sort_by);
    url.searchParams.set("sort_order", sort_order);
    url.searchParams.delete("page");
    router.push(url.pathname + url.search);
  }

  return (
    <Select value={currentValue} onValueChange={handleChange}>
      <SelectTrigger className="w-48 rounded-xl">
        <SelectValue placeholder="Sort by" />
      </SelectTrigger>
      <SelectContent>
        {SORT_OPTIONS.map((opt) => {
          const val = `${opt.sort_by}:${opt.sort_order}`;
          return (
            <SelectItem key={val} value={val}>
              {opt.label}
            </SelectItem>
          );
        })}
      </SelectContent>
    </Select>
  );
}
