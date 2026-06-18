import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "ZapMarket — Seller Portal",
  description: "Manage your products and orders on ZapMarket",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
