import { SkeletonStatCards, SkeletonTableCard } from "@/app/components/Skeleton";

export default function OverviewLoading() {
  return (
    <div>
      <div className="page-header">
        <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
          <div className="skeleton" style={{ width: "7rem", height: "1rem" }} />
          <div className="skeleton" style={{ width: "12rem", height: "0.75rem" }} />
        </div>
      </div>
      <SkeletonStatCards count={4} />
      <SkeletonTableCard cols={5} rows={4} title="Upstream Metrics" />
    </div>
  );
}
