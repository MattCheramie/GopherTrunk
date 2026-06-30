import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The RF Scope SPA bundles everything into dist/ (no CDN) and is embedded into
// the gophertrunk binary. In dev, /api proxies to the default `rfscope serve`
// listen address.
export default defineConfig({
  base: "./",
  plugins: [react()],
  server: {
    port: 5272,
    proxy: {
      "/api": { target: "http://127.0.0.1:8098", changeOrigin: true },
    },
  },
  build: {
    target: "es2020",
    sourcemap: false,
    chunkSizeWarningLimit: 4096,
  },
});
