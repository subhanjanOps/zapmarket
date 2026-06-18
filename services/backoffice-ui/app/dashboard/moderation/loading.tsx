import { TableSkeleton } from "@/app/components/Skeleton";
export default function Loading() {
  return (
    <div style={{ padding: "2rem" }}>
      <div className="skeleton" style={{ height: 28, width: 180, marginBottom: "1.75rem" }} />
      <div className="card" style={{ padding: 0, overflow: "hidden" }}>
        <table><tbody><TableSkeleton rows={8} cols={5} /></tbody></table>
      </div>
    </div>
  );
}
