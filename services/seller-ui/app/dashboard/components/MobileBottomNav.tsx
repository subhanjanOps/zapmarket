"use client";
import Link from "next/link";
import { LayoutDashboard, Package, ShoppingBag } from "lucide-react";

const NAV = [
  { href: "/dashboard",          label: "Overview",  Icon: LayoutDashboard },
  { href: "/dashboard/products", label: "Products",  Icon: Package         },
  { href: "/dashboard/orders",   label: "Orders",    Icon: ShoppingBag     },
];

interface Props {
  pathname: string;
}

export function MobileBottomNav({ pathname }: Props) {
  return (
    <nav
      className="dash-bottom-nav"
      style={{
        display: "none", position: "fixed", bottom: 0, left: 0, right: 0,
        height: "4rem",
        background: "var(--surface)",
        borderTop: "1px solid var(--border)",
        boxShadow: "0 -4px 16px rgba(0,0,0,0.08)",
        zIndex: 30, paddingBottom: "env(safe-area-inset-bottom)",
      }}
    >
      <div style={{ display: "flex", height: "100%", alignItems: "center" }}>
        {NAV.map(({ href, label, Icon }) => {
          const active = pathname === href || (href !== "/dashboard" && pathname.startsWith(href));
          return (
            <Link
              key={href}
              href={href}
              style={{
                flex: 1, display: "flex", flexDirection: "column", alignItems: "center",
                justifyContent: "center", gap: "0.2rem", textDecoration: "none",
                color: active ? "var(--accent)" : "var(--muted)",
                fontSize: "0.625rem", fontWeight: active ? 700 : 500,
                letterSpacing: "0.02em", paddingTop: "0.25rem", transition: "color 0.15s",
              }}
            >
              <Icon size={20} strokeWidth={active ? 2.5 : 1.8} />
              {label}
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
