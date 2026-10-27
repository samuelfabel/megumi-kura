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
  pickup_frequency?: string | null;
  last_delivered_on?: string | null;
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

export type PublicSettings = {
  name: string;
  description: string;
  logo: string;
  favicon: string;
  primary_color: string;
  secondary_color: string;
  accent_color: string;
  default_locale: string;
  captcha_provider?: string;
  captcha_site_key?: string;
};

export type Kit = {
  id: number;
  code: string;
  name: string;
  description: string;
  icon_media_id?: number | null;
  sort_order: number;
  active: boolean;
  items?: Array<{
    food_id: number;
    code: string;
    name: string;
    unit: string;
    quantity: string;
  }>;
  available?: number;
};

export type ShortageRisk = {
  food_id: number;
  code: string;
  name: string;
  unit: string;
  stock: string;
  promised: string;
  projected_need: string;
  days_to_empty?: number | null;
  alert: boolean;
  severity: string;
};

export type ProjectionSeries = {
  food_id: number;
  code: string;
  name: string;
  unit: string;
  points: Array<{ day: string; balance: string; in: string; out: string; promised: string }>;
};

export type SocialPost = {
  id: number;
  kind: "article" | "testimonial";
  title: string;
  body: string;
  image_media_id?: number | null;
  published: boolean;
  created_at: string;
  comments?: Array<{
    id: number;
    author_name: string;
    body: string;
    status: string;
    created_at: string;
  }>;
};

export type Delivery = {
  id: number;
  person_id: number;
  person_name?: string;
  kit_id?: number | null;
  delivered_on: string;
  note: string;
  deduct_stock: boolean;
  items?: Array<{ food_id: number; food_name?: string; food_unit?: string; quantity: string }>;
};

export type Summary = {
  site: PublicSettings;
  stock: StockItem[];
  available_baskets: number;
  limiting_food?: { code: string; name: string };
  promises_today: PromiseAgg[];
  needs: NeedItem[];
  community_needs?: CommunityNeed[];
  kits?: Kit[];
  shortage_risks?: ShortageRisk[];
  as_of: string;
};

export type PromiseInput = {
  food_id: number;
  person_name: string;
  quantity: string;
  promise_date: string;
  captcha_token?: string;
  captcha_answer?: string;
};

export type PromiseBatchInput = {
  person_name: string;
  promise_date: string;
  kit_id?: number | null;
  items: Array<{ food_id: number; quantity: string }>;
  captcha_token: string;
  captcha_answer: string;
};

export type CaptchaChallenge = {
  provider: string;
  site_key?: string;
  captcha_token?: string;
  question?: string;
};

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    credentials: "same-origin",
    headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) },
    ...init,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new ApiError(
      res.status,
      (body as { error?: string }).error || `HTTP ${res.status}`,
    );
  }
  return res.json() as Promise<T>;
}

export function fetchSummary() {
  return request<Summary>("/api/v1/public/summary");
}

export function fetchCaptcha() {
  return request<CaptchaChallenge>("/api/v1/public/captcha");
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

export function createPerson(input: {
  full_name: string;
  notes?: string;
  pickup_frequency?: string | null;
}) {
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
  kit_id?: number | null;
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

export function createPublicPromiseBatch(input: PromiseBatchInput) {
  return request<{ batch_id: number }>("/api/v1/public/promise-batches", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function fetchProjection(foodId: number, days = 30) {
  return request<ProjectionSeries>(
    `/api/v1/public/projections?food_id=${foodId}&days=${days}`,
  );
}

export function subscribeMail(email: string) {
  return request<{ ok: boolean }>("/api/v1/public/mail/subscribe", {
    method: "POST",
    body: JSON.stringify({ email }),
  });
}

export function listPublicKits() {
  return request<{ kits: Kit[] }>("/api/v1/public/kits");
}

export function listAdminKits() {
  return request<{ kits: Kit[] }>("/api/v1/admin/kits");
}

export function createKit(input: {
  code: string;
  name: string;
  description?: string;
  sort_order?: number;
  icon_media_id?: number | null;
}) {
  return request<Kit>("/api/v1/admin/kits", { method: "POST", body: JSON.stringify(input) });
}

export function putKitItems(id: number, items: Array<{ food_id: number; quantity: string }>) {
  return request<Kit>(`/api/v1/admin/kits/${id}/items`, {
    method: "PUT",
    body: JSON.stringify({ items }),
  });
}

export function listSocialPostsPublic() {
  return request<{ posts: SocialPost[] }>("/api/v1/public/social/posts");
}

export function getSocialPostPublic(id: number) {
  return request<SocialPost>(`/api/v1/public/social/posts/${id}`);
}

export function commentSocialPost(id: number, author_name: string, body: string) {
  return request<{ id: number; status: string }>(`/api/v1/public/social/posts/${id}/comments`, {
    method: "POST",
    body: JSON.stringify({ author_name, body }),
  });
}

export function listSocialAdmin() {
  return request<{
    posts: SocialPost[];
    pending_comments: Array<{
      id: number;
      post_id: number;
      author_name: string;
      body: string;
      status: string;
    }>;
  }>("/api/v1/admin/social/posts");
}

export function createSocialPost(input: {
  kind: string;
  title: string;
  body: string;
  published?: boolean;
  image_media_id?: number | null;
}) {
  return request<SocialPost>("/api/v1/admin/social/posts", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function moderateComment(id: number, status: "approved" | "rejected") {
  return request(`/api/v1/admin/social/comments/${id}/moderate`, {
    method: "POST",
    body: JSON.stringify({ status }),
  });
}

export function listMailTemplates() {
  return request<{
    templates: Array<{
      key: string;
      name: string;
      subject: string;
      body_html: string;
      required_tags: string[];
      optional_tags: string[];
    }>;
  }>("/api/v1/admin/mail/templates");
}

export function updateMailTemplate(key: string, subject: string, body_html: string) {
  return request(`/api/v1/admin/mail/templates/${key}`, {
    method: "PUT",
    body: JSON.stringify({ subject, body_html }),
  });
}

export function getMailSettings() {
  return request<{
    smtp: { host: string; port: string; user: string; from: string };
    alerts: { AutoEnabled?: boolean; auto_enabled?: boolean; AutoFrequency?: string; auto_frequency?: string; GlobalThreshold?: string };
    subscriber_count: number;
  }>("/api/v1/admin/mail/settings");
}

export function putMailSettings(body: unknown) {
  return request("/api/v1/admin/mail/settings", { method: "PUT", body: JSON.stringify(body) });
}

export function sendShortageMail() {
  return request<{ sent: number; shortage_count: number; skipped_reason?: string }>(
    "/api/v1/admin/mail/send-shortage",
    { method: "POST" },
  );
}

export function listUsers() {
  return request<{
    users: Array<{
      id: number;
      username: string;
      email?: string | null;
      display_name: string;
      active: boolean;
      is_bootstrap_admin: boolean;
      permissions: string[];
    }>;
    all_permissions: string[];
  }>("/api/v1/admin/users");
}

export function createUser(input: {
  username: string;
  email: string;
  display_name: string;
  password: string;
  permissions: string[];
}) {
  return request("/api/v1/admin/users", { method: "POST", body: JSON.stringify(input) });
}

export function updateUser(
  id: number,
  body: { permissions: string[]; active?: boolean },
) {
  return request(`/api/v1/admin/users/${id}`, { method: "PUT", body: JSON.stringify(body) });
}

export function uploadMedia(file: File, kind: string) {
  const fd = new FormData();
  fd.append("file", file);
  fd.append("kind", kind);
  return fetch("/api/v1/admin/media", { method: "POST", credentials: "same-origin", body: fd }).then(
    async (res) => {
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new ApiError(res.status, (body as { error?: string }).error || `HTTP ${res.status}`);
      }
      return res.json() as Promise<{ id: number }>;
    },
  );
}
