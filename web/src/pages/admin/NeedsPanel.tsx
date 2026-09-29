import { useEffect, useState } from "react";
import { t } from "../../i18n";
import {
  createCommunityNeed,
  deleteCommunityNeed,
  type CommunityNeed,
  type Food,
} from "../../services/api";

export function NeedsPanel({
  foods,
  needs,
  onRefresh,
  onError,
}: {
  foods: Food[];
  needs: CommunityNeed[];
  onRefresh: () => void;
  onError: (s: string) => void;
}) {
  const [foodId, setFoodId] = useState(0);
  const [title, setTitle] = useState("");
  const [qty, setQty] = useState("10");
  const [kind, setKind] = useState<"sporadic" | "recurring">("sporadic");
  const [freq, setFreq] = useState("monthly");

  useEffect(() => {
    if (foods[0] && !foodId) setFoodId(foods[0].id);
  }, [foods, foodId]);

  return (
    <div className="panel-grid">
      <section className="panel-card">
        <h2>{t("admin.community_needs_title")}</h2>
        <p className="section-lead">{t("admin.community_needs_lead")}</p>
        <form
          className="stack-form"
          onSubmit={(e) => {
            e.preventDefault();
            void createCommunityNeed({
              food_id: foodId,
              title,
              quantity: qty,
              need_kind: kind,
              frequency: kind === "recurring" ? freq : null,
            })
              .then(() => {
                setTitle("");
                onRefresh();
              })
              .catch((err: Error) => onError(err.message));
          }}
        >
          <label>
            {t("admin.item")}
            <select value={foodId} onChange={(e) => setFoodId(Number(e.target.value))}>
              {foods.map((f) => (
                <option key={f.id} value={f.id}>
                  {f.name}{f.category_name ? ` · ${f.category_name}` : ""}
                </option>
              ))}
            </select>
          </label>
          <label>
            {t("admin.need_title")}
            <input value={title} onChange={(e) => setTitle(e.target.value)} />
          </label>
          <div className="promise-row">
            <label>
              {t("admin.quantity")}
              <input value={qty} onChange={(e) => setQty(e.target.value)} />
            </label>
            <label>
              {t("admin.need_kind")}
              <select value={kind} onChange={(e) => setKind(e.target.value as "sporadic" | "recurring")}>
                <option value="sporadic">{t("admin.kind_sporadic")}</option>
                <option value="recurring">{t("admin.kind_recurring")}</option>
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
      </section>
      <section className="panel-card">
        <h2>{t("admin.needs_list")}</h2>
        <ul className="data-list">
          {needs.map((n) => (
            <li key={n.id}>
              <div>
                <strong>{n.title || n.food_name}</strong>
                <div className="tiny muted">
                  {n.category_name} · {n.food_name} {n.quantity} {n.food_unit} ·{" "}
                  {n.need_kind === "sporadic" ? t("admin.kind_sporadic") : `${t("admin.kind_recurring")} / ${n.frequency}`}
                </div>
              </div>
              <button
                type="button"
                className="btn-ghost"
                onClick={() => void deleteCommunityNeed(n.id).then(onRefresh).catch((err: Error) => onError(err.message))}
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
