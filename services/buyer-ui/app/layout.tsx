import type { Metadata } from "next";
import { Roboto } from "next/font/google";
import { cookies } from "next/headers";
import "./globals.css";
import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";
import { AnnouncementBar } from "@/components/AnnouncementBar";
import VerificationBanner from "@/components/VerificationBanner";
import { decodeJwtUser } from "@/lib/api";
import { cn } from "@/lib/utils";
import { Toaster } from "sonner";
import { Providers } from "./providers";

const roboto = Roboto({
  subsets: ["latin"],
  variable: "--font-roboto",
  weight: ["300", "400", "500", "700", "900"],
  display: "swap",
});

export const metadata: Metadata = {
  title: { default: "ZapMarket — Shop Smarter", template: "%s | ZapMarket" },
  description: "India's fastest online marketplace. Browse products, read guides, and shop deals.",
};

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const jar = await cookies();
  const token = jar.get("buyer_token")?.value;
  const raw = token ? decodeJwtUser(token) : null;
  const user = raw ? { name: raw.name ?? "", email: raw.email ?? "" } : null;
  const showVerificationBanner = raw && !raw.is_verified && !!raw.email;

  return (
    <html lang="en" className={roboto.variable}>
      <body className="min-h-screen flex flex-col" style={{ fontFamily: "var(--font-roboto), system-ui, sans-serif" }}>
        <Providers>
          <AnnouncementBar />
          {showVerificationBanner && <VerificationBanner email={raw!.email!} />}
          <Navbar user={user} />
          <main className="flex-1">{children}</main>
          <Footer />
          <Toaster position="bottom-right" richColors />
        </Providers>
      </body>
    </html>
  );
}
