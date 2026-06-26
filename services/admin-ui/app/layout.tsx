import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "ZapMarket Gateway Admin",
  description: "API Gateway administration panel",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" data-theme="terminal">
      <head>
        {/* Inline script applies saved theme before first paint to prevent FOUC */}
        <script
          dangerouslySetInnerHTML={{
            __html: `(function(){var t=localStorage.getItem('zap-theme');if(t)document.documentElement.setAttribute('data-theme',t);})()`,
          }}
        />
      </head>
      <body>{children}</body>
    </html>
  );
}
