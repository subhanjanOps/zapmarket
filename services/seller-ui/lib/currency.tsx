"use client";
import { createContext, useCallback, useContext, useEffect, useState } from "react";

// ── Types ────────────────────────────────────────────────────────────────────

export interface CurrencyMeta {
  code: string;
  name: string;
  flag: string;
  decimals: number;
}

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
    const locale = navigator.language;
    const region = locale.split("-")[1]?.toUpperCase();
    return REGION_CURRENCY[region] ?? "USD";
  } catch {
    return "USD";
  }
}

// ── Exchange-rate fetch with 1-hour localStorage cache ────────────────────────

const RATES_KEY = "zap-fx-rates";
const CURRENCIES_KEY = "zap-currencies";
const RATES_TTL = 60 * 60 * 1000;       // 1 hour
const CURRENCIES_TTL = 24 * 60 * 60 * 1000; // 24 hours

// No-decimal currencies (kept as client-side fallback for formatting)
const ZERO_DECIMAL = new Set(["JPY", "KRW", "IDR", "VND", "CLP", "PYG", "XAF", "XOF"]);

interface RatesPayload {
  base: string;
  as_of: string;
  date: string;
  stale: boolean;
  rates: Record<string, number>;
}

export async function fetchRates(): Promise<{ rates: Record<string, number>; asOf: string; stale: boolean }> {
  try {
    const cached = localStorage.getItem(RATES_KEY);
    if (cached) {
      const { rates, ts, asOf, stale } = JSON.parse(cached) as {
        rates: Record<string, number>;
        ts: number;
        asOf: string;
        stale: boolean;
      };
      if (Date.now() - ts < RATES_TTL) return { rates, asOf, stale };
    }
    const res = await fetch("/api/proxy/v1/currencies/rates");
    const data = await res.json() as RatesPayload;
    const rates: Record<string, number> = { USD: 1, ...data.rates };
    localStorage.setItem(RATES_KEY, JSON.stringify({
      rates, ts: Date.now(), asOf: data.as_of ?? data.date, stale: data.stale ?? false,
    }));
    return { rates, asOf: data.as_of ?? data.date, stale: data.stale ?? false };
  } catch {
    try {
      const cached = localStorage.getItem(RATES_KEY);
      if (cached) {
        const { rates, asOf, stale } = JSON.parse(cached) as { rates: Record<string, number>; asOf: string; stale: boolean };
        return { rates, asOf, stale: true };
      }
    } catch { /* */ }
    return { rates: { USD: 1 }, asOf: "", stale: true };
  }
}

async function fetchCurrencyList(): Promise<CurrencyMeta[]> {
  try {
    const cached = localStorage.getItem(CURRENCIES_KEY);
    if (cached) {
      const { list, ts } = JSON.parse(cached) as { list: CurrencyMeta[]; ts: number };
      if (Date.now() - ts < CURRENCIES_TTL) return list;
    }
    const res = await fetch("/api/proxy/v1/currencies");
    if (!res.ok) throw new Error(`currency list fetch: ${res.status}`);
    const list = await res.json() as CurrencyMeta[];
    localStorage.setItem(CURRENCIES_KEY, JSON.stringify({ list, ts: Date.now() }));
    return list;
  } catch {
    // Serve stale cached list on network failure
    try {
      const cached = localStorage.getItem(CURRENCIES_KEY);
      if (cached) return (JSON.parse(cached) as { list: CurrencyMeta[] }).list;
    } catch { /* */ }
    return [];
  }
}

async function fetchServerPreference(): Promise<string | null> {
  try {
    const res = await fetch("/api/proxy/v1/users/me/preferences");
    if (!res.ok) return null;
    const data = await res.json() as { display_currency?: string };
    return data.display_currency ?? null;
  } catch {
    return null;
  }
}

async function saveServerPreference(code: string): Promise<void> {
  try {
    await fetch("/api/proxy/v1/users/me/preferences", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ display_currency: code }),
    });
  } catch {
    // Silently ignore — localStorage is the source of truth on failure
  }
}

// ── Context ───────────────────────────────────────────────────────────────────

interface CurrencyContextValue {
  currency:     string;
  setCurrency:  (c: string) => void;
  currencies:   CurrencyMeta[];
  rates:        Record<string, number>;
  ratesLoading: boolean;
  ratesDate:    string;
  stale:        boolean;
  /** Format USD cents → display currency string */
  format:       (usdCents: number) => string;
  /** Format an amount stored in fromCurrency → display currency string */
  formatFrom:   (cents: number, fromCurrency: string) => string;
}

const CurrencyContext = createContext<CurrencyContextValue>({
  currency:     "USD",
  setCurrency:  () => {},
  currencies:   [],
  rates:        { USD: 1 },
  ratesLoading: true,
  ratesDate:    "",
  stale:        false,
  format:       (c) => `$${(c / 100).toFixed(2)}`,
  formatFrom:   (c) => `$${(c / 100).toFixed(2)}`,
});

export function CurrencyProvider({ children }: { children: React.ReactNode }) {
  const [currency,     setCurrencyState] = useState("USD");
  const [currencies,   setCurrencies]    = useState<CurrencyMeta[]>([]);
  const [rates,        setRates]         = useState<Record<string, number>>({ USD: 1 });
  const [ratesLoading, setRatesLoading]  = useState(true);
  const [ratesDate,    setRatesDate]     = useState("");
  const [stale,        setStale]         = useState(false);

  useEffect(() => {
    const saved = localStorage.getItem("zap-currency");
    const initial = saved ?? detectCurrency();
    setCurrencyState(initial);

    // Fetch rates, currency list, and server preference in parallel
    Promise.all([fetchRates(), fetchCurrencyList(), fetchServerPreference()]).then(
      ([ratesResult, list, serverCurrency]) => {
        setRates(ratesResult.rates);
        setStale(ratesResult.stale);
        setRatesLoading(false);
        if (ratesResult.asOf) {
          try {
            setRatesDate(new Date(ratesResult.asOf).toLocaleDateString(undefined, {
              month: "short", day: "numeric", hour: "2-digit", minute: "2-digit",
            }));
          } catch { /* */ }
        }
        if (list.length > 0) setCurrencies(list);
        // Server preference wins over localStorage/locale detection
        if (serverCurrency) {
          setCurrencyState(serverCurrency);
          localStorage.setItem("zap-currency", serverCurrency);
        }
      }
    );
  }, []);

  const setCurrency = useCallback((c: string) => {
    setCurrencyState(c);
    localStorage.setItem("zap-currency", c);
    saveServerPreference(c); // fire-and-forget
  }, []);

  const getDecimals = useCallback((cur: string): number => {
    const meta = currencies.find(c => c.code === cur);
    if (meta) return meta.decimals;
    return ZERO_DECIMAL.has(cur) ? 0 : 2;
  }, [currencies]);

  const fmt = useCallback((amount: number, cur: string) =>
    new Intl.NumberFormat(undefined, {
      style:                 "currency",
      currency:              cur,
      maximumFractionDigits: getDecimals(cur),
      minimumFractionDigits: getDecimals(cur),
    }).format(amount),
  [getDecimals]);

  const format = useCallback((usdCents: number): string => {
    const amount = (usdCents / 100) * (rates[currency] ?? 1);
    return fmt(amount, currency);
  }, [currency, rates, fmt]);

  const formatFrom = useCallback((cents: number, fromCurrency: string): string => {
    const usd    = cents / 100 / (rates[fromCurrency] ?? 1);
    const amount = usd * (rates[currency] ?? 1);
    return fmt(amount, currency);
  }, [currency, rates, fmt]);

  return (
    <CurrencyContext.Provider value={{
      currency, setCurrency, currencies, rates, ratesLoading, ratesDate, stale, format, formatFrom,
    }}>
      {children}
    </CurrencyContext.Provider>
  );
}

export function useCurrency() {
  return useContext(CurrencyContext);
}

export function toCents(amount: number, currencyCode: string, currencies: CurrencyMeta[]): number {
  const decimals = currencies.find((c) => c.code === currencyCode)?.decimals ?? 2;
  return decimals === 0 ? Math.round(amount) : Math.round(amount * 100);
}

export function fromCents(amount: number, currencyCode: string, currencies: CurrencyMeta[]): number {
  const decimals = currencies.find((c) => c.code === currencyCode)?.decimals ?? 2;
  return decimals === 0 ? amount : amount / 100;
}
