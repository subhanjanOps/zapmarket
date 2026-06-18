import { SkeletonTableCard, SkeletonCard } from "@/app/components/Skeleton";

export default function AuditLoading() {
  return (
    <div>
      <div className="page-header">
        <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
          <div className="skeleton" style={{ width: "6rem", height: "1rem" }} />
          <div className="skeleton" style={{ width: "8rem", height: "0.75rem" }} />
        </div>
      </div>
      <SkeletonCard rows={3} />
      <div style={{ marginTop: "1rem" }}>
        <SkeletonTableCard cols={8} rows={8} />
      </div>
    </div>
  );
}
