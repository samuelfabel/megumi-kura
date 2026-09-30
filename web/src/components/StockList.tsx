import { useMemo } from "react";
import { t } from "../i18n";
import type { CommunityNeed, StockItem } from "../services/api";

type Props = { items: StockItem[] };

export function StockList({ items }: Props) {
  const groups = useMemo(() => {
    const map = new Map<string, StockItem[]>();
    for (const item of items) {
      const key = item.category_name || t("admin.uncategorized");
      const list = map.get(key) ?? [];
      list.push(item);
      map.set(key, list);
    }
    return [...map.entries()];
  }, [items]);

  return (
    <section className="section reveal" style={{ animationDelay: "120ms" }}>
      <h2>{t("dashboard.current_stock")}</h2>
      <div className="stock-groups">
        {groups.map(([category, list]) => (
          <div key={category} className="stock-group">
            <h3 className="group-title">{category}</h3>
            <ul className="metric-list compact">
              {list.map((item, index) => (
                <li
                  key={item.food_id}
                  className="metric-row"
                  style={{ animationDelay: `${140 + index * 20}ms` }}
                >
                  <span className="metric-name">{item.name}</span>
                  <span className="metric-value">
                    <strong>{item.quantity}</strong>
                    <span className="metric-unit">{item.unit}</span>
                  </span>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>
    </section>
  );
}

type NeedsProps = { items: CommunityNeed[] };

export function CommunityNeedsList({ items }: NeedsProps) {
  if (!items.length) return null;
  const groups = new Map<string, CommunityNeed[]>();
  for (const item of items) {
    const key = item.category_name || t("admin.uncategorized");
    const list = groups.get(key) ?? [];
    list.push(item);
    groups.set(key, list);
  }

  return (
    <section className="section reveal" style={{ animationDelay: "280ms" }}>
      <h2>{t("dashboard.community_needs")}</h2>
      <p className="section-lead">{t("dashboard.community_needs_lead")}</p>
      <div className="stock-groups">
        {[...groups.entries()].map(([category, list]) => (
          <div key={category} className="stock-group">
            <h3 className="group-title">{category}</h3>
            <ul className="metric-list compact">
              {list.map((n) => (
                <li key={n.id} className="metric-row">
                  <span className="metric-name">
                    {n.title || n.food_name}
                    <span className="need-tag">
                      {n.need_kind === "sporadic"
                        ? t("admin.kind_sporadic")
                        : `${t("admin.kind_recurring")} · ${n.frequency}`}
                    </span>
                  </span>
                  <span className="metric-value">
                    <strong>{n.quantity}</strong>
                    <span className="metric-unit">{n.food_unit}</span>
                  </span>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>
    </section>
  );
}
