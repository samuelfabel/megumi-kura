import { useEffect } from "react";
import { BasketHighlight } from "../components/BasketHighlight";
import { NeedsList } from "../components/NeedsList";
import { ProjectionChart } from "../components/ProjectionChart";
import { PublicNav, type PublicView } from "../components/PublicNav";
import { StockList } from "../components/StockList";
import { useSummary } from "../hooks/useSummary";
import { setLocale, t } from "../i18n";
import { applyTheme } from "../theme/theme";

type Props = {
  onNavigate: (view: PublicView) => void;
  onAdmin: () => void;
};

export function StockPage({ onNavigate, onAdmin }: Props) {
  const { data, error, loading, reload } = useSummary();

  useEffect(() => {
    if (!data) return;
    setLocale(data.site.default_locale);
    applyTheme({
      primary: data.site.primary_color,
      secondary: data.site.secondary_color,
      accent: data.site.accent_color,
    });
    document.title = `${t("nav.stock")} · ${data.site.name || t("brand.name")}`;
  }, [data]);

  const brand = data?.site.name || t("brand.name");

  return (
    <div className="page">
      <PublicNav current="stock" onNavigate={onNavigate} brand={brand} />

      <header className="page-header">
        <h1>{t("nav.stock")}</h1>
        <p className="section-lead">{t("stock.page_lead")}</p>
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
          <div className="stack-sections">
            <BasketHighlight count={data.available_baskets} />
            {(data.kits?.length ?? 0) > 0 && (
              <section className="section">
                <h2>{t("dashboard.kits")}</h2>
                <ul className="kits-public">
                  {data.kits!.map((k) => (
                    <li key={k.id}>
                      <strong>{k.name}</strong>
                      <span>
                        {k.available ?? 0} {t("dashboard.kits_available")}
                      </span>
                    </li>
                  ))}
                </ul>
              </section>
            )}
            <StockList items={data.stock ?? []} />
            <NeedsList items={data.needs ?? []} risks={data.shortage_risks ?? []} />
            <ProjectionChart foods={data.stock ?? []} />
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
