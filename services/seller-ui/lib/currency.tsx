"use client";
import { createContext, useCallback, useContext, useEffect, useState } from "react";

// ── Currency catalogue ────────────────────────────────────────────────────────

export interface CurrencyMeta {
  code: string;
  name: string;
  flag: string;
}

export const CURRENCIES: CurrencyMeta[] = [
  { code: "USD", name: "US Dollar",          flag: "🇺🇸" },
  { code: "EUR", name: "Euro",               flag: "🇪🇺" },
  { code: "GBP", name: "British Pound",      flag: "🇬🇧" },
  { code: "JPY", name: "Japanese Yen",       flag: "🇯🇵" },
  { code: "CAD", name: "Canadian Dollar",    flag: "🇨🇦" },
  { code: "AUD", name: "Australian Dollar",  flag: "🇦🇺" },
  { code: "CHF", name: "Swiss Franc",        flag: "🇨🇭" },
  { code: "CNY", name: "Chinese Yuan",       flag: "🇨🇳" },
  { code: "INR", name: "Indian Rupee",       flag: "🇮🇳" },
  { code: "BRL", name: "Brazilian Real",     flag: "🇧🇷" },
  { code: "MXN", name: "Mexican Peso",       flag: "🇲🇽" },
  { code: "SGD", name: "Singapore Dollar",   flag: "🇸🇬" },
  { code: "HKD", name: "Hong Kong Dollar",   flag: "🇭🇰" },
  { code: "NOK", name: "Norwegian Krone",    flag: "🇳🇴" },
  { code: "SEK", name: "Swedish Krona",      flag: "🇸🇪" },
  { code: "DKK", name: "Danish Krone",       flag: "🇩🇰" },
  { code: "NZD", name: "New Zealand Dollar", flag: "🇳🇿" },
  { code: "ZAR", name: "South African Rand", flag: "🇿🇦" },
  { code: "AED", name: "UAE Dirham",         flag: "🇦🇪" },
  { code: "SAR", name: "Saudi Riyal",        flag: "🇸🇦" },
  { code: "KRW", name: "South Korean Won",   flag: "🇰🇷" },
  { code: "THB", name: "Thai Baht",          flag: "🇹🇭" },
  { code: "IDR", name: "Indonesian Rupiah",  flag: "🇮🇩" },
  { code: "MYR", name: "Malaysian Ringgit",  flag: "🇲🇾" },
  { code: "PHP", name: "Philippine Peso",    flag: "🇵🇭" },
];

// ── Locale → currency detection ───────────────────────────────────────────────

const REGION_CURRENCY: Record<string, string> = {
  US: "USD", CA: "CAD", GB: "GBP", AU: "AUD", NZ: "NZD", CH: "CHF",
  DE: "EUR", FR: "EUR", IT: "EUR", ES: "EUR", NL: "EUR", PT: "EUR",
  AT: "EUR", BE: "EUR", FI: "EUR", IE: "EUR", GR: "EUR", SK: "EUR",
  LU: "EUR", MT: "EUR", CY: "EUR", EE: "EUR", LV: "EUR", LT: "EUR",
  SI: "EUR", HR: "EUR",
  JP: "JPY", CN: "CNY", IN: "INR", BR: "BRL", MX: "MXN",
  SG: "SGD", HK: "HKD", NO: "NOK", SE: "SEK", DK: "DKK",
  ZA: "ZAR", AE: "AED", SA: "SAR", KR: "KRW", TH: "THB",
  ID: "IDR", MY: "MYR", PH: "PHP",
};

function detectCurrency(): string {
  try {
    const locale = navigator.language;           // "en-IN", "de-DE"
    const region = locale.split("-")[1]?.toUpperCase();
    return REGION_CURRENCY[region] ?? "USD";
  } catch {
    return "USD";
  }
}

// ── Exchange-rate fetch with 1-hour localStorage cache ────────────────────────

const RATES_KEY = "zap-fx-rates";
const RATES_TTL = 60 * 60 * 1000; // 1 hour

export async function fetchRates(): Promise<Record<string, number>> {
  try {
    const cached = localStorage.getItem(RATES_KEY);
    if (cached) {
      const { rates, ts } = JSON.parse(cached) as { rates: Record<string, number>; ts: number };
      if (Date.now() - ts < RATES_TTL) return rates;
    }
    const res  = await fetch("/api/proxy/api/v1/currencies/rates");
    const data = await res.json() as { base: string; date: string; rates: Record<string, number> };
    const rates: Record<string, number> = { USD: 1, ...data.rates };
    localStorage.setItem(RATES_KEY, JSON.stringify({ rates, ts: Date.now() }));
    return rates;
  } catch {
    // Serve stale cache on network failure rather than breaking the UI
    try {
      const cached = localStorage.getItem(RATES_KEY);
      if (cached) return (JSON.parse(cached) as { rates: Record<string, number> }).rates;
    } catch { /* */ }
    return { USD: 1 };
  }
}

// ── Context ───────────────────────────────────────────────────────────────────

// No-decimal currencies
const ZERO_DECIMAL = new Set(["JPY", "KRW", "IDR", "VND", "CLP", "PYG", "XAF", "XOF"]);

interface CurrencyContextValue {
  currency:     string;
  setCurrency:  (c: string) => void;
  rates:        Record<string, number>; // all relative to USD (USD = 1)
  ratesLoading: boolean;
  ratesDate:    string;                 // ISO date of rates, e.g. "2026-06-21"
  /** Format USD cents → display currency string */
  format:       (usdCents: number) => string;
  /** Format an amount stored in fromCurrency → display currency string */
  formatFrom:   (cents: number, fromCurrency: string) => string;
}

const CurrencyContext = createContext<CurrencyContextValue>({
  currency:     "USD",
  setCurrency:  () => {},
  rates:        { USD: 1 },
  ratesLoading: true,
  ratesDate:    "",
  format:       (c) => `$${(c / 100).toFixed(2)}`,
  formatFrom:   (c) => `$${(c / 100).toFixed(2)}`,
});

export function CurrencyProvider({ children }: { children: React.ReactNode }) {
  const [currency,     setCurrencyState] = useState("USD");
  const [rates,        setRates]         = useState<Record<string, number>>({ USD: 1 });
  const [ratesLoading, setRatesLoading]  = useState(true);
  const [ratesDate,    setRatesDate]     = useState("");

  useEffect(() => {
    const saved = localStorage.getItem("zap-currency");
    setCurrencyState(saved ?? detectCurrency());

    fetchRates().then((r) => {
      setRates(r);
      setRatesLoading(false);
      // Parse date from cache for display
      try {
        const raw = localStorage.getItem(RATES_KEY);
        if (raw) {
          const { ts } = JSON.parse(raw) as { ts: number };
          setRatesDate(new Date(ts).toLocaleDateString(undefined, { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" }));
        }
      } catch { /* */ }
    });
  }, []);

  const setCurrency = useCallback((c: string) => {
    setCurrencyState(c);
    localStorage.setItem("zap-currency", c);
  }, []);

  const fmt = useCallback((amount: number, cur: string) =>
    new Intl.NumberFormat(undefined, {
      style:                 "currency",
      currency:              cur,
      maximumFractionDigits: ZERO_DECIMAL.has(cur) ? 0 : 2,
      minimumFractionDigits: ZERO_DECIMAL.has(cur) ? 0 : 2,
    }).format(amount),
  []);

  const format = useCallback((usdCents: number): string => {
    const amount = (usdCents / 100) * (rates[currency] ?? 1);
    return fmt(amount, currency);
  }, [currency, rates, fmt]);

  const formatFrom = useCallback((cents: number, fromCurrency: string): string => {
    // Convert stored currency → USD → display currency
    const usd    = cents / 100 / (rates[fromCurrency] ?? 1);
    const amount = usd * (rates[currency] ?? 1);
    return fmt(amount, currency);
  }, [currency, rates, fmt]);

  return (
    <CurrencyContext.Provider value={{ currency, setCurrency, rates, ratesLoading, ratesDate, format, formatFrom }}>
      {children}
    </CurrencyContext.Provider>
  );
}

export function useCurrency() {
  return useContext(CurrencyContext);
}
