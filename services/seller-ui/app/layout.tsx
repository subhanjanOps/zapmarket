import type { Metadata } from "next";
import "./globals.css";
import { DialogProvider } from "./components/Dialog";

export const metadata: Metadata = {
  title: "ZapMarket — Seller Portal",
  description: "Manage your products and orders on ZapMarket",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" data-theme="vibrant">
      <head>
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
      </head>
      <body>
        <DialogProvider>{children}</DialogProvider>
      </body>
    </html>
  );
}
