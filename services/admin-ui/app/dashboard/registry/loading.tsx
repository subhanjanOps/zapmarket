import { SkeletonTableCard } from "@/app/components/Skeleton";

export default function RegistryLoading() {
  return (
    <div>
      <div className="page-header">
        <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
          <div className="skeleton" style={{ width: "9rem", height: "1rem" }} />
          <div className="skeleton" style={{ width: "12rem", height: "0.75rem" }} />
        </div>
      </div>
      <div style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
        <SkeletonTableCard cols={4} rows={3} />
        <SkeletonTableCard cols={4} rows={2} />
      </div>
    </div>
  );
}
