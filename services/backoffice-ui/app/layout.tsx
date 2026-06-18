import type { Metadata } from "next";
import "./globals.css";
import { DialogProvider } from "./components/Dialog";

export const metadata: Metadata = {
  title: "ZapMarket Backoffice",
  description: "Business operations — catalog, moderation, and more",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <DialogProvider>{children}</DialogProvider>
      </body>
    </html>
  );
}
