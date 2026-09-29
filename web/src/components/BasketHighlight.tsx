import { t } from "../i18n";

type Props = { count: number };

export function BasketHighlight({ count }: Props) {
  return (
    <section className="section baskets reveal" style={{ animationDelay: "80ms" }}>
      <h2>{t("dashboard.available_baskets")}</h2>
      <p className="basket-count" aria-live="polite">
        <span className="basket-number">{count}</span>
        <span className="basket-label">{t("dashboard.baskets_unit")}</span>
      </p>
    </section>
  );
}
