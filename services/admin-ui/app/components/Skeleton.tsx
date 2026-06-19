"use client";
import { useEffect, useState } from "react";

export function Spinner({ size = 28 }: { size?: number }) {
  return (
    <div
      className="spinner"
      style={{ width: size, height: size, borderWidth: size < 20 ? 2 : 2.5 }}
    />
  );
}

const BOOT_STEPS = [
  "initializing runtime…",
  "auth subsystem…",
  "route registry…",
  "connecting gateway…",
  "ready",
];

export function Splash() {
  const [progress, setProgress] = useState(0);
  const [stepIdx, setStepIdx] = useState(0);

  useEffect(() => {
    const targets = [0, 22, 48, 74, 100];
    const delays  = [0, 120, 240, 360, 480];
    const timers: ReturnType<typeof setTimeout>[] = [];
    delays.forEach((delay, i) => {
      timers.push(setTimeout(() => {
        setProgress(targets[i]);
        setStepIdx(i);
      }, delay));
    });
    return () => timers.forEach(clearTimeout);
  }, []);

  return (
    <div className="splash">
      <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "1.25rem" }}>
        <div className="splash-wordmark" style={{ display: "flex", alignItems: "center", gap: "0.375rem", fontSize: "1.125rem" }}>
          <span style={{ color: "var(--accent)", fontFamily: '"JetBrains Mono", monospace', fontWeight: 700 }}>{">"}_</span>
          Zap<span style={{ color: "var(--accent)" }}>Market</span>
        </div>

        <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "0.5rem" }}>
          <div className="splash-boot-bar-track">
            <div className="splash-boot-bar-fill" style={{ width: `${progress}%` }} />
          </div>
          <div className="splash-boot-msg">{BOOT_STEPS[stepIdx]}</div>
        </div>
      </div>
    </div>
  );
}

export function Skel({ w, h = "0.8rem", style }: { w?: string; h?: string; style?: React.CSSProperties }) {
  return (
    <div
      className="skeleton"
      style={{ width: w ?? "100%", height: h, borderRadius: 5, ...style }}
    />
  );
}

export function SkeletonStatCards({ count = 4 }: { count?: number }) {
  return (
    <div style={{
      display: "grid",
      gridTemplateColumns: "repeat(auto-fill, minmax(11rem, 1fr))",
      gap: "1rem",
      marginBottom: "1.75rem",
    }}>
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="card animate-in" style={{ padding: "1.25rem 1.375rem", display: "flex", flexDirection: "column", gap: "0.75rem", animationDelay: `${i * 0.04}s` }}>
          <Skel w="55%" h="0.5625rem" />
          <Skel w="65%" h="2rem" style={{ marginTop: "0.125rem" }} />
          <Skel w="80%" h="0.375rem" style={{ borderRadius: 2 }} />
          <Skel w="40%" h="0.5625rem" />
        </div>
      ))}
    </div>
  );
}

export function SkeletonTableRows({ cols, rows = 5 }: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }).map((_, r) => (
        <tr key={r} style={{ opacity: 1 - r * 0.12 }}>
          {Array.from({ length: cols }).map((_, c) => (
            <td key={c}>
              <Skel w={c === 0 ? "70%" : c % 2 === 0 ? "50%" : "80%"} />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

export function SkeletonCard({ rows = 4 }: { rows?: number }) {
  return (
    <div className="card" style={{ display: "flex", flexDirection: "column", gap: "0.75rem" }}>
      {Array.from({ length: rows }).map((_, i) => (
        <Skel key={i} w={i % 3 === 0 ? "60%" : i % 3 === 1 ? "90%" : "40%"} />
      ))}
    </div>
  );
}

export function SkeletonTableCard({ cols, rows = 6, title }: { cols: number; rows?: number; title?: string }) {
  return (
    <div className="card animate-in" style={{ padding: 0, overflow: "hidden" }}>
      {title && (
        <div className="card-header">
          <Skel w="8rem" h="0.75rem" />
        </div>
      )}
      <table>
        <thead>
          <tr>
            {Array.from({ length: cols }).map((_, i) => (
              <th key={i}><Skel w={i === 0 ? "5rem" : "4rem"} h="0.5625rem" /></th>
            ))}
          </tr>
        </thead>
        <tbody>
          <SkeletonTableRows cols={cols} rows={rows} />
        </tbody>
      </table>
    </div>
  );
}
