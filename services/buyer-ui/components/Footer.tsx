import Link from "next/link";
import { Zap, Twitter, Instagram, Linkedin } from "lucide-react";
import { Separator } from "@/components/ui/separator";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import NewsletterForm from "./NewsletterForm";

export default function Footer() {
  return (
    <footer className="bg-foreground mt-20">
      <div className="max-w-7xl mx-auto px-4 py-14 grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-10 text-sm">
        {/* Brand column */}
        <div className="flex flex-col gap-4">
          <div className="flex items-center gap-2">
            <Zap size={20} fill="#FF2D78" stroke="#FF2D78" />
            <span
              className="font-extrabold text-base uppercase tracking-wider text-background"
              style={{ fontFamily: "var(--font-syne)" }}
            >
              ZapMarket
            </span>
          </div>
          <p className="text-background/50 text-xs leading-relaxed">
            The fastest marketplace in India. Real sellers, real products, zapped to your door.
          </p>
          <div className="flex items-center gap-1 mt-1">
            {[
              { href: "https://x.com", label: "Twitter / X", Icon: Twitter },
              { href: "https://instagram.com", label: "Instagram", Icon: Instagram },
              { href: "https://linkedin.com", label: "LinkedIn", Icon: Linkedin },
            ].map(({ href, label, Icon }) => (
              <a
                key={label}
                href={href}
                target="_blank"
                rel="noopener noreferrer"
                aria-label={label}
                className={cn(
                  buttonVariants({ variant: "ghost", size: "icon" }),
                  "text-background/60 hover:text-background hover:bg-background/10"
                )}
              >
                <Icon size={16} />
              </a>
            ))}
          </div>
        </div>

        {/* Shop column */}
        <div>
          <h3
            className="font-semibold text-xs uppercase tracking-wider text-background/40 mb-4"
            style={{ fontFamily: "var(--font-syne)" }}
          >
            Shop
          </h3>
          <ul className="space-y-3">
            <li>
              <Link href="/products" className="text-background/70 hover:text-background transition-colors duration-150">
                All products
              </Link>
            </li>
            <li>
              <Link href="/deals/summer-sale" className="text-background/70 hover:text-background transition-colors duration-150">
                Today&apos;s deals
              </Link>
            </li>
            <li>
              <Link href="/categories" className="text-background/70 hover:text-background transition-colors duration-150">
                Categories
              </Link>
            </li>
          </ul>
        </div>

        {/* Company column */}
        <div>
          <h3
            className="font-semibold text-xs uppercase tracking-wider text-background/40 mb-4"
            style={{ fontFamily: "var(--font-syne)" }}
          >
            Company
          </h3>
          <ul className="space-y-3">
            <li>
              <Link href="/about" className="text-background/70 hover:text-background transition-colors duration-150">
                About us
              </Link>
            </li>
            <li>
              <Link href="/blog" className="text-background/70 hover:text-background transition-colors duration-150">
                Blog
              </Link>
            </li>
            <li>
              <Link href="/careers" className="text-background/70 hover:text-background transition-colors duration-150">
                Careers
              </Link>
            </li>
            <li>
              <Link href="/contact" className="text-background/70 hover:text-background transition-colors duration-150">
                Contact
              </Link>
            </li>
          </ul>
        </div>

        {/* Newsletter column */}
        <div>
          <h3
            className="font-semibold text-xs uppercase tracking-wider text-background/40 mb-4"
            style={{ fontFamily: "var(--font-syne)" }}
          >
            Stay in the loop
          </h3>
          <p className="text-background/50 text-xs mb-4 leading-relaxed">
            New deals and products, straight to your inbox.
          </p>
          <NewsletterForm />
        </div>
      </div>

      <Separator className="bg-background/10" />

      <div className="max-w-7xl mx-auto px-4 py-4 flex flex-col sm:flex-row items-center justify-between gap-2 text-xs text-background/30">
        <span>&copy; {new Date().getFullYear()} ZapMarket. All rights reserved.</span>
        <span>Made with ♥ in India</span>
      </div>
    </footer>
  );
}
