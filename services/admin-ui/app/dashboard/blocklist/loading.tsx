import { SkeletonTableCard } from "@/app/components/Skeleton";

export default function BlocklistLoading() {
  return (
    <div>
      <div className="page-header">
        <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
          <div className="skeleton" style={{ width: "7rem", height: "1rem" }} />
          <div className="skeleton" style={{ width: "10rem", height: "0.75rem" }} />
        </div>
      </div>
      <SkeletonTableCard cols={4} rows={5} />
    </div>
  );
}
