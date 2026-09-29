export type StockItem = {
  food_id: number;
  code: string;
  name: string;
  unit: string;
  quantity: string;
  category_id?: number;
  category_code?: string;
  category_name?: string;
};

export type PromiseAgg = {
  food_id: number;
  code: string;
  name: string;
  unit: string;
  quantity: string;
};

export type NeedItem = {
  food_id: number;
  code: string;
  name: string;
  unit: string;
  needed: string;
  stock: string;
  promised: string;
  total: string;
  missing: string;
  percent: number;
};

export type CommunityNeed = {
  id: number;
  food_id: number;
  food_name?: string;
  food_unit?: string;
  food_code?: string;
  category_id?: number;
  category_name?: string;
  title: string;
  quantity: string;
  need_kind: "sporadic" | "recurring";
  frequency?: string | null;
  active: boolean;
  starts_on: string;
  ends_on?: string | null;
};

export type Category = {
  id: number;
  code: string;
  name: string;
  sort_order: number;
  active: boolean;
};

export type Food = {
  id: number;
  code: string;
  name: string;
  unit: string;
  category_id?: number | null;
  category_name?: string;
  category_code?: string;
  sort_order: number;
  active: boolean;
};

export type Person = {
  id: number;
  full_name: string;
  notes: string;
  active: boolean;
};

export type PersonNeed = {
  id: number;
  person_id: number;
  food_id: number;
  food_name?: string;
  food_unit?: string;
  category_name?: string;
  quantity: string;
  need_kind: "sporadic" | "recurring";
  frequency?: string | null;
  notes: string;
  active: boolean;
};

export type Delivery = {
  id: number;
  person_id: number;
  person_name?: string;
  delivered_on: string;
  note: string;
  deduct_stock: boolean;
  items?: Array<{ food_id: number; food_name?: string; food_unit?: string; quantity: string }>;
};

export type PublicSettings = {
  name: string;
  description: string;
  logo: string;
  favicon: string;
  primary_color: string;
  secondary_color: string;
  accent_color: string;
  default_locale: string;
};

export type Summary = {
  site: PublicSettings;
  stock: StockItem[];
  available_baskets: number;
  limiting_food?: { code: string; name: string };
  promises_today: PromiseAgg[];
  needs: NeedItem[];
  community_needs?: CommunityNeed[];
  as_of: string;
};

export type PromiseInput = {
  food_id: number;
  person_name: string;
  quantity: string;
  promise_date: string;
};

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    credentials: "same-origin",
    headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) },
    ...init,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error((body as { error?: string }).error || `HTTP ${res.status}`);
  }
  return res.json() as Promise<T>;
}

export function fetchSummary() {
  return request<Summary>("/api/v1/public/summary");
}

export function createPublicPromise(input: PromiseInput) {
  return request<{ id: number; food_id: number; quantity: string; promise_date: string }>(
    "/api/v1/public/promises",
    { method: "POST", body: JSON.stringify(input) },
  );
}

export function login(username: string, password: string) {
  return request<{ user: { id: number; username: string; display_name: string } }>(
    "/api/v1/auth/login",
    { method: "POST", body: JSON.stringify({ username, password }) },
  );
}

export function logout() {
  return request<{ ok: boolean }>("/api/v1/auth/logout", { method: "POST" });
}

export function fetchMe() {
  return request<{ user: { id: number; username: string; display_name: string } }>(
    "/api/v1/auth/me",
  );
}

export function stockIn(foodId: number, quantity: string) {
  return request("/api/v1/admin/stock/in", {
    method: "POST",
    body: JSON.stringify({ food_id: foodId, quantity }),
  });
}

export function stockOut(foodId: number, quantity: string) {
  return request("/api/v1/admin/stock/out", {
    method: "POST",
    body: JSON.stringify({ food_id: foodId, quantity }),
  });
}

export function createPromise(input: PromiseInput) {
  return request("/api/v1/admin/promises", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function listFoods() {
  return request<{ foods: Food[] }>("/api/v1/admin/foods");
}

export function createFood(input: {
  code: string;
  name: string;
  unit: string;
  category_id?: number | null;
  sort_order?: number;
}) {
  return request<Food>("/api/v1/admin/foods", { method: "POST", body: JSON.stringify(input) });
}

export function listCategories() {
  return request<{ categories: Category[] }>("/api/v1/admin/categories");
}

export function createCategory(input: { code: string; name: string; sort_order?: number }) {
  return request<Category>("/api/v1/admin/categories", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function listCommunityNeeds() {
  return request<{ needs: CommunityNeed[] }>("/api/v1/admin/community-needs");
}

export function createCommunityNeed(input: {
  food_id: number;
  title: string;
  quantity: string;
  need_kind: string;
  frequency?: string | null;
  starts_on?: string;
  ends_on?: string | null;
}) {
  return request<CommunityNeed>("/api/v1/admin/community-needs", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function deleteCommunityNeed(id: number) {
  return request<{ ok: boolean }>(`/api/v1/admin/community-needs/${id}`, { method: "DELETE" });
}

export function listPeople() {
  return request<{ people: Person[] }>("/api/v1/admin/people");
}

export function createPerson(input: { full_name: string; notes?: string }) {
  return request<Person>("/api/v1/admin/people", { method: "POST", body: JSON.stringify(input) });
}

export function listPersonNeeds(personId?: number) {
  const q = personId ? `?person_id=${personId}` : "";
  return request<{ needs: PersonNeed[] }>(`/api/v1/admin/person-needs${q}`);
}

export function createPersonNeed(input: {
  person_id: number;
  food_id: number;
  quantity: string;
  need_kind: string;
  frequency?: string | null;
  notes?: string;
}) {
  return request<PersonNeed>("/api/v1/admin/person-needs", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function deletePersonNeed(id: number) {
  return request<{ ok: boolean }>(`/api/v1/admin/person-needs/${id}`, { method: "DELETE" });
}

export function listDeliveries() {
  return request<{ deliveries: Delivery[] }>("/api/v1/admin/deliveries");
}

export function createDelivery(input: {
  person_id: number;
  delivered_on?: string;
  note?: string;
  deduct_stock?: boolean;
  items: Array<{ food_id: number; quantity: string }>;
}) {
  return request<Delivery>("/api/v1/admin/deliveries", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function getBasket() {
  return request<{
    items: Array<{ food_id: number; code: string; name: string; unit: string; quantity: string }>;
    available_baskets: number;
  }>("/api/v1/admin/basket");
}
