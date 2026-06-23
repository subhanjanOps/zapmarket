// All authenticated requests proxy through /api/proxy/[...path] which reads
// the httpOnly bo_token cookie and adds Authorization server-side.

async function req<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const res = await fetch(`/api/proxy${path}`, {
    cache: "no-store",
    ...opts,
    headers: { "Content-Type": "application/json", ...opts.headers },
  });
  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try { const j = await res.json(); msg = j.message ?? j.error ?? msg; } catch { /* */ }
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

// ── Types ─────────────────────────────────────────────────────────────────────

export interface Category {
  id: string;
  name: string;
  slug: string;
  parent_id?: string;
  created_at: string;
  updated_at: string;
}

export interface Product {
  id: string;
  category_id: string;
  seller_id: string;
  name: string;
  slug: string;
  description?: string;
  attributes?: Record<string, unknown>;
  status: "DRAFT" | "ACTIVE" | "ARCHIVED";
  created_at: string;
  updated_at: string;
}

export interface SKU {
  id: string;
  product_id: string;
  sku_code: string;
  variant_attributes?: Record<string, unknown>;
  price_amount: number;
  compare_price?: number;
  currency: string;
  weight_grams?: number;
  is_active: boolean;
  created_at: string;
}

export interface ProductImage {
  id: string;
  product_id: string;
  sku_id?: string;
  url: string;
  position: number;
  created_at: string;
}

// ── Paginated list helper ─────────────────────────────────────────────────────

interface PageEnvelope<T> {
  data: T[];
  total: number;
  page: number;
  page_size: number;
}

async function listReq<T>(path: string): Promise<PageEnvelope<T>> {
  const r = await req<PageEnvelope<T>>(path);
  return { ...r, data: r.data ?? [] };
}

// ── Categories ────────────────────────────────────────────────────────────────

export const getCategories = (params: {
  search?: string; parent_id?: string | null; root_only?: boolean;
  limit?: number; offset?: number;
  sort_by?: string; sort_order?: "asc" | "desc";
} = {}) => {
  const q = new URLSearchParams();
  if (params.search)          q.set("search",     params.search);
  if (params.parent_id)       q.set("parent_id",  params.parent_id);
  if (params.root_only)       q.set("root_only",  "true");
  if (params.limit  != null)  q.set("limit",      String(params.limit));
  if (params.offset != null)  q.set("offset",     String(params.offset));
  if (params.sort_by)         q.set("sort_by",    params.sort_by);
  if (params.sort_order)      q.set("sort_order", params.sort_order);
  const qs = q.toString() ? `?${q}` : "";
  return listReq<Category>(`/v1/categories${qs}`);
};

export const getCategory = (id: string) =>
  req<{ data: Category }>(`/v1/categories/${id}`).then((r) => r.data);

export const createCategory = (
  body: { name: string; slug: string; parent_id?: string },
) => req<{ data: Category }>("/v1/categories", {
  method: "POST", body: JSON.stringify(body),
}).then((r) => r.data);

export const bulkCreateCategories = (
  categories: { name: string; slug: string; parent_name?: string }[],
) => req<{ data: Category[] }>("/v1/categories/bulk", {
  method: "POST", body: JSON.stringify({ categories }),
}).then((r) => r.data ?? []);

export const updateCategory = (
  id: string,
  body: { name: string; slug: string; parent_id?: string },
) => req<{ data: Category }>(`/v1/categories/${id}`, {
  method: "PUT", body: JSON.stringify(body),
}).then((r) => r.data);

export const deleteCategory = (id: string) =>
  req(`/v1/categories/${id}`, { method: "DELETE" });

// ── Products ──────────────────────────────────────────────────────────────────

export interface ProductParams {
  status?: string;
  search?: string;
  category_id?: string;
  seller_id?: string;
  sort_by?: string;
  sort_order?: string;
  limit?: number;
  offset?: number;
}

export const getProducts = (params: ProductParams = {}) => {
  const q = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => v !== undefined && q.set(k, String(v)));
  const qs = q.toString() ? `?${q}` : "";
  return listReq<Product>(`/v1/products${qs}`);
};

export const getProduct = (id: string) =>
  req<{ data: Product }>(`/v1/products/${id}`).then((r) => r.data);

export const createProduct = (
  body: { name: string; slug: string; category_id: string; seller_id: string; description?: string; status?: string },
) => req<{ data: Product }>("/v1/products", {
  method: "POST", body: JSON.stringify(body),
}).then((r) => r.data);

export const updateProduct = (
  id: string,
  body: Partial<{ name: string; slug: string; description: string; category_id: string; status: string }>,
) => req<{ data: Product }>(`/v1/products/${id}`, {
  method: "PUT", body: JSON.stringify(body),
}).then((r) => r.data);

export const deleteProduct = (id: string) =>
  req(`/v1/products/${id}`, { method: "DELETE" });

// ── SKUs ──────────────────────────────────────────────────────────────────────

export interface SKUParams {
  product_id?: string;
  sku_code?: string;
  is_active?: boolean;
  limit?: number;
  offset?: number;
}

export const getSkus = (params: SKUParams = {}) => {
  const q = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => v !== undefined && q.set(k, String(v)));
  const qs = q.toString() ? `?${q}` : "";
  return listReq<SKU>(`/v1/skus${qs}`);
};

export const getSku = (id: string) =>
  req<{ data: SKU }>(`/v1/skus/${id}`).then((r) => r.data);

export const createSku = (
  body: {
    product_id: string;
    sku_code: string;
    price_amount: number;
    currency: string;
    is_active: boolean;
    compare_price?: number;
    weight_grams?: number;
    variant_attrs?: Record<string, unknown>;
  },
) => req<{ data: SKU }>("/v1/skus", {
  method: "POST", body: JSON.stringify(body),
}).then((r) => r.data);

export const updateSku = (
  id: string,
  body: Partial<{
    sku_code: string;
    price_amount: number;
    currency: string;
    is_active: boolean;
    compare_price: number;
    weight_grams: number;
    variant_attrs: Record<string, unknown>;
  }>,
) => req<{ data: SKU }>(`/v1/skus/${id}`, {
  method: "PUT", body: JSON.stringify(body),
}).then((r) => r.data);

export const deleteSku = (id: string) =>
  req(`/v1/skus/${id}`, { method: "DELETE" });

// ── Admin: Users ─────────────────────────────────────────────────────────────

export interface AdminUser {
  id: string;
  email: string;
  full_name: string;
  role: "buyer" | "seller" | "admin";
  is_verified: boolean;
  seller_status?: string;
  created_at: string;
}

export interface UserParams {
  role?: string;
  search?: string;
  limit?: number;
  offset?: number;
}

export const adminListUsers = (params: UserParams = {}) => {
  const q = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => v !== undefined && q.set(k, String(v)));
  const qs = q.toString() ? `?${q}` : "";
  return listReq<AdminUser>(`/v1/admin/users${qs}`);
};

export const adminGetUser = (id: string) =>
  req<{ data: AdminUser }>(`/v1/admin/users/${id}`).then((r) => r.data);

export const adminUpdateUserRole = (id: string, role: string) =>
  req<{ data: AdminUser }>(`/v1/admin/users/${id}/role`, {
    method: "PUT", body: JSON.stringify({ role }),
  }).then((r) => r.data);

export const adminDeactivateUser = (id: string) =>
  req(`/v1/admin/users/${id}`, { method: "DELETE" });

// ── Admin: Sellers ────────────────────────────────────────────────────────────

export const adminListSellers = (status?: string, limit = 20, offset = 0) => {
  const q = new URLSearchParams({ limit: String(limit), offset: String(offset) });
  if (status) q.set("status", status);
  return listReq<AdminUser>(`/v1/admin/sellers?${q}`);
};

export const adminUpdateSellerStatus = (id: string, status: string) =>
  req(`/v1/admin/sellers/${id}/status`, {
    method: "PATCH", body: JSON.stringify({ status }),
  });

// ── Admin: Orders ─────────────────────────────────────────────────────────────

export interface AdminOrder {
  id: string;
  user_id: string;
  idempotency_key: string;
  status: string;
  total_amount: number;
  currency: string;
  payment_id?: string;
  created_at: string;
  updated_at: string;
}

export interface OrderItem {
  id: string;
  order_id: string;
  sku_id: string;
  seller_id?: string;
  quantity: number;
  unit_price: number;
  reservation_id?: string;
  created_at: string;
}

export interface AdminOrderParams {
  status?: string;
  user_id?: string;
  from?: string;
  to?: string;
  limit?: number;
  offset?: number;
}

export const adminListOrders = (params: AdminOrderParams = {}) => {
  const q = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => v !== undefined && q.set(k, String(v)));
  const qs = q.toString() ? `?${q}` : "";
  return listReq<AdminOrder>(`/v1/admin/orders${qs}`);
};

export const adminGetOrder = (id: string) =>
  req<{ data: AdminOrder & { items: OrderItem[] } }>(`/v1/admin/orders/${id}`).then((r) => r.data);

export const adminCancelOrder = (id: string) =>
  req(`/v1/admin/orders/${id}/cancel`, { method: "POST" });

// ── Images ────────────────────────────────────────────────────────────────────

export const getImages = (productId: string) =>
  listReq<ProductImage>(`/v1/products/${productId}/images`);

export const deleteImage = (productId: string, imageId: string) =>
  req(`/v1/products/${productId}/images/${imageId}`, { method: "DELETE" });
