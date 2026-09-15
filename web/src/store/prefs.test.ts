import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { applyTheme, prefs, themeAttr, themeColor, THEMES } from "./prefs";

// The theme axis (#1176 added "high-contrast"). These pin the two things
// that have to agree for a theme to render: the stored preference and the
// `data-theme` attribute the stylesheets key their token blocks on.
describe("prefs theme", () => {
  let meta: HTMLMetaElement;

  beforeEach(() => {
    window.localStorage.clear();
    delete document.documentElement.dataset.theme;
    meta = document.createElement("meta");
    meta.name = "theme-color";
    meta.content = "#0f172a";
    document.head.appendChild(meta);
  });

  afterEach(() => {
    meta.remove();
  });

  it("maps every theme to the attribute value its CSS block uses", () => {
    expect(themeAttr("dark")).toBe("dark");
    expect(themeAttr("monochrome")).toBe("mono");
    expect(themeAttr("light")).toBe("light");
    expect(themeAttr("high-contrast")).toBe("contrast");
  });

  it("lists high-contrast as a selectable theme", () => {
    expect(THEMES).toContain("high-contrast");
  });

  it("round-trips high-contrast through storage", () => {
    prefs.setTheme("high-contrast");
    expect(prefs.theme()).toBe("high-contrast");
    expect(window.localStorage.getItem("gt.ui.theme")).toBe("high-contrast");
  });

  it("falls back to dark on an unknown stored value", () => {
    window.localStorage.setItem("gt.ui.theme", "sepia");
    expect(prefs.theme()).toBe("dark");
  });

  it("applyTheme stamps data-theme and the theme-color meta", () => {
    applyTheme("high-contrast");
    expect(document.documentElement.dataset.theme).toBe("contrast");
    expect(meta.content).toBe(themeColor("high-contrast"));
    expect(meta.content).toBe("#ffffff");

    applyTheme("dark");
    expect(document.documentElement.dataset.theme).toBe("dark");
    expect(meta.content).toBe("#0f172a");
  });
});
