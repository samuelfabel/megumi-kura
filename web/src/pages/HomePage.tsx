import { useEffect } from "react";
import { MailSubscribe } from "../components/MailSocial";
import { PublicNav, type PublicView } from "../components/PublicNav";
import { SocialFeed } from "../components/SocialFeed";
import { CommunityNeedsList } from "../components/StockList";
import { useSummary } from "../hooks/useSummary";
import { setLocale, t } from "../i18n";
import { applyTheme } from "../theme/theme";

type Props = {
  onNavigate: (view: PublicView) => void;
  onAdmin: () => void;
};

export function HomePage({ onNavigate, onAdmin }: Props) {
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
      <PublicNav current="home" onNavigate={onNavigate} brand={brand} />

      <header className="hero home-hero">
        <div className="hero-plane" aria-hidden="true" />
        <div className="hero-inner">
          <p className="brand reveal">{brand}</p>
          <p className="tagline reveal" style={{ animationDelay: "60ms" }}>
            {data?.site.description || t("dashboard.tagline_fallback")}
          </p>
          <div className="hero-actions reveal" style={{ animationDelay: "120ms" }}>
            <button type="button" className="btn-primary" onClick={() => onNavigate("promise")}>
              {t("promise.cta")}
            </button>
            <button type="button" className="btn-hero-ghost" onClick={() => onNavigate("stock")}>
              {t("nav.stock")}
            </button>
          </div>
        </div>
      </header>

      <aside className="home-banner reveal" style={{ animationDelay: "80ms" }} aria-label={t("home.banner_label")}>
        <p className="home-banner-kicker">{t("home.banner_kicker")}</p>
        <p className="home-banner-text">{t("home.banner_text")}</p>
      </aside>

      <main className="content home-content">
        {loading && <p className="muted status-line">{t("dashboard.loading")}</p>}
        {error && (
          <div className="error-box">
            <p>{t("dashboard.error")}</p>
            <button type="button" className="btn-primary" onClick={() => void reload()}>
              {t("dashboard.retry")}
            </button>
          </div>
        )}

        {data && <CommunityNeedsList items={data.community_needs ?? []} />}

        <SocialFeed />

        <MailSubscribe />
      </main>

      <footer className="site-footer">
        <button type="button" className="footer-admin" onClick={onAdmin}>
          {t("nav.admin")}
        </button>
      </footer>
    </div>
  );
}
