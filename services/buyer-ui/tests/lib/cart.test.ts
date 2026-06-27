import { useCartStore, selectCartTotal } from "@/lib/cart";

const item1 = {
  skuId: "sku-001",
  name: "Widget A",
  image: "https://example.com/widget-a.png",
  price: 1099,
  currency: "USD",
};

const item2 = {
  skuId: "sku-002",
  name: "Widget B",
  image: "https://example.com/widget-b.png",
  price: 499,
  currency: "USD",
};

beforeEach(() => {
  useCartStore.getState().clearCart();
});

describe("addItem", () => {
  it("adds a new item with qty 1 by default", () => {
    useCartStore.getState().addItem(item1);
    const { items } = useCartStore.getState();
    expect(items).toHaveLength(1);
    expect(items[0].skuId).toBe("sku-001");
    expect(items[0].qty).toBe(1);
  });

  it("increments qty when the same skuId is added again", () => {
    useCartStore.getState().addItem(item1);
    useCartStore.getState().addItem(item1);
    const { items } = useCartStore.getState();
    expect(items).toHaveLength(1);
    expect(items[0].qty).toBe(2);
  });

  it("increments by custom qty when same skuId is added again with qty", () => {
    useCartStore.getState().addItem(item1);
    useCartStore.getState().addItem({ ...item1, qty: 3 });
    const { items } = useCartStore.getState();
    expect(items).toHaveLength(1);
    expect(items[0].qty).toBe(4);
  });

  it("respects a custom qty parameter for a new item", () => {
    useCartStore.getState().addItem({ ...item1, qty: 5 });
    const { items } = useCartStore.getState();
    expect(items).toHaveLength(1);
    expect(items[0].qty).toBe(5);
  });

  it("stores all fields — name, image, price, currency", () => {
    useCartStore.getState().addItem(item1);
    const stored = useCartStore.getState().items[0];
    expect(stored.name).toBe(item1.name);
    expect(stored.image).toBe(item1.image);
    expect(stored.price).toBe(item1.price);
    expect(stored.currency).toBe(item1.currency);
  });
});

describe("removeItem", () => {
  it("removes an item by skuId", () => {
    useCartStore.getState().addItem(item1);
    useCartStore.getState().addItem(item2);
    useCartStore.getState().removeItem("sku-001");
    const { items } = useCartStore.getState();
    expect(items).toHaveLength(1);
    expect(items[0].skuId).toBe("sku-002");
  });

  it("is a no-op when skuId does not exist", () => {
    useCartStore.getState().addItem(item1);
    useCartStore.getState().removeItem("sku-999");
    expect(useCartStore.getState().items).toHaveLength(1);
  });
});

describe("updateQty", () => {
  it("updates the quantity of an existing item", () => {
    useCartStore.getState().addItem(item1);
    useCartStore.getState().updateQty("sku-001", 7);
    const { items } = useCartStore.getState();
    expect(items[0].qty).toBe(7);
  });

  it("removes the item when qty is set to 0", () => {
    useCartStore.getState().addItem(item1);
    useCartStore.getState().updateQty("sku-001", 0);
    expect(useCartStore.getState().items).toHaveLength(0);
  });

  it("removes the item when qty is set to a negative number", () => {
    useCartStore.getState().addItem(item1);
    useCartStore.getState().updateQty("sku-001", -3);
    expect(useCartStore.getState().items).toHaveLength(0);
  });
});

describe("clearCart", () => {
  it("empties the items array", () => {
    useCartStore.getState().addItem(item1);
    useCartStore.getState().addItem(item2);
    useCartStore.getState().clearCart();
    expect(useCartStore.getState().items).toHaveLength(0);
  });

  it("is safe to call on an already empty cart", () => {
    expect(() => useCartStore.getState().clearCart()).not.toThrow();
    expect(useCartStore.getState().items).toHaveLength(0);
  });
});

describe("selectCartTotal()", () => {
  it("returns 0 for an empty cart", () => {
    expect(selectCartTotal(useCartStore.getState().items)).toBe(0);
  });

  it("sums price * qty for a single item (handles cents)", () => {
    useCartStore.getState().addItem({ ...item1, qty: 3 });
    // 1099 cents * 3 = 3297
    expect(selectCartTotal(useCartStore.getState().items)).toBe(3297);
  });

  it("sums price * qty correctly across multiple items", () => {
    useCartStore.getState().addItem({ ...item1, qty: 2 }); // 1099 * 2 = 2198
    useCartStore.getState().addItem({ ...item2, qty: 4 }); // 499 * 4  = 1996
    // total = 4194
    expect(selectCartTotal(useCartStore.getState().items)).toBe(4194);
  });

  it("recalculates total correctly after updateQty", () => {
    useCartStore.getState().addItem({ ...item1, qty: 2 }); // 1099 * 2 = 2198
    useCartStore.getState().updateQty("sku-001", 1);       // 1099 * 1 = 1099
    expect(selectCartTotal(useCartStore.getState().items)).toBe(1099);
  });

  it("recalculates total correctly after removeItem", () => {
    useCartStore.getState().addItem({ ...item1, qty: 1 }); // 1099
    useCartStore.getState().addItem({ ...item2, qty: 1 }); // 499
    useCartStore.getState().removeItem("sku-001");
    expect(selectCartTotal(useCartStore.getState().items)).toBe(499);
  });
});
