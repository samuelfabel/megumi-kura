import { useEffect, useMemo, useState } from "react";
import { t } from "../../i18n";
import {
  createDelivery,
  type Delivery,
  type Food,
  type Person,
} from "../../services/api";
import { todayISO } from "./todayISO";

export function DeliveriesPanel({
  people,
  foods,
  basketItems,
  deliveries,
  onRefresh,
  onError,
}: {
  people: Person[];
  foods: Food[];
  basketItems: Array<{ food_id: number; name: string; unit: string; quantity: string }>;
  deliveries: Delivery[];
  onRefresh: () => void;
  onError: (s: string) => void;
}) {
  const [personId, setPersonId] = useState(0);
  const [date, setDate] = useState(todayISO());
  const [note, setNote] = useState("");
  const [deduct, setDeduct] = useState(true);
  const [useBasket, setUseBasket] = useState(true);
  const [foodId, setFoodId] = useState(0);
  const [qty, setQty] = useState("1");
  const [extraItems, setExtraItems] = useState<Array<{ food_id: number; quantity: string; label: string }>>([]);

  useEffect(() => {
    if (people[0] && !personId) setPersonId(people[0].id);
    if (foods[0] && !foodId) setFoodId(foods[0].id);
  }, [people, foods, personId, foodId]);

  const itemsPayload = useMemo(() => {
    const base = useBasket
      ? basketItems.map((i) => ({ food_id: i.food_id, quantity: i.quantity }))
      : [];
    return [...base, ...extraItems.map((i) => ({ food_id: i.food_id, quantity: i.quantity }))];
  }, [useBasket, basketItems, extraItems]);

  return (
    <div className="panel-grid">
      <section className="panel-card">
        <h2>{t("admin.delivery_title")}</h2>
        <p className="section-lead">{t("admin.delivery_lead")}</p>
        <form
          className="stack-form"
          onSubmit={(e) => {
            e.preventDefault();
            if (itemsPayload.length === 0) {
              onError(t("admin.delivery_empty"));
              return;
            }
            void createDelivery({
              person_id: personId,
              delivered_on: date,
              note,
              deduct_stock: deduct,
              items: itemsPayload,
            })
              .then(() => {
                setExtraItems([]);
                setNote("");
                onRefresh();
              })
              .catch((err: Error) => onError(err.message));
          }}
        >
          <label>
            {t("admin.person")}
            <select value={personId} onChange={(e) => setPersonId(Number(e.target.value))}>
              {people.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.full_name}
                </option>
              ))}
            </select>
          </label>
          <label>
            {t("admin.delivery_date")}
            <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </label>
          <label className="check-row">
            <input type="checkbox" checked={useBasket} onChange={(e) => setUseBasket(e.target.checked)} />
            {t("admin.include_basket")}
          </label>
          {useBasket && (
            <ul className="tiny-list">
              {basketItems.map((i) => (
                <li key={i.food_id}>
                  {i.name}: {i.quantity} {i.unit}
                </li>
              ))}
            </ul>
          )}
          <div className="promise-row">
            <label>
              {t("admin.item")}
              <select value={foodId} onChange={(e) => setFoodId(Number(e.target.value))}>
                {foods.map((f) => (
                  <option key={f.id} value={f.id}>
                    {f.name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              {t("admin.quantity")}
              <input value={qty} onChange={(e) => setQty(e.target.value)} />
            </label>
          </div>
          <button
            type="button"
            className="btn-ghost"
            onClick={() => {
              const f = foods.find((x) => x.id === foodId);
              if (!f) return;
              setExtraItems((prev) => [...prev, { food_id: foodId, quantity: qty, label: `${f.name} ${qty} ${f.unit}` }]);
            }}
          >
            {t("admin.add_item_line")}
          </button>
          {extraItems.length > 0 && (
            <ul className="tiny-list">
              {extraItems.map((i, idx) => (
                <li key={`${i.food_id}-${idx}`}>{i.label}</li>
              ))}
            </ul>
          )}
          <label>
            {t("admin.note")}
            <input value={note} onChange={(e) => setNote(e.target.value)} />
          </label>
          <label className="check-row">
            <input type="checkbox" checked={deduct} onChange={(e) => setDeduct(e.target.checked)} />
            {t("admin.deduct_stock")}
          </label>
          <button type="submit" className="btn-primary">
            {t("admin.register_delivery")}
          </button>
        </form>
      </section>

      <section className="panel-card">
        <h2>{t("admin.delivery_history")}</h2>
        <ul className="data-list">
          {deliveries.map((d) => (
            <li key={d.id} className="delivery-row">
              <div>
                <strong>{d.person_name}</strong>
                <div className="tiny muted">{d.delivered_on}</div>
                <div className="tiny">
                  {(d.items ?? []).map((i) => `${i.food_name} ${i.quantity}${i.food_unit ? ` ${i.food_unit}` : ""}`).join(" · ")}
                </div>
                {d.note && <div className="tiny muted">{d.note}</div>}
              </div>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
