const GW = process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";

async function req<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const res = await fetch(`${GW}${path}`, {
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

function auth(token: string): Record<string, string> {
  return { Authorization: `Bearer ${token}` };
}

// ── Auth ─────────────────────────────────────────────────────────────────────

export interface LoginResponse { token: string; }

export async function login(email: string, password: string): Promise<LoginResponse> {
  const r = await req<{ access_token: string }>("/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
  return { token: r.access_token };
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

export async function getMe(token: string): Promise<MeResponse> {
  return req<MeResponse>("/v1/auth/me", { headers: auth(token) });
}

export async function register(
  first_name: string,
  last_name: string,
  email: string,
  password: string,
): Promise<void> {
  return req("/v1/auth/register", {
    method: "POST",
    body: JSON.stringify({ full_name: first_name+" "+last_name, email, password, role: "seller" }),
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
  token: string,
  params: { status?: string; search?: string; limit?: number; offset?: number } = {},
): Promise<ProductsResponse> {
  const q = new URLSearchParams();
  if (params.status)  q.set("status",  params.status);
  if (params.search)  q.set("search",  params.search);
  if (params.limit)   q.set("limit",   String(params.limit));
  if (params.offset)  q.set("offset",  String(params.offset));
  const qs = q.toString() ? `?${q}` : "";
  const r = await req<{ data: Product[]; total: number; page: number; page_size: number }>(
    `/api/v1/products${qs}`, { headers: auth(token) },
  );
  return { products: r.data ?? [], total: r.total ?? 0, limit: r.page_size ?? 20, offset: ((r.page ?? 1) - 1) * (r.page_size ?? 20) };
}

export async function getProduct(token: string, id: string): Promise<Product> {
  const r = await req<{ data: Product }>(`/api/v1/products/${id}`, { headers: auth(token) });
  return r.data;
}

export async function createProduct(
  token: string,
  data: { name: string; slug: string; description: string; category_id: string; status: string },
): Promise<Product> {
  const r = await req<{ data: Product }>("/api/v1/products", { method: "POST", body: JSON.stringify(data), headers: auth(token) });
  return r.data;
}

export async function updateProduct(
  token: string,
  id: string,
  data: Partial<{ name: string; slug: string; description: string; category_id: string; status: string }>,
): Promise<Product> {
  const r = await req<{ data: Product }>(`/api/v1/products/${id}`, { method: "PUT", body: JSON.stringify(data), headers: auth(token) });
  return r.data;
}

export async function deleteProduct(token: string, id: string): Promise<void> {
  return req(`/api/v1/products/${id}`, { method: "DELETE", headers: auth(token) });
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

export async function getSkus(token: string, productId: string): Promise<{ skus: SKU[] }> {
  const r = await req<{ data: SKU[] }>(`/api/v1/skus?product_id=${productId}`, { headers: auth(token) });
  return { skus: r.data ?? [] };
}

export async function createSku(
  token: string,
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
  const r = await req<{ data: SKU }>("/api/v1/skus", { method: "POST", body: JSON.stringify(data), headers: auth(token) });
  return r.data;
}

export async function updateSku(
  token: string,
  id: string,
  data: Partial<Omit<SKU, "id" | "product_id" | "created_at">>,
): Promise<SKU> {
  const r = await req<{ data: SKU }>(`/api/v1/skus/${id}`, { method: "PUT", body: JSON.stringify(data), headers: auth(token) });
  return r.data;
}

export async function deleteSku(token: string, id: string): Promise<void> {
  return req(`/api/v1/skus/${id}`, { method: "DELETE", headers: auth(token) });
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

export async function getImages(token: string, productId: string): Promise<{ images: ProductImage[] }> {
  const r = await req<{ data: ProductImage[] }>(`/api/v1/products/${productId}/images`, { headers: auth(token) });
  return { images: r.data ?? [] };
}

export async function uploadImage(
  token: string,
  productId: string,
  file: File,
  skuId?: string,
): Promise<ProductImage> {
  const fd = new FormData();
  fd.append("image", file);
  if (skuId) fd.append("sku_id", skuId);
  const res = await fetch(`${GW}/api/v1/products/${productId}/images`, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` },
    body: fd,
  });
  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try { const j = await res.json(); msg = j.error ?? j.message ?? msg; } catch { /* */ }
    throw new Error(msg);
  }
  const r = await res.json();
  return r.data ?? r;
}

export async function setImagePosition(
  token: string,
  productId: string,
  imageId: string,
  position: number,
): Promise<void> {
  return req(`/api/v1/products/${productId}/images/${imageId}/position`, {
    method: "PATCH",
    body: JSON.stringify({ position }),
    headers: auth(token),
  });
}

export async function deleteImage(token: string, productId: string, imageId: string): Promise<void> {
  return req(`/api/v1/products/${productId}/images/${imageId}`, { method: "DELETE", headers: auth(token) });
}

// ── Categories ────────────────────────────────────────────────────────────────

export interface Category { id: string; name: string; slug: string; parent_id?: string; }

export async function getCategories(params: {
  search?: string; parent_id?: string; limit?: number; offset?: number;
} = {}): Promise<{ categories: Category[]; total: number }> {
  const q = new URLSearchParams();
  if (params.search)             q.set("search",    params.search);
  if (params.parent_id)          q.set("parent_id", params.parent_id);
  if (params.limit  != null)     q.set("limit",     String(params.limit));
  if (params.offset != null)     q.set("offset",    String(params.offset));
  const qs = q.toString() ? `?${q}` : "";
  const r = await req<{ data: Category[]; total: number }>(`/api/v1/categories${qs}`);
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
  token: string,
  params: { status?: string; from?: string; to?: string; limit?: number; offset?: number } = {},
): Promise<OrdersResponse> {
  const q = new URLSearchParams();
  if (params.status) q.set("status", params.status);
  if (params.from)   q.set("from",   params.from);
  if (params.to)     q.set("to",     params.to);
  if (params.limit)  q.set("limit",  String(params.limit));
  if (params.offset) q.set("offset", String(params.offset));
  const qs = q.toString() ? `?${q}` : "";
  const r = await req<{ data: Order[] | null }>(`/v1/orders/seller${qs}`, { headers: auth(token) });
  const orders = r.data ?? [];
  return { orders, total: orders.length, limit: params.limit ?? 20, offset: params.offset ?? 0 };
}

export async function getSellerOrder(
  token: string,
  id: string,
): Promise<{ order: Order; items: OrderItem[] }> {
  const r = await req<{ data: Order & { items: OrderItem[] } }>(`/v1/orders/seller/${id}`, { headers: auth(token) });
  const { items, ...order } = r.data;
  return { order: order as Order, items: items ?? [] };
}

export async function cancelOrder(token: string, id: string): Promise<Order> {
  const r = await req<{ data: Order }>(`/v1/orders/${id}/cancel`, { method: "POST", headers: auth(token) });
  return r.data;
}
