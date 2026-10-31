import { afterEach, describe, expect, it } from "vitest";
import { getLocale, setLocale, supportedLocales, t } from "./index";

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

  it("translates the brand name in every locale", () => {
    for (const locale of supportedLocales) {
      setLocale(locale);
      expect(t("brand.name")).not.toBe("brand.name");
    }
  });
});
