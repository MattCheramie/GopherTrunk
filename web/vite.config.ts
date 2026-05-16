/// <reference types="vitest" />
import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import { VitePWA } from "vite-plugin-pwa";

// Vite bundles every dependency listed in package.json into the
// shipped `dist/`. The output has no runtime Node dependency and
// no CDN fetches — everything required to run the SPA is in the
// emitted HTML/JS/CSS. The base path defaults to "./" so the
// `dist/` folder works equally well when opened via file://, hosted
// at a static path, or rooted at "/".
export default defineConfig({
  base: "./",
  plugins: [
    react(),
    VitePWA({
      registerType: "autoUpdate",
      includeAssets: ["favicon.svg"],
      manifest: {
        name: "GopherTrunk",
        short_name: "GopherTrunk",
        description:
          "Operator console for the GopherTrunk trunked-radio daemon — runs entirely in your browser, points at any daemon on the network.",
        theme_color: "#0f172a",
        background_color: "#0f172a",
        display: "standalone",
        orientation: "any",
        start_url: "./",
        scope: "./",
        // SVG icons cover Chrome, Firefox, Edge, and Android. Apple
        // platforms fall back to apple-touch-icon (a PNG referenced
        // from index.html); a follow-up PR adds raster fallbacks for
        // older Safari builds.
        icons: [
          {
            src: "favicon.svg",
            sizes: "any",
            type: "image/svg+xml",
            purpose: "any",
          },
          {
            src: "favicon.svg",
            sizes: "any",
            type: "image/svg+xml",
            purpose: "maskable",
          },
        ],
      },
      workbox: {
        // Pre-cache the SPA bundle so the app installs offline-first
        // (the connect screen still needs a live daemon to do
        // anything useful, but the UI itself loads instantly and
        // works without a network round-trip for the assets).
        globPatterns: ["**/*.{js,css,html,svg,png,ico,webmanifest}"],
        // API responses are always fetched live; the SW never caches
        // /api/* or /metrics responses.
        navigateFallbackDenylist: [/^\/api\//, /^\/metrics/],
      },
      devOptions: { enabled: false },
    }),
  ],
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://127.0.0.1:8080",
        changeOrigin: true,
        ws: true,
      },
      "/metrics": {
        target: "http://127.0.0.1:8080",
        changeOrigin: true,
      },
    },
  },
  build: {
    target: "es2020",
    sourcemap: false,
    // Single chunked bundle keeps the install size small and the
    // pre-cache list short. Chart.js + D3 are the largest deps and
    // ship in the main bundle; splitting them only matters when
    // the SPA grows multiple unrelated entry points.
  },
  test: {
    // jsdom mirrors enough of the browser API surface for React
    // Testing Library + the SPA's runtime. Each test gets its own
    // isolated DOM. setupFiles wires in @testing-library/jest-dom's
    // assertion vocabulary so `expect(node).toBeInTheDocument()`
    // works.
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    // Limit the test workspace to the SPA — the embedded SW + the
    // workbox precache list contain non-test JS we don't want vitest
    // collecting.
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
  },
});
