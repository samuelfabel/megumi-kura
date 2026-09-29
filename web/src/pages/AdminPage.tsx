import { useCallback, useEffect, useMemo, useState, type FormEvent } from "react";
import { t } from "../i18n";
import {
  createCategory,
  createCommunityNeed,
  createDelivery,
  createFood,
  createPerson,
  createPersonNeed,
  createPromise,
  deleteCommunityNeed,
  deletePersonNeed,
  fetchMe,
  getBasket,
  listCategories,
  listCommunityNeeds,
  listDeliveries,
  listFoods,
  listPeople,
  listPersonNeeds,
  login,
  logout,
  stockIn,
  stockOut,
  type Category,
  type CommunityNeed,
  type Delivery,
  type Food,
  type Person,
  type PersonNeed,
} from "../services/api";

type Props = { onBack: () => void };
type Tab = "stock" | "catalog" | "needs" | "people" | "deliveries";

function todayISO() {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

export function AdminPage({ onBack }: Props) {
  const [user, setUser] = useState<string | null>(null);
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("");
  const [tab, setTab] = useState<Tab>("people");
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  const [foods, setFoods] = useState<Food[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [people, setPeople] = useState<Person[]>([]);
  const [personNeeds, setPersonNeeds] = useState<PersonNeed[]>([]);
  const [communityNeeds, setCommunityNeeds] = useState<CommunityNeed[]>([]);
  const [deliveries, setDeliveries] = useState<Delivery[]>([]);
  const [basketItems, setBasketItems] = useState<
    Array<{ food_id: number; name: string; unit: string; quantity: string }>
  >([]);

  const refresh = useCallback(async () => {
    const [f, c, p, pn, cn, d, b] = await Promise.all([
      listFoods(),
      listCategories(),
      listPeople(),
      listPersonNeeds(),
      listCommunityNeeds(),
      listDeliveries(),
      getBasket(),
    ]);
    setFoods(f.foods);
    setCategories(c.categories);
    setPeople(p.people);
    setPersonNeeds(pn.needs);
    setCommunityNeeds(cn.needs);
    setDeliveries(d.deliveries);
    setBasketItems(b.items);
  }, []);

  useEffect(() => {
    void fetchMe()
      .then(async (r) => {
        setUser(r.user.username);
        await refresh();
      })
      .catch(() => setUser(null));
  }, [refresh]);

  async function onLogin(e: FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      const r = await login(username, password);
      setUser(r.user.username);
      setPassword("");
      await refresh();
    } catch {
      setError("invalid");
    }
  }

  async function onLogout() {
    await logout();
    setUser(null);
  }

  return (
    <div className="page admin-shell">
      <header className="admin-top">
        <div>
          <p className="admin-brand">{t("brand.name")}</p>
          <p className="admin-sub">{t("admin.console")}</p>
        </div>
        <div className="admin-top-actions">
          {user && <span className="admin-user">{user}</span>}
          {user && (
            <button type="button" className="btn-ghost" onClick={() => void onLogout()}>
              {t("admin.logout")}
            </button>
          )}
          <button type="button" className="btn-ghost" onClick={onBack}>
            {t("admin.back_public")}
          </button>
        </div>
      </header>

      {!user ? (
        <form className="admin-form login-card" onSubmit={onLogin}>
          <h2>{t("admin.login_title")}</h2>
          <label>
            {t("admin.username")}
            <input value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="username" />
          </label>
          <label>
            {t("admin.password")}
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
            />
          </label>
          {error && <p className="error-text">Login failed</p>}
          <button type="submit" className="btn-primary">
            {t("admin.login")}
          </button>
        </form>
      ) : (
        <div className="admin-layout">
          <nav className="admin-nav" aria-label="Admin">
            {(
              [
                ["people", t("admin.tab_people")],
                ["deliveries", t("admin.tab_deliveries")],
                ["needs", t("admin.tab_needs")],
                ["catalog", t("admin.tab_catalog")],
                ["stock", t("admin.tab_stock")],
              ] as Array<[Tab, string]>
            ).map(([id, label]) => (
              <button
                key={id}
                type="button"
                className={tab === id ? "nav-item active" : "nav-item"}
                onClick={() => {
                  setTab(id);
                  setMessage(null);
                  setError(null);
                }}
              >
                {label}
              </button>
            ))}
          </nav>

          <div className="admin-main">
            {message && <p className="ok-text banner">{message}</p>}
            {error && <p className="error-text banner">{error}</p>}

            {tab === "people" && (
              <PeoplePanel
                people={people}
                foods={foods}
                needs={personNeeds}
                onRefresh={() => void refresh().then(() => setMessage(t("admin.saved")))}
                onError={setError}
              />
            )}
            {tab === "deliveries" && (
              <DeliveriesPanel
                people={people}
                foods={foods}
                basketItems={basketItems}
                deliveries={deliveries}
                onRefresh={() => void refresh().then(() => setMessage(t("admin.saved")))}
                onError={setError}
              />
            )}
            {tab === "needs" && (
              <NeedsPanel
                foods={foods}
                needs={communityNeeds}
                onRefresh={() => void refresh().then(() => setMessage(t("admin.saved")))}
                onError={setError}
              />
            )}
            {tab === "catalog" && (
              <CatalogPanel
                categories={categories}
                foods={foods}
                onRefresh={() => void refresh().then(() => setMessage(t("admin.saved")))}
                onError={setError}
              />
            )}
            {tab === "stock" && (
              <StockPanel
                foods={foods}
                onRefresh={() => void refresh().then(() => setMessage(t("admin.saved")))}
                onError={setError}
              />
            )}
          </div>
        </div>
      )}
    </div>
  );
}
