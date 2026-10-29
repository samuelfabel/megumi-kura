import { useEffect, useState } from "react";
import { t } from "../i18n";
import { fetchProjection, type ProjectionSeries, type StockItem } from "../services/api";

type Props = { foods: StockItem[] };

export function ProjectionChart({ foods }: Props) {
  const [foodId, setFoodId] = useState(0);
  const [series, setSeries] = useState<ProjectionSeries | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (foods[0] && foodId === 0) setFoodId(foods[0].food_id);
  }, [foods, foodId]);

  useEffect(() => {
    if (!foodId) return;
    let cancelled = false;
    void fetchProjection(foodId, 30)
      .then((s) => {
        if (!cancelled) {
          setSeries(s);
          setError(null);
        }
      })
      .catch(() => {
        if (!cancelled) setError(t("dashboard.chart_error"));
      });
    return () => {
      cancelled = true;
    };
  }, [foodId]);

  if (foods.length === 0) return null;

  const points = series?.points ?? [];
  const values = points.map((p) => Number(p.balance) || 0);
  const max = Math.max(1, ...values);
  const min = Math.min(0, ...values);
  const span = Math.max(1, max - min);
  const w = 560;
  const h = 160;
  const pad = 12;
  const path = points
    .map((p, i) => {
      const x = pad + (i / Math.max(1, points.length - 1)) * (w - pad * 2);
      const y = h - pad - ((Number(p.balance) - min) / span) * (h - pad * 2);
      return `${i === 0 ? "M" : "L"}${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(" ");

  return (
    <section className="section reveal" style={{ animationDelay: "280ms" }}>
      <h2>{t("dashboard.projection")}</h2>
      <p className="section-lead">{t("dashboard.projection_lead")}</p>
      <label className="chart-food">
        <span>{t("promise.food")}</span>
        <select value={foodId} onChange={(e) => setFoodId(Number(e.target.value))}>
          {foods.map((f) => (
            <option key={f.food_id} value={f.food_id}>
              {f.name}
            </option>
          ))}
        </select>
      </label>
      {error && <p className="error-text">{error}</p>}
      {points.length > 0 && (
        <svg
          className="projection-svg"
          viewBox={`0 0 ${w} ${h}`}
          role="img"
          aria-label={t("dashboard.projection")}
        >
          <path d={path} fill="none" stroke="var(--color-primary)" strokeWidth="2.5" />
          {points.map((p, i) => {
            const x = pad + (i / Math.max(1, points.length - 1)) * (w - pad * 2);
            const y = h - pad - ((Number(p.balance) - min) / span) * (h - pad * 2);
            return <circle key={p.day} cx={x} cy={y} r="2.5" fill="var(--color-accent)" />;
          })}
        </svg>
      )}
      {series && (
        <p className="muted chart-meta">
          {series.name} ({series.unit}) · {points.length} {t("dashboard.days")}
        </p>
      )}
    </section>
  );
}
