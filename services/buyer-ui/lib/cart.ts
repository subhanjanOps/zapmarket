import { create } from "zustand";
import { persist } from "zustand/middleware";

export interface CartItem {
  skuId: string;
  name: string;
  image: string;
  price: number;
  currency: string;
  qty: number;
}

interface CartStore {
  items: CartItem[];
  addItem: (item: Omit<CartItem, "qty">) => void;
  removeItem: (skuId: string) => void;
  updateQty: (skuId: string, qty: number) => void;
  clearCart: () => void;
  total: () => number;
}

export const useCartStore = create<CartStore>()(
  persist(
    (set, get) => ({
      items: [],
      addItem: (item) =>
        set((s) => {
          const existing = s.items.find((i) => i.skuId === item.skuId);
          if (existing) {
            return { items: s.items.map((i) => i.skuId === item.skuId ? { ...i, qty: i.qty + 1 } : i) };
          }
          return { items: [...s.items, { ...item, qty: 1 }] };
        }),
      removeItem: (skuId) =>
        set((s) => ({ items: s.items.filter((i) => i.skuId !== skuId) })),
      updateQty: (skuId, qty) =>
        set((s) => ({
          items: qty <= 0
            ? s.items.filter((i) => i.skuId !== skuId)
            : s.items.map((i) => i.skuId === skuId ? { ...i, qty } : i),
        })),
      clearCart: () => set({ items: [] }),
      total: () => get().items.reduce((sum, i) => sum + i.price * i.qty, 0),
    }),
    { name: "buyer-cart" }
  )
);
