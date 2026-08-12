import { useMemo } from "react";
import { useShared } from "../store/shared";

// The navigation registry is the single source of truth for every
// reachable panel. The desktop sidebar, the mobile drawer, the mobile
// bottom-nav, and the command palette all render from this one list, so
// a panel can never again be reachable on one surface but orphaned on
// another (the pre-overhaul bug: 21 panels were unreachable on phones
// and 6 DSP panels were unreachable everywhere).

export interface NavItem {
  /** Route path (e.g. "/dashboard"). For `external` items this is the
   *  URL opened in a new tab. */
  to: string;
  label: string;
  /** Decorative glyph. Always rendered `aria-hidden`; the label is the
   *  accessible name. */
  icon: string;
  /** Extra search terms for the command palette (acronyms, protocols). */
  keywords?: string[];
  /** Shown in the 4-slot mobile bottom nav. Exactly four items carry
   *  this flag. */
  primary?: boolean;
  /** Opens `to` in a new browser tab instead of routing (Config
   *  Builder is a separate SPA the daemon serves at /config/). */
  external?: boolean;
  /** Shown only when the daemon advertises the Crypto Lab console
   *  (runtime.cryptolab_console — a -tags cryptolab build). */
  requiresCryptolab?: boolean;
  /** Shown only when the daemon advertises the Signal Lab console
   *  (runtime.siglab_console — the SPA is mounted at /siglab/). */
  requiresSiglab?: boolean;
  /** Shown only when the daemon advertises the RF Scope console
   *  (runtime.rfscope_console — the SPA is mounted at /rfscope/). */
  requiresRFScope?: boolean;
  /** Shown only when the daemon serves the GopherTrunk Bundle endpoints
   *  (runtime.bundle — part of the standard daemon build). */
  requiresBundle?: boolean;
}

export interface NavGroup {
  id: string;
  label: string;
  items: NavItem[];
}

// tabKey maps a route path to the key the daemon emits in
// runtime.hidden_tabs (the path minus its leading slash). External
// items have no daemon-side tab key and are never hidden this way.
export function tabKey(item: NavItem): string {
  return item.to.replace(/^\//, "");
}

export const NAV_GROUPS: NavGroup[] = [
  {
    id: "live",
    label: "Live Ops",
    items: [
      { to: "/dashboard", label: "Dashboard", icon: "▤", primary: true, keywords: ["home", "overview", "status"] },
      { to: "/active", label: "Active", icon: "●", primary: true, keywords: ["calls", "live", "talk"] },
      { to: "/scanner", label: "Scanner", icon: "≋", primary: true, keywords: ["scan", "conventional", "trunk"] },
      { to: "/cc", label: "CC Activity", icon: "⌁", keywords: ["control channel", "grants"] },
      { to: "/history", label: "History", icon: "↺", keywords: ["calls", "archive", "past", "log"] },
    ],
  },
  {
    id: "discovery",
    label: "Discovery",
    items: [
      { to: "/hunt", label: "Hunt", icon: "🔍", primary: true, keywords: ["discover", "sweep", "spectrum", "survey"] },
      { to: "/import", label: "Import", icon: "↗", keywords: ["radioreference", "rr", "csv", "pdf"] },
    ],
  },
  {
    id: "signals",
    label: "Signals & DSP",
    items: [
      { to: "/spectrum", label: "Spectrum", icon: "≈", keywords: ["waterfall", "fft"] },
      // The Plots hub tabs the per-signal scopes (constellation, symbols, eye,
      // mixer, tuning, histogram) — they are one screen, opened in context, not
      // six separate always-listed panels. Their standalone routes still resolve
      // for deep links; their search terms are carried here so ⌘K still finds them.
      {
        to: "/plots",
        label: "Plots",
        icon: "✦",
        keywords: [
          "dsp", "charts", "constellation", "iq", "scatter", "psk",
          "symbol", "symbols", "slicer", "eye", "c4fm", "mixer", "baseband",
          "tuning", "receiver", "afc", "histogram", "levels", "quality",
        ],
      },
    ],
  },
  {
    id: "logs",
    label: "Logs",
    items: [
      { to: "/events", label: "Events", icon: "≣", keywords: ["log", "stream"] },
      { to: "/tones", label: "Tones", icon: "♪", keywords: ["tone out", "two tone", "qcii"] },
    ],
  },
  {
    id: "receivers",
    label: "Other Receivers",
    items: [
      { to: "/pagers", label: "Pagers", icon: "✉", keywords: ["pocsag", "flex", "messages"] },
      { to: "/aprs", label: "APRS", icon: "⛯", keywords: ["packet", "position", "ax25"] },
      { to: "/ais", label: "AIS", icon: "⚓", keywords: ["vessel", "marine", "ship"] },
      { to: "/dsc", label: "DSC", icon: "📡", keywords: ["marine", "distress"] },
      { to: "/adsb", label: "ADS-B", icon: "✈", keywords: ["aircraft", "mode s", "aviation"] },
      { to: "/mdc1200", label: "MDC1200", icon: "📻", keywords: ["signaling", "ptt id"] },
      { to: "/lora", label: "LoRa", icon: "🌐", keywords: ["frame", "iot"] },
    ],
  },
  {
    id: "database",
    label: "Database",
    items: [
      { to: "/systems", label: "Systems", icon: "❖", keywords: ["trunk", "sites"] },
      { to: "/talkgroups", label: "Talkgroups", icon: "☷", keywords: ["tg", "tgid", "alpha"] },
      { to: "/rids", label: "Radio IDs", icon: "⌖", keywords: ["rid", "subscriber", "alias", "unit"] },
      { to: "/bookmarks", label: "Bookmarks", icon: "★", keywords: ["frequency", "favorite", "memory"] },
    ],
  },
  {
    id: "system",
    label: "System",
    items: [
      { to: "/settings", label: "Settings", icon: "⚙", keywords: ["preferences", "theme", "config"] },
      { to: "/devices", label: "Devices", icon: "⌗", keywords: ["sdr", "pool", "rtl", "dongle"] },
      { to: "/metrics", label: "Metrics", icon: "▰", keywords: ["prometheus", "stats", "graphs"] },
      {
        to: "/bundles",
        label: "Bundles",
        icon: "📦",
        requiresBundle: true,
        keywords: ["bundle", "gtb", "case", "capture", "archive", "manifest", "verify", "sigmf"],
      },
      { to: "/config/", label: "Config Builder", icon: "🛠", external: true, keywords: ["yaml", "editor"] },
      {
        to: "/siglab/",
        label: "Signal Lab",
        icon: "◷",
        external: true,
        requiresSiglab: true,
        keywords: ["siglab", "signal", "analysis", "capture", "iq", "demod", "synthesize", "identify", "offline"],
      },
      {
        to: "/rfscope/",
        label: "RF Scope",
        icon: "⧉",
        external: true,
        requiresRFScope: true,
        keywords: ["rfscope", "rf scope", "protocol", "analyzer", "network", "capture", "offline"],
      },
      {
        to: "/cryptolab/",
        label: "Crypto Lab",
        icon: "🔐",
        external: true,
        requiresCryptolab: true,
        keywords: ["crypto", "encryption", "cipher", "decrypt", "tea1", "adp", "des", "aes", "randomness", "keystream", "assess", "recipe"],
      },
    ],
  },
];

/** Flat list of every nav item across all groups. */
export const NAV_ITEMS: NavItem[] = NAV_GROUPS.flatMap((g) => g.items);

/** The four bottom-nav primaries, in registry order. */
export const PRIMARY_ITEMS: NavItem[] = NAV_ITEMS.filter((i) => i.primary);

/**
 * Returns the nav groups with daemon-hidden tabs (web.tabs in config,
 * surfaced via /api/v1/runtime → hidden_tabs) filtered out. External
 * items are never hidden this way. Empty groups are dropped so the
 * sidebar/drawer don't render bare headers.
 */
export function useVisibleNavGroups(): NavGroup[] {
  const hiddenTabs = useShared((s) => s.hiddenTabs);
  const cryptolabConsole = useShared((s) => s.cryptolabConsole);
  const siglabConsole = useShared((s) => s.siglabConsole);
  const rfscopeConsole = useShared((s) => s.rfscopeConsole);
  const bundleEnabled = useShared((s) => s.bundleEnabled);
  return useMemo(() => {
    const hidden = new Set(hiddenTabs);
    return NAV_GROUPS.map((g) => ({
      ...g,
      items: g.items.filter((i) => {
        if (i.requiresCryptolab && !cryptolabConsole) return false;
        if (i.requiresSiglab && !siglabConsole) return false;
        if (i.requiresRFScope && !rfscopeConsole) return false;
        if (i.requiresBundle && !bundleEnabled) return false;
        return i.external || !hidden.has(tabKey(i));
      }),
    })).filter((g) => g.items.length > 0);
  }, [hiddenTabs, cryptolabConsole, siglabConsole, rfscopeConsole, bundleEnabled]);
}

/** Flat visible items (for the command palette and bottom nav). */
export function useVisibleNavItems(): NavItem[] {
  const groups = useVisibleNavGroups();
  return useMemo(() => groups.flatMap((g) => g.items), [groups]);
}
