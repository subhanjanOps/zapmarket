import type { Metadata } from "next";
import "./globals.css";
import { DialogProvider } from "./components/Dialog";
import { ToastProvider } from "./components/Toast";
import { Providers } from "./providers";

export const metadata: Metadata = {
  title: "ZapMarket Backoffice",
  description: "Business operations — catalog, moderation, and more",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" data-theme="enterprise-dark">
      <head>
        {/* Inline script applies saved theme before first paint to prevent FOUC */}
        <script
          dangerouslySetInnerHTML={{
            __html: `(function(){var t=localStorage.getItem('bo_theme');if(t)document.documentElement.setAttribute('data-theme',t);})()`,
          }}
        />
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
      </head>
      <body>
        <Providers>
          <DialogProvider><ToastProvider>{children}</ToastProvider></DialogProvider>
        </Providers>
      </body>
    </html>
  );
}
