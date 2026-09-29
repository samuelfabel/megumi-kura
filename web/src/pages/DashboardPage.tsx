import { useEffect } from "react";
import { BasketHighlight } from "../components/BasketHighlight";
import { NeedsList } from "../components/NeedsList";
import { PromiseForm } from "../components/PromiseForm";
import { PromisesList } from "../components/PromisesList";
import { CommunityNeedsList, StockList } from "../components/StockList";
import { useSummary } from "../hooks/useSummary";
import { setLocale, t } from "../i18n";
import { applyTheme } from "../theme/theme";

type Props = { onAdmin: () => void };

export function DashboardPage({ onAdmin }: Props) {
  const { data, error, loading, reload } = useSummary();

  useEffect(() => {
    if (!data) return;
    setLocale(data.site.default_locale);
    applyTheme({
      primary: data.site.primary_color,
      secondary: data.site.secondary_color,
      accent: data.site.accent_color,
    });
    document.title = data.site.name || "Megumi Kura";
  }, [data]);

  const brand = data?.site.name || t("brand.name");

  return (
    <div className="page">
      <header className="hero">
        <div className="hero-plane" aria-hidden="true" />
        <div className="hero-inner">
          <p className="brand reveal">{brand}</p>
          <p className="tagline reveal" style={{ animationDelay: "60ms" }}>
            {data?.site.description || t("dashboard.tagline_fallback")}
          </p>
          <div className="hero-actions reveal" style={{ animationDelay: "120ms" }}>
            <a className="btn-primary" href="#doar">
              {t("promise.cta")}
            </a>
          </div>
        </div>
      </header>

      <main className="content">
        {loading && <p className="muted status-line">{t("dashboard.loading")}</p>}
        {error && (
          <div className="error-box">
            <p>{t("dashboard.error")}</p>
            <button type="button" className="btn-primary" onClick={() => void reload()}>
              {t("dashboard.retry")}
            </button>
          </div>
        )}
        {data && (
          <div className="dashboard-grid">
            <div className="dash-col">
              <BasketHighlight count={data.available_baskets} />
              <StockList items={data.stock} />
              <PromisesList items={data.promises_today} />
            </div>
            <div className="dash-col">
              <PromiseForm foods={data.stock} onSubmitted={() => void reload()} />
              <CommunityNeedsList items={data.community_needs ?? []} />
              <NeedsList items={data.needs} />
            </div>
          </div>
        )}
      </main>

      <footer className="site-footer">
        <button type="button" className="footer-admin" onClick={onAdmin}>
          {t("nav.admin")}
        </button>
      </footer>
    </div>
  );
}
