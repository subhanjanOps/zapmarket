import { SkeletonStatCards, SkeletonTableCard } from "@/app/components/Skeleton";

export default function Loading() {
  return (
    <>
      <div style={{ height: "2.5rem", marginBottom: "1.75rem" }} />
      <SkeletonStatCards count={4} />
      <SkeletonTableCard cols={5} rows={6} />
    </>
  );
}
