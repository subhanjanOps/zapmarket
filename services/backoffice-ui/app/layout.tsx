import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "ZapMarket Backoffice",
  description: "Business operations — catalog, moderation, and more",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
