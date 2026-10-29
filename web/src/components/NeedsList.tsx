import { t } from "../i18n";
import type { NeedItem, ShortageRisk } from "../services/api";

type Props = { items: NeedItem[]; risks?: ShortageRisk[] };

export function NeedsList({ items, risks = [] }: Props) {
  const list = items ?? [];
  const riskList = risks ?? [];
  if (list.length === 0 && riskList.filter((r) => r.alert).length === 0) return null;
  const riskByFood = new Map(riskList.map((r) => [r.food_id, r]));

  return (
    <section className="section reveal" style={{ animationDelay: "320ms" }}>
      <h2>{t("dashboard.needs")}</h2>
      <p className="section-lead">{t("dashboard.needs_lead")}</p>
      <ul className="needs-list">
        {list.map((item) => {
          const needed = Number(item.needed) || 0;
          const stockPct = needed > 0 ? Math.min(100, (Number(item.stock) / needed) * 100) : 0;
          const promisedPct =
            needed > 0 ? Math.min(100 - stockPct, (Number(item.promised) / needed) * 100) : 0;
          const risk = riskByFood.get(item.food_id);
          return (
            <li key={item.food_id} className="need-item">
              <div className="need-head">
                <strong>
                  {item.name}
                  {risk?.alert && (
                    <span
                      className={`risk-badge risk-${risk.severity}`}
                      title={t("dashboard.risk_hint")}
                    >
                      ⚠
                    </span>
                  )}
                </strong>
                <span>{item.percent}%</span>
              </div>
              <div
                className="need-bar stacked"
                role="progressbar"
                aria-valuenow={item.percent}
                aria-valuemin={0}
                aria-valuemax={100}
                aria-label={`${item.name}: ${t("dashboard.stock")} ${stockPct.toFixed(0)}%, ${t("dashboard.promised")} ${promisedPct.toFixed(0)}%`}
              >
                <span className="bar-stock" style={{ width: `${stockPct}%` }} />
                <span className="bar-promised" style={{ width: `${promisedPct}%` }} />
              </div>
              <p className="need-legend">
                <span className="leg-stock">{t("dashboard.stock")}</span>
                <span className="leg-promised">{t("dashboard.promised")}</span>
              </p>
              <p className="need-meta">
                {t("dashboard.needed")}: {item.needed} {item.unit} · {t("dashboard.stock")}:{" "}
                {item.stock} {item.unit} · {t("dashboard.promised")}: {item.promised} {item.unit}
              </p>
              {risk?.days_to_empty != null && (
                <p className="need-meta risk-line">
                  {t("dashboard.days_to_empty").replace("{days}", String(risk.days_to_empty))}
                </p>
              )}
            </li>
          );
        })}
      </ul>
    </section>
  );
}
