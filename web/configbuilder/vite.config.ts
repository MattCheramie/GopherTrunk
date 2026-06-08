/// <reference types="vitest" />
import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import { VitePWA } from "vite-plugin-pwa";

const setupPath = fileURLToPath(new URL("./src/test/setup.ts", import.meta.url));

// The Config Builder SPA bundles everything into dist/ (no CDN) so the
// single-binary embed serves it offline. base: "./" makes the bundle work
// whether served at / (standalone `config serve`) or /config/ (daemon).
export default defineConfig({
  base: "./",
  plugins: [
    react(),
    VitePWA({
      registerType: "autoUpdate",
      includeAssets: ["favicon.svg"],
      manifest: {
        name: "GopherTrunk Config Builder",
        short_name: "ConfigBuilder",
        description:
          "Browse, build, validate and save GopherTrunk config files in your browser.",
        theme_color: "#0f172a",
        background_color: "#0f172a",
        display: "standalone",
        start_url: "./",
        scope: "./",
        icons: [
          { src: "favicon.svg", sizes: "any", type: "image/svg+xml", purpose: "any" },
          { src: "favicon.svg", sizes: "any", type: "image/svg+xml", purpose: "maskable" },
        ],
      },
      workbox: {
        globPatterns: ["**/*.{js,css,html,svg,png,ico,webmanifest}"],
        navigateFallbackDenylist: [/^\/api\//],
      },
      devOptions: { enabled: false },
    }),
  ],
  server: {
    port: 5274,
    proxy: {
      "/api": { target: "http://127.0.0.1:8077", changeOrigin: true },
    },
  },
  build: {
    target: "es2020",
    sourcemap: false,
    chunkSizeWarningLimit: 2048,
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: [setupPath],
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
  },
});
