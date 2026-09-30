import { t } from "../i18n";
import type { PromiseAgg } from "../services/api";

type Props = { items: PromiseAgg[] };

export function PromisesList({ items }: Props) {
  return (
    <section className="section reveal" style={{ animationDelay: "160ms" }}>
      <h2>{t("dashboard.promises_today")}</h2>
      <p className="section-lead">{t("dashboard.promises_lead")}</p>
      {items.length === 0 ? (
        <p className="muted">{t("dashboard.empty_promises")}</p>
      ) : (
        <ul className="metric-list compact">
          {items.map((item, index) => (
            <li
              key={item.food_id}
              className="metric-row"
              style={{ animationDelay: `${180 + index * 30}ms` }}
            >
              <span className="metric-name">{item.name}</span>
              <span className="metric-value">
                <strong>{item.quantity}</strong>
                <span className="metric-unit">{item.unit}</span>
              </span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
