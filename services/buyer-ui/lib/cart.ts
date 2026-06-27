import { create } from "zustand";
import { persist } from "zustand/middleware";

const MAX_CART_ITEMS = 50;

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
  addItem: (item: Omit<CartItem, "qty"> & { qty?: number }) => void;
  removeItem: (skuId: string) => void;
  updateQty: (skuId: string, qty: number) => void;
  clearCart: () => void;
}

export const useCartStore = create<CartStore>()(
  persist(
    (set) => ({
      items: [],
      addItem: (item) =>
        set((s) => {
          const qty = item.qty ?? 1;
          const existing = s.items.find((i) => i.skuId === item.skuId);
          if (existing) {
            return {
              items: s.items.map((i) =>
                i.skuId === item.skuId ? { ...i, qty: i.qty + qty } : i
              ),
            };
          }
          if (s.items.length >= MAX_CART_ITEMS) return s; // hard cap to bound localStorage size
          return { items: [...s.items, { ...item, qty }] };
        }),
      removeItem: (skuId) =>
        set((s) => ({ items: s.items.filter((i) => i.skuId !== skuId) })),
      updateQty: (skuId, qty) =>
        set((s) => ({
          items:
            qty <= 0
              ? s.items.filter((i) => i.skuId !== skuId)
              : s.items.map((i) => (i.skuId === skuId ? { ...i, qty } : i)),
        })),
      clearCart: () => set({ items: [] }),
    }),
    { name: "buyer-cart" }
  )
);

export const selectCartTotal = (items: CartItem[]): number =>
  items.reduce((sum, i) => sum + i.price * i.qty, 0);
