import { TrendingUp, TrendingDown, Minus } from "lucide-react";
import type { LucideIcon } from "lucide-react";

export function StatCard({
  label,
  value,
  sub,
  accent = false,
  icon: Icon,
  iconColor,
  trend,
}: {
  label: string;
  value: string | number;
  sub?: string;
  accent?: boolean;
  icon?: LucideIcon;
  iconColor?: string;
  trend?: { delta: number; label?: string };
}) {
  const TrendIcon = trend
    ? trend.delta > 0
      ? TrendingUp
      : trend.delta < 0
      ? TrendingDown
      : Minus
    : null;

  const trendClass = trend
    ? trend.delta > 0
      ? "trend-up"
      : trend.delta < 0
      ? "trend-down"
      : "trend-flat"
    : "";

  const resolvedIconColor = iconColor ?? (accent ? "var(--accent)" : "var(--text-2)");
  const iconBg = iconColor
    ? `color-mix(in srgb, ${iconColor} 14%, transparent)`
    : accent
    ? "var(--accent-bg)"
    : "var(--surface2)";

  return (
    <div className="card" style={{ display: "flex", flexDirection: "column", gap: "0.875rem" }}>
      <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", gap: "0.75rem" }}>
        <div style={{ fontSize: "0.75rem", color: "var(--muted)", fontWeight: 500, lineHeight: 1.3, letterSpacing: "0.01em" }}>
          {label}
        </div>
        {Icon && (
          <div className="stat-icon" style={{ background: iconBg }}>
            <Icon size={18} style={{ color: resolvedIconColor }} strokeWidth={2} />
          </div>
        )}
      </div>

      <div>
        <div
          style={{
            fontSize: "2rem",
            fontWeight: 800,
            lineHeight: 1,
            color: accent ? "var(--accent)" : "var(--text)",
            letterSpacing: "-0.03em",
            fontFamily: '"Rubik", "Outfit", system-ui, sans-serif',
            fontVariantNumeric: "tabular-nums",
          }}
        >
          {value}
        </div>

        <div style={{ display: "flex", alignItems: "center", gap: "0.375rem", marginTop: "0.375rem", flexWrap: "wrap" }}>
          {sub && (
            <span style={{ fontSize: "0.75rem", color: "var(--muted)" }}>{sub}</span>
          )}
          {TrendIcon && trend && (
            <span className={trendClass} style={{ display: "inline-flex", alignItems: "center", gap: "0.2rem", fontSize: "0.75rem", fontWeight: 600 }}>
              <TrendIcon size={12} />
              {trend.delta > 0 ? "+" : ""}{trend.delta}%{trend.label ? ` ${trend.label}` : ""}
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
