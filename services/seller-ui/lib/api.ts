const GW = process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000";

async function req<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const res = await fetch(`${GW}${path}`, {
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
  return req("/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
}

export async function register(
  first_name: string,
  last_name: string,
  email: string,
  password: string,
): Promise<void> {
  return req("/v1/auth/register", {
    method: "POST",
    body: JSON.stringify({ first_name, last_name, email, password, role: "seller" }),
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
  return req(`/v1/products${qs}`, { headers: auth(token) });
}

export async function getProduct(token: string, id: string): Promise<Product> {
  return req(`/v1/products/${id}`, { headers: auth(token) });
}

export async function createProduct(
  token: string,
  data: { name: string; slug: string; description: string; category_id: string; status: string },
): Promise<Product> {
  return req("/v1/products", { method: "POST", body: JSON.stringify(data), headers: auth(token) });
}

export async function updateProduct(
  token: string,
  id: string,
  data: Partial<{ name: string; slug: string; description: string; category_id: string; status: string }>,
): Promise<Product> {
  return req(`/v1/products/${id}`, { method: "PUT", body: JSON.stringify(data), headers: auth(token) });
}

export async function deleteProduct(token: string, id: string): Promise<void> {
  return req(`/v1/products/${id}`, { method: "DELETE", headers: auth(token) });
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
  return req(`/v1/products/${productId}/skus`, { headers: auth(token) });
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
  return req("/v1/skus", { method: "POST", body: JSON.stringify(data), headers: auth(token) });
}

export async function updateSku(
  token: string,
  id: string,
  data: Partial<Omit<SKU, "id" | "product_id" | "created_at">>,
): Promise<SKU> {
  return req(`/v1/skus/${id}`, { method: "PUT", body: JSON.stringify(data), headers: auth(token) });
}

export async function deleteSku(token: string, id: string): Promise<void> {
  return req(`/v1/skus/${id}`, { method: "DELETE", headers: auth(token) });
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
  return req(`/v1/products/${productId}/images`, { headers: auth(token) });
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
  const res = await fetch(`${GW}/v1/products/${productId}/images`, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` },
    body: fd,
  });
  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try { const j = await res.json(); msg = j.error ?? j.message ?? msg; } catch { /* */ }
    throw new Error(msg);
  }
  return res.json();
}

export async function setImagePosition(
  token: string,
  productId: string,
  imageId: string,
  position: number,
): Promise<void> {
  return req(`/v1/products/${productId}/images/${imageId}`, {
    method: "PATCH",
    body: JSON.stringify({ position }),
    headers: auth(token),
  });
}

export async function deleteImage(token: string, productId: string, imageId: string): Promise<void> {
  return req(`/v1/products/${productId}/images/${imageId}`, { method: "DELETE", headers: auth(token) });
}

// ── Categories ────────────────────────────────────────────────────────────────

export interface Category { id: string; name: string; slug: string; }

export async function getCategories(): Promise<{ categories: Category[] }> {
  return req("/v1/categories");
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
  return req(`/v1/orders/seller${qs}`, { headers: auth(token) });
}

export async function getSellerOrder(
  token: string,
  id: string,
): Promise<{ order: Order; items: OrderItem[] }> {
  return req(`/v1/orders/seller/${id}`, { headers: auth(token) });
}

export async function cancelOrder(token: string, id: string): Promise<Order> {
  return req(`/v1/orders/${id}/cancel`, { method: "POST", headers: auth(token) });
}
