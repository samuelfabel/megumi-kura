import { useEffect, useMemo, useState } from "react";
import { t } from "../../i18n";
import {
  createCategory,
  createFood,
  type Category,
  type Food,
} from "../../services/api";

export function CatalogPanel({
  categories,
  foods,
  onRefresh,
  onError,
}: {
  categories: Category[];
  foods: Food[];
  onRefresh: () => void;
  onError: (s: string) => void;
}) {
  const [catCode, setCatCode] = useState("");
  const [catName, setCatName] = useState("");
  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [unit, setUnit] = useState("un");
  const [categoryId, setCategoryId] = useState(0);

  useEffect(() => {
    if (categories[0] && !categoryId) setCategoryId(categories[0].id);
  }, [categories, categoryId]);

  const byCategory = useMemo(() => {
    const map = new Map<string, Food[]>();
    for (const f of foods) {
      const key = f.category_name || t("admin.uncategorized");
      const list = map.get(key) ?? [];
      list.push(f);
      map.set(key, list);
    }
    return [...map.entries()];
  }, [foods]);

  return (
    <div className="panel-grid">
      <section className="panel-card">
        <h2>{t("admin.categories_title")}</h2>
        <form
          className="stack-form"
          onSubmit={(e) => {
            e.preventDefault();
            void createCategory({ code: catCode, name: catName })
              .then(() => {
                setCatCode("");
                setCatName("");
                onRefresh();
              })
              .catch((err: Error) => onError(err.message));
          }}
        >
          <label>
            {t("admin.code")}
            <input value={catCode} onChange={(e) => setCatCode(e.target.value)} required />
          </label>
          <label>
            {t("admin.name")}
            <input value={catName} onChange={(e) => setCatName(e.target.value)} required />
          </label>
          <button type="submit" className="btn-primary">
            {t("admin.add_category")}
          </button>
        </form>
        <ul className="simple-list">
          {categories.map((c) => (
            <li key={c.id}>
              <strong>{c.name}</strong> <span className="muted tiny">{c.code}</span>
            </li>
          ))}
        </ul>
      </section>

      <section className="panel-card">
        <h2>{t("admin.items_title")}</h2>
        <form
          className="stack-form"
          onSubmit={(e) => {
            e.preventDefault();
            void createFood({ code, name, unit, category_id: categoryId || null })
              .then(() => {
                setCode("");
                setName("");
                onRefresh();
              })
              .catch((err: Error) => onError(err.message));
          }}
        >
          <label>
            {t("admin.code")}
            <input value={code} onChange={(e) => setCode(e.target.value)} required />
          </label>
          <label>
            {t("admin.name")}
            <input value={name} onChange={(e) => setName(e.target.value)} required />
          </label>
          <div className="promise-row">
            <label>
              {t("admin.unit")}
              <select value={unit} onChange={(e) => setUnit(e.target.value)}>
                <option value="kg">kg</option>
                <option value="g">g</option>
                <option value="L">L</option>
                <option value="un">un</option>
              </select>
            </label>
            <label>
              {t("admin.category")}
              <select value={categoryId} onChange={(e) => setCategoryId(Number(e.target.value))}>
                {categories.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <button type="submit" className="btn-primary">
            {t("admin.add_item")}
          </button>
        </form>
        {byCategory.map(([cat, items]) => (
          <div key={cat} className="catalog-group">
            <h3>{cat}</h3>
            <ul className="tiny-list">
              {items.map((f) => (
                <li key={f.id}>
                  {f.name} <span className="muted">({f.unit})</span>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </section>
    </div>
  );
}
