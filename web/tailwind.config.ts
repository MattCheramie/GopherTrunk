import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        // Slate-leaning palette that mirrors the TUI's monochrome
        // theme. CSS custom properties drive the live theme so the
        // Settings panel can swap dark / monochrome without a
        // re-render.
        bg: "rgb(var(--gt-bg) / <alpha-value>)",
        panel: "rgb(var(--gt-panel) / <alpha-value>)",
        muted: "rgb(var(--gt-muted) / <alpha-value>)",
        fg: "rgb(var(--gt-fg) / <alpha-value>)",
        accent: "rgb(var(--gt-accent) / <alpha-value>)",
        ok: "rgb(var(--gt-ok) / <alpha-value>)",
        warn: "rgb(var(--gt-warn) / <alpha-value>)",
        err: "rgb(var(--gt-err) / <alpha-value>)",
        // Surface elevation + hairline tokens (Phase 3 token refresh).
        // In dark/mono these map onto the existing palette so nothing
        // shifts; the light theme and new UI primitives rely on them.
        surface: "rgb(var(--gt-surface-1) / <alpha-value>)",
        surface2: "rgb(var(--gt-surface-2) / <alpha-value>)",
        surface3: "rgb(var(--gt-surface-3) / <alpha-value>)",
        line: "rgb(var(--gt-border) / <alpha-value>)",
      },
      fontFamily: {
        mono: [
          "ui-monospace",
          "SFMono-Regular",
          "Menlo",
          "Consolas",
          "monospace",
        ],
      },
      // Safe-area insets so the bottom nav clears the home indicator
      // on iOS, and the top bar clears the notch.
      spacing: {
        "safe-top": "env(safe-area-inset-top)",
        "safe-bottom": "env(safe-area-inset-bottom)",
        "safe-left": "env(safe-area-inset-left)",
        "safe-right": "env(safe-area-inset-right)",
      },
    },
  },
  plugins: [],
} satisfies Config;
