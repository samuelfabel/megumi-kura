import { useEffect, useMemo, useState } from "react";
import { t } from "../../i18n";
import {
  createPerson,
  createPersonNeed,
  deletePersonNeed,
  type Food,
  type Person,
  type PersonNeed,
} from "../../services/api";

export function PeoplePanel({
  people,
  foods,
  needs,
  onRefresh,
  onError,
}: {
  people: Person[];
  foods: Food[];
  needs: PersonNeed[];
  onRefresh: () => void;
  onError: (s: string) => void;
}) {
  const [name, setName] = useState("");
  const [notes, setNotes] = useState("");
  const [personId, setPersonId] = useState<number>(0);
  const [foodId, setFoodId] = useState<number>(0);
  const [qty, setQty] = useState("1");
  const [kind, setKind] = useState<"sporadic" | "recurring">("recurring");
  const [freq, setFreq] = useState("monthly");

  useEffect(() => {
    if (people[0] && !personId) setPersonId(people[0].id);
    if (foods[0] && !foodId) setFoodId(foods[0].id);
  }, [people, foods, personId, foodId]);

  const filtered = useMemo(
    () => (personId ? needs.filter((n) => n.person_id === personId) : needs),
    [needs, personId],
  );

  return (
    <div className="panel-grid">
      <section className="panel-card">
        <h2>{t("admin.people_title")}</h2>
        <form
          className="stack-form"
          onSubmit={(e) => {
            e.preventDefault();
            void createPerson({ full_name: name, notes })
              .then(() => {
                setName("");
                setNotes("");
                onRefresh();
              })
              .catch((err: Error) => onError(err.message));
          }}
        >
          <label>
            {t("admin.person_name")}
            <input value={name} onChange={(e) => setName(e.target.value)} required />
          </label>
          <label>
            {t("admin.person_notes")}
            <input value={notes} onChange={(e) => setNotes(e.target.value)} />
          </label>
          <button type="submit" className="btn-primary">
            {t("admin.add_person")}
          </button>
        </form>
        <ul className="simple-list">
          {people.map((p) => (
            <li key={p.id}>
              <button type="button" className={personId === p.id ? "chip active" : "chip"} onClick={() => setPersonId(p.id)}>
                {p.full_name}
              </button>
              {p.notes && <span className="muted tiny">{p.notes}</span>}
            </li>
          ))}
        </ul>
      </section>

      <section className="panel-card">
        <h2>{t("admin.person_needs_title")}</h2>
        <form
          className="stack-form"
          onSubmit={(e) => {
            e.preventDefault();
            void createPersonNeed({
              person_id: personId,
              food_id: foodId,
              quantity: qty,
              need_kind: kind,
              frequency: kind === "recurring" ? freq : null,
            })
              .then(onRefresh)
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
            {t("admin.item")}
            <select value={foodId} onChange={(e) => setFoodId(Number(e.target.value))}>
              {foods.map((f) => (
                <option key={f.id} value={f.id}>
                  {f.name} ({f.unit}){f.category_name ? ` · ${f.category_name}` : ""}
                </option>
              ))}
            </select>
          </label>
          <div className="promise-row">
            <label>
              {t("admin.quantity")}
              <input value={qty} onChange={(e) => setQty(e.target.value)} required />
            </label>
            <label>
              {t("admin.need_kind")}
              <select value={kind} onChange={(e) => setKind(e.target.value as "sporadic" | "recurring")}>
                <option value="recurring">{t("admin.kind_recurring")}</option>
                <option value="sporadic">{t("admin.kind_sporadic")}</option>
              </select>
            </label>
          </div>
          {kind === "recurring" && (
            <label>
              {t("admin.frequency")}
              <select value={freq} onChange={(e) => setFreq(e.target.value)}>
                <option value="weekly">{t("admin.freq_weekly")}</option>
                <option value="biweekly">{t("admin.freq_biweekly")}</option>
                <option value="monthly">{t("admin.freq_monthly")}</option>
              </select>
            </label>
          )}
          <button type="submit" className="btn-primary">
            {t("admin.add_need")}
          </button>
        </form>

        <ul className="data-list">
          {filtered.map((n) => (
            <li key={n.id}>
              <div>
                <strong>{n.food_name}</strong>{" "}
                <span className="muted">
                  {n.quantity} {n.food_unit}
                </span>
                <div className="tiny muted">
                  {n.category_name} · {n.need_kind === "sporadic" ? t("admin.kind_sporadic") : `${t("admin.kind_recurring")} / ${n.frequency}`}
                </div>
              </div>
              <button
                type="button"
                className="btn-ghost"
                onClick={() => void deletePersonNeed(n.id).then(onRefresh).catch((err: Error) => onError(err.message))}
              >
                {t("admin.remove")}
              </button>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
