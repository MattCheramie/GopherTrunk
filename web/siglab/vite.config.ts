/// <reference types="vitest" />
import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import { VitePWA } from "vite-plugin-pwa";

const setupPath = fileURLToPath(new URL("./src/test/setup.ts", import.meta.url));

// The Signal Lab SPA bundles every dependency into dist/ (no CDN). The
// heavy visualization libraries (Plotly, TensorFlow.js, stdlib, Observable
// Plot) are split into their own chunks via manualChunks and loaded on
// demand by the panels that use them, so the initial download stays small
// while the offline single-binary embed still ships everything.
export default defineConfig({
  base: "./",
  plugins: [
    react(),
    VitePWA({
      registerType: "autoUpdate",
      includeAssets: ["favicon.svg"],
      manifest: {
        name: "GopherTrunk Signal Lab",
        short_name: "SignalLab",
        description:
          "Offline signal-analysis console for GopherTrunk — analyze, visualize and compare RF captures in your browser.",
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
        // Bump the precache size ceiling — the tfjs/plotly chunks are large.
        maximumFileSizeToCacheInBytes: 8 * 1024 * 1024,
        navigateFallbackDenylist: [/^\/api\//],
      },
      devOptions: { enabled: false },
    }),
  ],
  server: {
    port: 5273,
    proxy: {
      "/api": { target: "http://127.0.0.1:8099", changeOrigin: true },
    },
  },
  build: {
    target: "es2020",
    sourcemap: false,
    chunkSizeWarningLimit: 4096,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes("node_modules")) {
            if (id.includes("@tensorflow")) return "tfjs";
            if (id.includes("plotly")) return "plotly";
            if (id.includes("@stdlib")) return "stdlib";
            if (id.includes("@observablehq/plot") || id.includes("d3"))
              return "d3plot";
          }
          return undefined;
        },
      },
    },
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: [setupPath],
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
  },
});
