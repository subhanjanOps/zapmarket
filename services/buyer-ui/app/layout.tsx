import type { Metadata } from "next";
import { Syne, Plus_Jakarta_Sans, Geist } from "next/font/google";
import { cookies } from "next/headers";
import "./globals.css";
import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";
import { AnnouncementBar } from "@/components/AnnouncementBar";
import { decodeJwtUser } from "@/lib/api";
import { cn } from "@/lib/utils";

const geist = Geist({subsets:['latin'],variable:'--font-sans'});

const syne = Syne({
  subsets: ["latin"],
  variable: "--font-syne",
  weight: ["400", "600", "700", "800"],
});

const jakarta = Plus_Jakarta_Sans({
  subsets: ["latin"],
  variable: "--font-jakarta",
  weight: ["400", "500", "600", "700"],
});

export const metadata: Metadata = {
  title: { default: "ZapMarket — Shop Smarter", template: "%s | ZapMarket" },
  description: "India's fastest online marketplace. Browse products, read guides, and shop deals.",
};

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const jar = await cookies();
  const token = jar.get("buyer_token")?.value;
  const user = token ? decodeJwtUser(token) : null;

  return (
    <html lang="en" className={cn(syne.variable, jakarta.variable, "font-sans", geist.variable)}>
      <body className="min-h-screen flex flex-col bg-[#F9F8F5] text-[#0F0A04]">
        <AnnouncementBar />
        <Navbar user={user} />
        <main className="flex-1">{children}</main>
        <Footer />
      </body>
    </html>
  );
}
