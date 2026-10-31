import { useEffect } from "react";
import { PromiseForm } from "../components/PromiseForm";
import { PromisesList } from "../components/PromisesList";
import { PublicNav, type PublicView } from "../components/PublicNav";
import { useSummary } from "../hooks/useSummary";
import { setLocale, t } from "../i18n";
import { applyTheme } from "../theme/theme";

type Props = {
  onNavigate: (view: PublicView) => void;
  onAdmin: () => void;
};

export function PromisePage({ onNavigate, onAdmin }: Props) {
  const { data, error, loading, reload } = useSummary();

  useEffect(() => {
    if (!data) return;
    setLocale(data.site.default_locale);
    applyTheme({
      primary: data.site.primary_color,
      secondary: data.site.secondary_color,
      accent: data.site.accent_color,
    });
    document.title = `${t("nav.promise")} · ${data.site.name || t("brand.name")}`;
  }, [data]);

  const brand = data?.site.name || t("brand.name");

  return (
    <div className="page">
      <PublicNav current="promise" onNavigate={onNavigate} brand={brand} />

      <header className="page-header">
        <h1>{t("promise.title")}</h1>
        <p className="section-lead">{t("promise.lead")}</p>
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
          <div className="stack-sections promise-layout">
            <PromiseForm
              foods={data.stock ?? []}
              kits={data.kits ?? []}
              onSubmitted={() => void reload()}
            />
            <PromisesList items={data.promises_today ?? []} />
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
