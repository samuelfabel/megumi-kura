import { t } from "../i18n";
import type { NeedItem } from "../services/api";

type Props = { items: NeedItem[] };

export function NeedsList({ items }: Props) {
  if (items.length === 0) return null;
  return (
    <section className="section reveal" style={{ animationDelay: "320ms" }}>
      <h2>{t("dashboard.needs")}</h2>
      <ul className="needs-list">
        {items.map((item) => (
          <li key={item.food_id} className="need-item">
            <div className="need-head">
              <strong>{item.name}</strong>
              <span>{item.percent}%</span>
            </div>
            <div
              className="need-bar"
              role="progressbar"
              aria-valuenow={item.percent}
              aria-valuemin={0}
              aria-valuemax={100}
            >
              <span style={{ width: `${Math.min(item.percent, 100)}%` }} />
            </div>
            <p className="need-meta">
              {t("dashboard.needed")}: {item.needed} {item.unit} · {t("dashboard.stock")}:{" "}
              {item.stock} {item.unit} · {t("dashboard.promised")}: {item.promised} {item.unit}
            </p>
            <p className="need-meta">
              {t("dashboard.total")}: {item.total} {item.unit} · {t("dashboard.missing")}:{" "}
              {item.missing} {item.unit}
            </p>
          </li>
        ))}
      </ul>
    </section>
  );
}
