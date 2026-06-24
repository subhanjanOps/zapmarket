"use client";
import { useEffect, useState } from "react";

export default function CountdownTimer({ validUntil }: { validUntil: string }) {
  const [diff, setDiff] = useState(0);

  useEffect(() => {
    const end = new Date(validUntil).getTime();
    const tick = () => setDiff(Math.max(0, end - Date.now()));
    tick();
    const t = setInterval(tick, 1000);
    return () => clearInterval(t);
  }, [validUntil]);

  if (diff <= 0) return <p className="text-red-500 font-semibold">Deal expired</p>;

  const h = Math.floor(diff / 3600000);
  const m = Math.floor((diff % 3600000) / 60000);
  const s = Math.floor((diff % 60000) / 1000);

  return (
    <p className="text-lg font-semibold text-gray-800">
      Deal ends in{" "}
      <span className="text-[#FF2D78]">
        {h}h {m}m {s}s
      </span>
    </p>
  );
}
