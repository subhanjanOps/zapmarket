import type { Metadata } from "next";
import "./globals.css";
import { DialogProvider } from "./components/Dialog";
import { ToastProvider } from "./components/Toast";

export const metadata: Metadata = {
  title: "ZapMarket Backoffice",
  description: "Business operations — catalog, moderation, and more",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" data-theme="enterprise-dark">
      <head>
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
      </head>
      <body>
        <DialogProvider><ToastProvider>{children}</ToastProvider></DialogProvider>
      </body>
    </html>
  );
}
