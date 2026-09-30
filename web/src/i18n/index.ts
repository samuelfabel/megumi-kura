import en from "./locales/en.json";
import es from "./locales/es.json";
import ja from "./locales/ja.json";
import ptBR from "./locales/pt-BR.json";
import zh from "./locales/zh.json";

export type Locale = "pt-BR" | "en" | "es" | "zh" | "ja";

const catalogs: Record<Locale, Record<string, string>> = {
  "pt-BR": ptBR,
  en,
  es,
  zh,
  ja,
};

let current: Locale = "pt-BR";

export function setLocale(locale: string | undefined | null) {
  if (locale && locale in catalogs) {
    current = locale as Locale;
    document.documentElement.lang = current;
  }
}

export function getLocale(): Locale {
  return current;
}

export function t(key: string): string {
  return catalogs[current][key] ?? catalogs.en[key] ?? key;
}

export const supportedLocales: Locale[] = ["pt-BR", "en", "es", "zh", "ja"];
