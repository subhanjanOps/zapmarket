import { SkeletonTableCard } from "@/app/components/Skeleton";

export default function MetricsLoading() {
  return (
    <div>
      <div className="page-header">
        <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
          <div className="skeleton" style={{ width: "5rem", height: "1rem" }} />
          <div className="skeleton" style={{ width: "14rem", height: "0.75rem" }} />
        </div>
      </div>
      <SkeletonTableCard cols={5} rows={5} />
    </div>
  );
}
