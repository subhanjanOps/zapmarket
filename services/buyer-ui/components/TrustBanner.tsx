import { Truck, Shield, RefreshCw, Headphones, Award, CreditCard } from "lucide-react";

const TRUST_ITEMS = [
  { icon: Truck,       title: "Free Delivery",      desc: "On orders above ₹499" },
  { icon: Shield,      title: "Secure Payments",    desc: "256-bit SSL encryption" },
  { icon: RefreshCw,   title: "Easy Returns",       desc: "15-day hassle-free" },
  { icon: Headphones,  title: "24/7 Support",       desc: "Always here to help" },
  { icon: Award,       title: "Authentic Products", desc: "100% verified sellers" },
  { icon: CreditCard,  title: "EMI Available",      desc: "No-cost EMI on ₹3000+" },
];

export function TrustBanner() {
  return (
    <section className="bg-[#0F0A04] py-8 sm:py-10">
      <div className="container-zap">
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 divide-x divide-y lg:divide-y-0 divide-white/8 border border-white/8 rounded-2xl overflow-hidden">
          {TRUST_ITEMS.map(({ icon: Icon, title, desc }) => (
            <div
              key={title}
              className="flex flex-col items-center text-center px-4 py-6 gap-3 hover:bg-white/5 transition-colors"
            >
              <div className="h-10 w-10 rounded-xl bg-[#E91E8C]/10 flex items-center justify-center shrink-0">
                <Icon className="h-5 w-5 text-[#E91E8C]" />
              </div>
              <div>
                <p className="text-white font-semibold text-sm leading-tight">{title}</p>
                <p className="text-white/40 text-xs mt-0.5">{desc}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
