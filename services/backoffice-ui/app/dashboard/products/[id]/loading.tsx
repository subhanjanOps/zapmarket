import Skeleton from "@/app/components/Skeleton";
import { TableSkeleton } from "@/app/components/Skeleton";

export default function Loading() {
  return (
    <div style={{ padding: "2rem" }}>
      <Skeleton h={14} w={120} />
      <div style={{ marginTop: "1.5rem", display: "grid", gridTemplateColumns: "1fr 320px", gap: "1.5rem" }}>
        <div className="card" style={{ padding: "1.5rem" }}>
          <Skeleton h={22} w={280} />
          <Skeleton h={12} w={180} />
          <div style={{ marginTop: "1.5rem" }}>
            <Skeleton h={12} w={80} />
            <Skeleton h={80} w="100%" />
          </div>
        </div>
        <div style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
          <div className="card" style={{ padding: "1.25rem" }}>
            <Skeleton h={12} w={80} />
            <Skeleton h={20} w={100} />
          </div>
          <div className="card" style={{ padding: "1.25rem" }}>
            <Skeleton h={12} w={100} />
            <Skeleton h={14} w={160} />
          </div>
        </div>
      </div>
      <div className="card" style={{ marginTop: "1.5rem", padding: 0, overflow: "hidden" }}>
        <table><tbody><TableSkeleton rows={4} cols={5} /></tbody></table>
      </div>
    </div>
  );
}
