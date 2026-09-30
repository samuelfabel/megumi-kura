import { useEffect, useState } from "react";
import { t } from "../../i18n";
import {
  createPromise,
  stockIn,
  stockOut,
  type Food,
} from "../../services/api";
import { todayISO } from "./todayISO";

export function StockPanel({
  foods,
  onRefresh,
  onError,
}: {
  foods: Food[];
  onRefresh: () => void;
  onError: (s: string) => void;
}) {
  const [foodId, setFoodId] = useState(0);
  const [qty, setQty] = useState("1");
  const [person, setPerson] = useState("");

  useEffect(() => {
    if (foods[0] && !foodId) setFoodId(foods[0].id);
  }, [foods, foodId]);

  return (
    <div className="panel-grid single">
      <section className="panel-card">
        <h2>{t("admin.tab_stock")}</h2>
        <div className="stack-form">
          <label>
            {t("admin.item")}
            <select value={foodId} onChange={(e) => setFoodId(Number(e.target.value))}>
              {foods.map((f) => (
                <option key={f.id} value={f.id}>
                  {f.name} ({f.unit})
                </option>
              ))}
            </select>
          </label>
          <label>
            {t("admin.quantity")}
            <input value={qty} onChange={(e) => setQty(e.target.value)} />
          </label>
          <div className="btn-row">
            <button type="button" className="btn-primary" onClick={() => void stockIn(foodId, qty).then(onRefresh).catch((e: Error) => onError(e.message))}>
              Stock IN
            </button>
            <button type="button" className="btn-primary" onClick={() => void stockOut(foodId, qty).then(onRefresh).catch((e: Error) => onError(e.message))}>
              Stock OUT
            </button>
          </div>
          <label>
            {t("promise.person")}
            <input value={person} onChange={(e) => setPerson(e.target.value)} />
          </label>
          <button
            type="button"
            className="btn-primary"
            onClick={() =>
              void createPromise({
                food_id: foodId,
                person_name: person,
                quantity: qty,
                promise_date: todayISO(),
              })
                .then(() => {
                  setPerson("");
                  onRefresh();
                })
                .catch((e: Error) => onError(e.message))
            }
          >
            {t("promise.submit")}
          </button>
        </div>
      </section>
    </div>
  );
}
