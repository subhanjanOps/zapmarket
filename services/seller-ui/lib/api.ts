// All authenticated requests proxy through /api/proxy/[...path] which reads
// the httpOnly seller_token cookie and adds Authorization server-side.

async function req<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const res = await fetch(path, {
    cache: "no-store",
    ...opts,
    headers: { "Content-Type": "application/json", ...opts.headers },
  });
  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try { const j = await res.json(); msg = j.error ?? j.message ?? msg; } catch { /* */ }
    throw new Error(msg);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

// ── Auth ─────────────────────────────────────────────────────────────────────

export async function login(email: string, password: string): Promise<void> {
  const res = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
    cache: "no-store",
  });
  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try { const j = await res.json(); msg = j.error ?? j.message ?? msg; } catch { /* */ }
    throw new Error(msg);
  }
}

export interface MeResponse {
  user: {
    id: string;
    email: string;
    full_name: string;
    role: string;
    is_verified: boolean;
    seller_status?: string;
    created_at: string;
  };
}

export async function getMe(): Promise<MeResponse> {
  return req<MeResponse>("/api/auth/me");
}

export async function register(
  first_name: string,
  last_name: string,
  email: string,
  password: string,
): Promise<void> {
  return req("/api/proxy/v1/auth/register", {
    method: "POST",
    body: JSON.stringify({ full_name: first_name + " " + last_name, email, password, role: "seller" }),
  });
}

// ── Products ─────────────────────────────────────────────────────────────────

export interface Product {
  id: string;
  name: string;
  slug: string;
  description: string;
  category_id: string;
  category_name?: string;
  status: "ACTIVE" | "DRAFT" | "ARCHIVED";
  seller_id: string;
  created_at: string;
  updated_at: string;
  images?: ProductImage[];
  skus?: SKU[];
}

export interface ProductsResponse {
  products: Product[];
  total: number;
  limit: number;
  offset: number;
}

export async function getProducts(
  params: { status?: string; search?: string; limit?: number; offset?: number } = {},
): Promise<ProductsResponse> {
  const q = new URLSearchParams();
  if (params.status)  q.set("status",  params.status);
  if (params.search)  q.set("search",  params.search);
  if (params.limit)   q.set("limit",   String(params.limit));
  if (params.offset)  q.set("offset",  String(params.offset));
  const qs = q.toString() ? `?${q}` : "";
  const r = await req<{ data: Product[]; total: number; page: number; page_size: number }>(
    `/api/proxy/api/v1/products${qs}`,
  );
  return { products: r.data ?? [], total: r.total ?? 0, limit: r.page_size ?? 20, offset: ((r.page ?? 1) - 1) * (r.page_size ?? 20) };
}

export async function getProduct(id: string): Promise<Product> {
  const r = await req<{ data: Product }>(`/api/proxy/api/v1/products/${id}`);
  return r.data;
}

export async function createProduct(
  data: { name: string; slug: string; description: string; category_id: string; status: string },
): Promise<Product> {
  const r = await req<{ data: Product }>("/api/proxy/api/v1/products", { method: "POST", body: JSON.stringify(data) });
  return r.data;
}

export async function updateProduct(
  id: string,
  data: Partial<{ name: string; slug: string; description: string; category_id: string; status: string }>,
): Promise<Product> {
  const r = await req<{ data: Product }>(`/api/proxy/api/v1/products/${id}`, { method: "PUT", body: JSON.stringify(data) });
  return r.data;
}

export async function deleteProduct(id: string): Promise<void> {
  return req(`/api/proxy/api/v1/products/${id}`, { method: "DELETE" });
}

// ── SKUs ──────────────────────────────────────────────────────────────────────

export interface SKU {
  id: string;
  product_id: string;
  sku_code: string;
  attributes: Record<string, string>;
  price_amount: number;
  price_currency: string;
  compare_price?: number;
  weight_grams?: number;
  is_active: boolean;
  created_at: string;
}

export async function getSkus(productId: string): Promise<{ skus: SKU[] }> {
  const r = await req<{ data: SKU[] }>(`/api/proxy/api/v1/skus?product_id=${productId}`);
  return { skus: r.data ?? [] };
}

export async function createSku(
  data: {
    product_id: string;
    sku_code: string;
    attributes: Record<string, string>;
    price_amount: number;
    price_currency: string;
    compare_price?: number;
    weight_grams?: number;
    is_active: boolean;
  },
): Promise<SKU> {
  const r = await req<{ data: SKU }>("/api/proxy/api/v1/skus", { method: "POST", body: JSON.stringify(data) });
  return r.data;
}

export async function updateSku(
  id: string,
  data: Partial<Omit<SKU, "id" | "product_id" | "created_at">>,
): Promise<SKU> {
  const r = await req<{ data: SKU }>(`/api/proxy/api/v1/skus/${id}`, { method: "PUT", body: JSON.stringify(data) });
  return r.data;
}

export async function deleteSku(id: string): Promise<void> {
  return req(`/api/proxy/api/v1/skus/${id}`, { method: "DELETE" });
}

// ── Images ────────────────────────────────────────────────────────────────────

export interface ProductImage {
  id: string;
  product_id: string;
  sku_id?: string;
  url: string;
  position: number;
  created_at: string;
}

export async function getImages(productId: string): Promise<{ images: ProductImage[] }> {
  const r = await req<{ data: ProductImage[] }>(`/api/proxy/api/v1/products/${productId}/images`);
  return { images: r.data ?? [] };
}

export async function uploadImage(productId: string, file: File, skuId?: string): Promise<ProductImage> {
  const fd = new FormData();
  fd.append("file", file);
  if (skuId) fd.append("sku_id", skuId);
  // No Content-Type — browser sets multipart/form-data with boundary automatically.
  const res = await fetch(`/api/proxy/api/v1/products/${productId}/images`, { method: "POST", body: fd, cache: "no-store" });
  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try { const j = await res.json(); msg = j.error ?? j.message ?? msg; } catch { /* */ }
    throw new Error(msg);
  }
  const r = await res.json();
  return r.data ?? r;
}

export async function setImagePosition(productId: string, imageId: string, position: number): Promise<void> {
  return req(`/api/proxy/api/v1/products/${productId}/images/${imageId}/position`, { method: "PATCH", body: JSON.stringify({ position }) });
}

export async function deleteImage(productId: string, imageId: string): Promise<void> {
  return req(`/api/proxy/api/v1/products/${productId}/images/${imageId}`, { method: "DELETE" });
}

// ── Categories ────────────────────────────────────────────────────────────────

export interface Category { id: string; name: string; slug: string; parent_id?: string; }

export async function getCategories(params: {
  search?: string; parent_id?: string; limit?: number; offset?: number;
} = {}): Promise<{ categories: Category[]; total: number }> {
  const q = new URLSearchParams();
  if (params.search)         q.set("search",    params.search);
  if (params.parent_id)      q.set("parent_id", params.parent_id);
  if (params.limit  != null) q.set("limit",     String(params.limit));
  if (params.offset != null) q.set("offset",    String(params.offset));
  const qs = q.toString() ? `?${q}` : "";
  const r = await req<{ data: Category[]; total: number }>(`/api/proxy/api/v1/categories${qs}`);
  return { categories: r.data ?? [], total: r.total ?? 0 };
}

// ── Orders (seller) ───────────────────────────────────────────────────────────

export interface OrderItem {
  id: string;
  order_id: string;
  sku_id: string;
  sku_code?: string;
  product_name?: string;
  quantity: number;
  unit_price: number;
  currency: string;
}

export interface Order {
  id: string;
  user_id: string;
  status: "PENDING" | "RESERVED" | "CONFIRMED" | "CANCELLED";
  total_amount: number;
  currency: string;
  idempotency_key: string;
  created_at: string;
  updated_at: string;
  items?: OrderItem[];
}

export interface OrdersResponse {
  orders: Order[];
  total: number;
  limit: number;
  offset: number;
}

export async function getSellerOrders(
  params: { status?: string; from?: string; to?: string; limit?: number; offset?: number } = {},
): Promise<OrdersResponse> {
  const q = new URLSearchParams();
  if (params.status) q.set("status", params.status);
  if (params.from)   q.set("from",   params.from);
  if (params.to)     q.set("to",     params.to);
  if (params.limit)  q.set("limit",  String(params.limit));
  if (params.offset) q.set("offset", String(params.offset));
  const qs = q.toString() ? `?${q}` : "";
  const r = await req<{ data: Order[] | null; total?: number; page?: number; page_size?: number }>(
    `/api/proxy/v1/orders/seller${qs}`,
  );
  const orders = r.data ?? [];
  return {
    orders,
    total: r.total ?? orders.length,
    limit: r.page_size ?? params.limit ?? 20,
    offset: r.page != null ? (r.page - 1) * (r.page_size ?? 20) : (params.offset ?? 0),
  };
}

export async function getSellerOrder(id: string): Promise<{ order: Order; items: OrderItem[] }> {
  const r = await req<{ data: Order & { items: OrderItem[] } }>(`/api/proxy/v1/orders/seller/${id}`);
  const { items, ...order } = r.data;
  return { order: order as Order, items: items ?? [] };
}

export async function cancelOrder(id: string): Promise<Order> {
  const r = await req<{ data: Order }>(`/api/proxy/v1/orders/${id}/cancel`, { method: "POST" });
  return r.data;
}
