import { t } from "../i18n";

export type PublicView = "home" | "stock" | "promise";

type Props = {
  current: PublicView;
  onNavigate: (view: PublicView) => void;
  brand: string;
};

export function PublicNav({ current, onNavigate, brand }: Props) {
  const items: Array<{ id: PublicView; label: string; hash: string }> = [
    { id: "home", label: t("nav.home"), hash: "" },
    { id: "stock", label: t("nav.stock"), hash: "estoque" },
    { id: "promise", label: t("nav.promise"), hash: "doar" },
  ];

  return (
    <nav className="public-nav" aria-label={t("nav.public")}>
      <button
        type="button"
        className="public-nav-brand"
        onClick={() => onNavigate("home")}
      >
        {brand}
      </button>
      <ul className="public-nav-list">
        {items.map((item) => (
          <li key={item.id}>
            <a
              href={item.hash ? `#${item.hash}` : "#"}
              className={current === item.id ? "public-nav-link active" : "public-nav-link"}
              aria-current={current === item.id ? "page" : undefined}
              onClick={(e) => {
                e.preventDefault();
                window.location.hash = item.hash;
                onNavigate(item.id);
              }}
            >
              {item.label}
            </a>
          </li>
        ))}
      </ul>
    </nav>
  );
}
