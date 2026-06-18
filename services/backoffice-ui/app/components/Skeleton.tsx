export default function Skeleton({ h = 16, w = "100%", radius = 5 }: { h?: number; w?: number | string; radius?: number }) {
  return (
    <div
      className="skeleton"
      style={{ height: h, width: w, borderRadius: radius, flexShrink: 0 }}
    />
  );
}

export function SkeletonRow({ cols = 4 }: { cols?: number }) {
  return (
    <tr>
      {Array.from({ length: cols }).map((_, i) => (
        <td key={i}>
          <Skeleton h={14} w={i === 0 ? 180 : i === cols - 1 ? 80 : 120} />
        </td>
      ))}
    </tr>
  );
}

export function TableSkeleton({ rows = 6, cols = 4 }: { rows?: number; cols?: number }) {
  return (
    <>
      {Array.from({ length: rows }).map((_, i) => (
        <SkeletonRow key={i} cols={cols} />
      ))}
    </>
  );
}
