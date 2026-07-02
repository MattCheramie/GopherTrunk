import { describe, it, expect, afterEach } from "vitest";
import { renderHook } from "@testing-library/react";
import {
  NAV_GROUPS,
  NAV_ITEMS,
  PRIMARY_ITEMS,
  tabKey,
  useVisibleNavGroups,
  useVisibleNavItems,
} from "./registry";
import { useShared } from "../store/shared";

// Every route mounted by App.tsx must have a registry entry, otherwise
// it becomes unreachable by navigation (the pre-overhaul bug, where six
// DSP panels were orphaned). Keep this list in sync with the <Route>
// table in App.tsx; the dynamic /plots/:tab, the "/" redirect, and the
// "*" catch-all are intentionally excluded.
const ROUTED_PATHS = [
  "/dashboard",
  "/active",
  "/scanner",
  "/hunt",
  "/spectrum",
  "/plots",
  "/constellation",
  "/symbols",
  "/eye",
  "/mixer",
  "/tuning",
  "/histogram",
  "/bookmarks",
  "/systems",
  "/talkgroups",
  "/rids",
  "/history",
  "/events",
  "/cc",
  "/tones",
  "/pagers",
  "/aprs",
  "/ais",
  "/dsc",
  "/adsb",
  "/mdc1200",
  "/lora",
  "/metrics",
  "/devices",
  "/settings",
  "/import",
];

describe("nav registry", () => {
  afterEach(() => {
    useShared.setState({ hiddenTabs: [] });
  });

  it("has a nav entry for every routed panel (no orphans)", () => {
    const registered = new Set(NAV_ITEMS.map((i) => i.to));
    for (const path of ROUTED_PATHS) {
      expect(registered, `missing nav entry for ${path}`).toContain(path);
    }
  });

  it("exposes exactly four mobile bottom-nav primaries", () => {
    expect(PRIMARY_ITEMS).toHaveLength(4);
    expect(PRIMARY_ITEMS.map((i) => i.to)).toEqual([
      "/dashboard",
      "/active",
      "/scanner",
      "/settings",
    ]);
  });

  it("has no duplicate route targets", () => {
    const seen = new Set<string>();
    for (const item of NAV_ITEMS) {
      expect(seen.has(item.to), `duplicate ${item.to}`).toBe(false);
      seen.add(item.to);
    }
  });

  it("derives the daemon tab key from the route path", () => {
    expect(tabKey({ to: "/talkgroups", label: "x", icon: "x" })).toBe(
      "talkgroups",
    );
  });

  it("includes the previously-orphaned DSP panels", () => {
    const registered = new Set(NAV_ITEMS.map((i) => i.to));
    for (const p of [
      "/constellation",
      "/symbols",
      "/eye",
      "/mixer",
      "/tuning",
      "/histogram",
    ]) {
      expect(registered).toContain(p);
    }
  });

  it("filters daemon-hidden tabs out of the visible groups", () => {
    useShared.setState({ hiddenTabs: ["talkgroups", "rids"] });
    const { result } = renderHook(() => useVisibleNavItems());
    const visible = new Set(result.current.map((i) => i.to));
    expect(visible.has("/talkgroups")).toBe(false);
    expect(visible.has("/rids")).toBe(false);
    expect(visible.has("/dashboard")).toBe(true);
  });

  it("drops groups that become empty after hiding", () => {
    // Hide both Discovery items.
    useShared.setState({ hiddenTabs: ["hunt", "import"] });
    const { result } = renderHook(() => useVisibleNavGroups());
    expect(result.current.find((g) => g.id === "discovery")).toBeUndefined();
  });

  it("never hides the external Config Builder item", () => {
    useShared.setState({ hiddenTabs: ["config/"] });
    const { result } = renderHook(() => useVisibleNavItems());
    expect(result.current.some((i) => i.external)).toBe(true);
  });

  it("gates Signal Lab / RF Scope / Crypto Lab on the runtime console flags", () => {
    // Default: none of the auxiliary consoles are advertised, so none show.
    const hidden = renderHook(() => useVisibleNavItems());
    const hiddenTargets = new Set(hidden.result.current.map((i) => i.to));
    expect(hiddenTargets.has("/siglab/")).toBe(false);
    expect(hiddenTargets.has("/rfscope/")).toBe(false);
    expect(hiddenTargets.has("/cryptolab/")).toBe(false);
    // Config Builder is unconditional, so it is always present.
    expect(hiddenTargets.has("/config/")).toBe(true);

    // Advertise all three → they appear.
    useShared.setState({
      siglabConsole: true,
      rfscopeConsole: true,
      cryptolabConsole: true,
    });
    const shown = renderHook(() => useVisibleNavItems());
    const shownTargets = new Set(shown.result.current.map((i) => i.to));
    expect(shownTargets.has("/siglab/")).toBe(true);
    expect(shownTargets.has("/rfscope/")).toBe(true);
    expect(shownTargets.has("/cryptolab/")).toBe(true);

    useShared.setState({
      siglabConsole: false,
      rfscopeConsole: false,
      cryptolabConsole: false,
    });
  });

  it("keeps every group definition non-empty by default", () => {
    for (const g of NAV_GROUPS) {
      expect(g.items.length).toBeGreaterThan(0);
    }
  });
});
