import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        bg: "rgb(var(--gt-bg) / <alpha-value>)",
        panel: "rgb(var(--gt-panel) / <alpha-value>)",
        muted: "rgb(var(--gt-muted) / <alpha-value>)",
        fg: "rgb(var(--gt-fg) / <alpha-value>)",
        accent: "rgb(var(--gt-accent) / <alpha-value>)",
        ok: "rgb(var(--gt-ok) / <alpha-value>)",
        warn: "rgb(var(--gt-warn) / <alpha-value>)",
        err: "rgb(var(--gt-err) / <alpha-value>)",
        surface: "rgb(var(--gt-surface-1) / <alpha-value>)",
        surface2: "rgb(var(--gt-surface-2) / <alpha-value>)",
        surface3: "rgb(var(--gt-surface-3) / <alpha-value>)",
        line: "rgb(var(--gt-border) / <alpha-value>)",
      },
      fontFamily: {
        mono: ["ui-monospace", "SFMono-Regular", "Menlo", "Consolas", "monospace"],
      },
    },
  },
  plugins: [],
} satisfies Config;
