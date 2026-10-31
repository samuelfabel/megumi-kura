import { afterEach, describe, expect, it } from "vitest";
import { getLocale, setLocale, supportedLocales, t } from "./index";
import en from "./locales/en.json";
import es from "./locales/es.json";
import ja from "./locales/ja.json";
import ptBR from "./locales/pt-BR.json";
import zh from "./locales/zh.json";

const catalogs: Record<string, Record<string, string>> = { "pt-BR": ptBR, en, es, ja, zh };
const placeholders = (s: string) => (s.match(/\{\w+\}/g) ?? []).sort();

afterEach(() => {
  setLocale("pt-BR");
});

describe("i18n", () => {
  it("defaults to pt-BR", () => {
    expect(getLocale()).toBe("pt-BR");
  });

  it("switches to a supported locale and updates the document language", () => {
    setLocale("ja");
    expect(getLocale()).toBe("ja");
    expect(document.documentElement.lang).toBe("ja");
  });

  it("ignores unsupported or empty locales", () => {
    setLocale("fr");
    setLocale(null);
    setLocale("");
    expect(getLocale()).toBe("pt-BR");
  });

  it("returns the key itself when no catalog has it", () => {
    for (const locale of supportedLocales) {
      setLocale(locale);
      expect(t("does.not.exist")).toBe("does.not.exist");
    }
  });

  it.each(Object.keys(catalogs))("%s has exactly the keys of en", (locale) => {
    const keys = Object.keys(catalogs[locale]);
    expect(Object.keys(en).filter((k) => !keys.includes(k))).toEqual([]);
    expect(keys.filter((k) => !(k in en))).toEqual([]);
  });

  it.each(Object.keys(catalogs))("%s keeps the placeholders of en", (locale) => {
    const catalog = catalogs[locale];
    const mismatched = Object.entries(en as Record<string, string>)
      .filter(([key, value]) => key in catalog && placeholders(catalog[key]).join() !== placeholders(value).join())
      .map(([key]) => key);
    expect(mismatched).toEqual([]);
  });

  it("translates the brand name in every locale", () => {
    for (const locale of supportedLocales) {
      setLocale(locale);
      expect(t("brand.name")).not.toBe("brand.name");
    }
  });
});
