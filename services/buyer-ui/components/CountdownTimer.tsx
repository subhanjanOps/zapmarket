"use client";
import { useEffect, useState } from "react";
import { Clock } from "lucide-react";

function pad(n: number) { return String(n).padStart(2, "0"); }

function Segment({ value, label }: { value: string; label: string }) {
  return (
    <div className="flex flex-col items-center">
      <div
        className="w-14 h-14 flex items-center justify-center rounded-2xl text-2xl font-extrabold"
        style={{
          background: "rgba(255,255,255,0.15)",
          color: "#fff",
          fontVariantNumeric: "tabular-nums",
          fontFamily: "var(--font-syne)",
          backdropFilter: "blur(8px)",
        }}
      >
        {value}
      </div>
      <span className="text-[10px] font-semibold mt-1 uppercase tracking-widest" style={{ color: "rgba(255,255,255,0.65)" }}>
        {label}
      </span>
    </div>
  );
}

export default function CountdownTimer({ validUntil }: { validUntil: string }) {
  const [diff, setDiff] = useState(0);

  useEffect(() => {
    const end = new Date(validUntil).getTime();
    const tick = () => setDiff(Math.max(0, end - Date.now()));
    tick();
    const t = setInterval(tick, 1000);
    return () => clearInterval(t);
  }, [validUntil]);

  if (diff <= 0) return (
    <p className="text-white/70 font-semibold text-sm flex items-center gap-1.5">
      <Clock size={14} /> Deal expired
    </p>
  );

  const d = Math.floor(diff / 86400000);
  const h = Math.floor((diff % 86400000) / 3600000);
  const m = Math.floor((diff % 3600000) / 60000);
  const s = Math.floor((diff % 60000) / 1000);

  return (
    <div className="flex flex-col items-center gap-2">
      <p className="text-xs font-semibold uppercase tracking-widest flex items-center gap-1.5" style={{ color: "rgba(255,255,255,0.7)" }}>
        <Clock size={12} /> Ends in
      </p>
      <div className="flex items-end gap-2">
        {d > 0 && <Segment value={String(d)} label="days" />}
        {d > 0 && <span className="text-white/50 font-bold text-xl mb-3">:</span>}
        <Segment value={pad(h)} label="hrs" />
        <span className="text-white/50 font-bold text-xl mb-3">:</span>
        <Segment value={pad(m)} label="min" />
        <span className="text-white/50 font-bold text-xl mb-3">:</span>
        <Segment value={pad(s)} label="sec" />
      </div>
    </div>
  );
}
